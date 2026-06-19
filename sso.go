package sfm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type gigyaAuthResponse struct {
	CookieValue string `json:"cookieValue"`
}

type cdcBaseResponse struct {
	ErrorCode    int    `json:"errorCode"`
	StatusCode   int    `json:"statusCode"`
	ErrorMessage string `json:"errorMessage"`
}

type cdcNotifyLoginResponse struct {
	cdcBaseResponse
	LoginToken string `json:"login_token"`
}

type cdcJWTResponse struct {
	cdcBaseResponse
	IDToken string `json:"id_token"`
}

type cdcAccountInfoResponse struct {
	cdcBaseResponse
	UID string `json:"UID"`
}

type uidDetailsResponse struct {
	Accounts map[string]uidDetailsAccount `json:"accounts"`
}

type uidDetailsAccount struct {
	LinkedAccounts []json.RawMessage `json:"linkedAccounts"`
}

type SSO struct {
	client            *core
	sdkBuildNumber    string
	getEndpointMetaFn func(context.Context, string, requestOptions) (string, map[string]string, error)
}

type authFlowContext struct {
	endpoint string
	meta     map[string]string
}

const maxSSOHops = 20

var sidPattern = regexp.MustCompile(`^[sS]\d+$`)

func (s *SSO) Login(ctx context.Context) error {
	if !sidPattern.MatchString(s.client.username) {
		return &Error{Kind: ErrInvalidSID, Msg: "please use SID like S1234567890"}
	}

	flow := &authFlowContext{endpoint: URLLaunchpad, meta: map[string]string{}}

	if err := s.stepTraverseSAML(ctx, flow); err != nil {
		return fmt.Errorf("sso step traverse_saml: %w", err)
	}
	if strings.Contains(flow.endpoint, "authn") {
		if err := s.stepHandleAuthn(ctx, flow); err != nil {
			return fmt.Errorf("sso step authn: %w", err)
		}
	}
	if strings.Contains(flow.endpoint, "gigya") {
		if err := s.stepHandleGigya(ctx, flow); err != nil {
			return fmt.Errorf("sso step gigya: %w", err)
		}
	}

	return nil
}

func (s *SSO) stepTraverseSAML(ctx context.Context, flow *authFlowContext) error {
	for hop := 0; hop < maxSSOHops; hop++ {
		if hasSSOTerminalMeta(flow.meta) {
			s.logStep(ctx, "traverse_saml_done", flow.endpoint, hop+1)
			return nil
		}

		nextEndpoint, nextMeta, err := s.nextSSOEndpointMeta(ctx, flow.endpoint, requestOptions{Form: flow.meta})
		if err != nil {
			return err
		}
		flow.endpoint = nextEndpoint
		flow.meta = nextMeta
		if _, ok := flow.meta["j_username"]; ok {
			flow.meta["j_username"] = s.client.username
			flow.meta["j_password"] = s.client.password
		}
		s.logStep(ctx, "traverse_saml_hop", flow.endpoint, hop+1)
	}

	return &Error{Kind: ErrClient, Msg: fmt.Sprintf("sso hop limit reached (%d)", maxSSOHops)}
}

func (s *SSO) stepHandleAuthn(ctx context.Context, flow *authFlowContext) error {
	supportEndpoint, supportMeta, err := s.nextSSOEndpointMeta(ctx, flow.endpoint, requestOptions{Form: flow.meta})
	if err != nil {
		return err
	}
	_, _, err = s.client.request(ctx, http.MethodPost, supportEndpoint, requestOptions{Form: supportMeta})
	if err != nil {
		return err
	}
	s.logStep(ctx, "authn_submitted", supportEndpoint, 1)
	return nil
}

func (s *SSO) stepHandleGigya(ctx context.Context, flow *authFlowContext) error {
	params, err := s.getGigyaLoginParams(ctx, flow.endpoint, flow.meta)
	if err != nil {
		return err
	}
	if err := s.gigyaWebSDKBootstrap(ctx, params); err != nil {
		return err
	}
	authCode, err := s.getGigyaAuthCode(ctx, s.client.username, s.client.password)
	if err != nil {
		return err
	}
	loginToken, err := s.getGigyaLoginToken(ctx, params, authCode)
	if err != nil {
		return err
	}
	uid, err := s.getUID(ctx, params, loginToken)
	if err != nil {
		return err
	}
	idToken, err := s.getIDToken(ctx, params, loginToken)
	if err != nil {
		return err
	}
	details, err := s.getUIDDetails(ctx, uid, idToken)
	if err != nil {
		return err
	}
	if isUIDLinkedMultipleSIDs(details) {
		if err := s.selectAccount(ctx, uid, s.client.username, idToken); err != nil {
			return err
		}
	}

	apiKey := params["apiKey"]
	samlContext := params["samlContext"]
	if apiKey == "" || samlContext == "" {
		return &Error{Kind: ErrParse, Msg: "gigya params missing apiKey or samlContext"}
	}

	idpEndpoint := strings.ReplaceAll(URLAccountSSOIDP, "{k}", apiKey)
	contextForm := map[string]string{"loginToken": loginToken, "samlContext": samlContext}
	allowRedirect := false
	nextEndpoint, nextMeta, err := s.nextSSOEndpointMeta(ctx, idpEndpoint, requestOptions{Params: contextForm, AllowRedirects: &allowRedirect})
	if err != nil {
		return err
	}

	gigyaHeaders := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*"}
	for hop := 0; hop < maxSSOHops; hop++ {
		if nextEndpoint == URLLaunchpad+"/" {
			_, _, err = s.client.request(ctx, http.MethodPost, nextEndpoint, requestOptions{Form: nextMeta, Headers: gigyaHeaders})
			if err != nil {
				return err
			}
			s.logStep(ctx, "gigya_finalize", nextEndpoint, hop+1)
			return nil
		}

		nextEndpoint, nextMeta, err = s.nextSSOEndpointMeta(ctx, nextEndpoint, requestOptions{Form: nextMeta, Headers: gigyaHeaders, AllowRedirects: &allowRedirect})
		if err != nil {
			return err
		}
		s.logStep(ctx, "gigya_hop", nextEndpoint, hop+1)
	}

	return &Error{Kind: ErrClient, Msg: fmt.Sprintf("gigya hop limit reached (%d)", maxSSOHops)}
}

func hasSSOTerminalMeta(meta map[string]string) bool {
	if meta == nil {
		return false
	}
	if _, ok := meta["SAMLResponse"]; ok {
		return true
	}
	if _, ok := meta["login_hint"]; ok {
		return true
	}
	return false
}

func (s *SSO) nextSSOEndpointMeta(ctx context.Context, rawURL string, opt requestOptions) (string, map[string]string, error) {
	if s.getEndpointMetaFn != nil {
		return s.getEndpointMetaFn(ctx, rawURL, opt)
	}
	return s.getSSOEndpointMeta(ctx, rawURL, opt)
}

func (s *SSO) logStep(ctx context.Context, step, endpoint string, hop int) {
	logger := s.client.logger
	if logger == nil {
		logger = slog.Default()
	}
	if !logger.Enabled(ctx, slog.LevelDebug) {
		return
	}
	host := ""
	if u, err := url.Parse(endpoint); err == nil {
		host = u.Host
	}
	logger.DebugContext(ctx, "sso step",
		slog.String("step", step),
		slog.String("endpoint_host", host),
		slog.Int("hop", hop),
	)
}

func (s *SSO) IsSessionActive(ctx context.Context) (bool, error) {
	res, _, err := s.client.request(ctx, http.MethodGet, URLAccountAttrs, requestOptions{})
	if err != nil {
		return false, err
	}
	contentType := res.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return true, nil
	}
	return false, nil
}

func (s *SSO) getSSOEndpointMeta(ctx context.Context, rawURL string, opt requestOptions) (string, map[string]string, error) {
	method := http.MethodGet
	if len(opt.Form) > 0 || opt.JSON != nil || len(opt.Body) > 0 {
		method = http.MethodPost
	}
	res, body, err := s.client.request(ctx, method, rawURL, opt)
	if err != nil {
		return "", nil, err
	}
	htmlBody := string(body)
	lower := strings.ToLower(htmlBody)
	if strings.Contains(lower, "we could not authenticate you") {
		return "", nil, &Error{Kind: ErrHTTP, Msg: "unauthorized", Status: 401, URL: res.Request.URL.String()}
	}
	if parseTitle(htmlBody) == "Change Your Password" {
		return "", nil, &Error{Kind: ErrHTTP, Msg: "your password has expired", Status: 401, URL: res.Request.URL.String()}
	}
	action, metadata, parseErr := parseFormActionAndInputs(htmlBody)
	if parseErr != nil {
		return "", nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("unable to find SAML form: %s", res.Request.URL.String())}
	}
	endpoint := action
	if !(strings.HasPrefix(action, "http://") || strings.HasPrefix(action, "https://")) {
		u, parseURLerr := url.Parse(res.Request.URL.String())
		if parseURLerr != nil {
			return "", nil, parseURLerr
		}
		ref, refErr := url.Parse(action)
		if refErr != nil {
			return "", nil, refErr
		}
		endpoint = u.ResolveReference(ref).String()
	}
	return endpoint, metadata, nil
}

func (s *SSO) getGigyaLoginParams(ctx context.Context, rawURL string, data map[string]string) (map[string]string, error) {
	allowRedirect := true
	res, _, err := s.client.request(ctx, http.MethodPost, rawURL, requestOptions{Form: data, AllowRedirects: &allowRedirect})
	if err != nil {
		return nil, err
	}
	params := map[string]string{}
	for k, v := range res.Request.URL.Query() {
		if len(v) > 0 {
			params[k] = v[0]
		}
	}
	return params, nil
}

func (s *SSO) gigyaWebSDKBootstrap(ctx context.Context, params map[string]string) error {
	p := cloneMap(params)
	p["pageURL"] = URLAccountSAMLProxy + "?apiKey=" + params["apiKey"]
	p["sdk"] = "js_latest"
	p["sdkBuild"] = "12426"
	p["format"] = "json"
	headers := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*"}
	_, _, err := s.client.request(ctx, http.MethodGet, URLAccountCDCApi+"/accounts.webSdkBootstrap", requestOptions{Params: p, Headers: headers})
	return err
}

func (s *SSO) getGigyaAuthCode(ctx context.Context, username, password string) (string, error) {
	headers := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*", "Content-Type": "application/json;charset=utf-8"}
	auth := map[string]string{"login": username, "password": password}
	_, body, err := s.client.request(ctx, http.MethodPost, URLAccountCoreAPI+"/authenticate", requestOptions{Params: map[string]string{"reqId": URLSupportPortal}, Headers: headers, JSON: auth})
	if err != nil {
		return "", err
	}
	var out gigyaAuthResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.CookieValue, nil
}

func (s *SSO) getGigyaLoginToken(ctx context.Context, samlParams map[string]string, authCode string) (string, error) {
	body, err := s.cdcAPIRequest(ctx, "socialize.notifyLogin", samlParams, map[string]string{"sessionExpiration": "0", "authCode": authCode})
	if err != nil {
		return "", err
	}
	var out cdcNotifyLoginResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.LoginToken, nil
}

func (s *SSO) getIDToken(ctx context.Context, samlParams map[string]string, loginToken string) (string, error) {
	body, err := s.cdcAPIRequest(ctx, "accounts.getJWT", samlParams, map[string]string{"expiration": "180", "login_token": loginToken})
	if err != nil {
		return "", err
	}
	var out cdcJWTResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.IDToken, nil
}

func (s *SSO) getUID(ctx context.Context, samlParams map[string]string, loginToken string) (string, error) {
	body, err := s.cdcAPIRequest(ctx, "accounts.getAccountInfo", samlParams, map[string]string{"include": "profile,data", "login_token": loginToken})
	if err != nil {
		return "", err
	}
	var out cdcAccountInfoResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.UID, nil
}

func (s *SSO) getUIDDetails(ctx context.Context, uid, idToken string) (*uidDetailsResponse, error) {
	headers := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*", "Authorization": "Bearer " + idToken}
	_, body, err := s.client.request(ctx, http.MethodGet, URLAccountCoreAPI+"/accounts/"+uid, requestOptions{Headers: headers})
	if err != nil {
		return nil, err
	}
	var out uidDetailsResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func isUIDLinkedMultipleSIDs(uidDetails *uidDetailsResponse) bool {
	if uidDetails == nil {
		return false
	}
	count := 0
	for _, account := range uidDetails.Accounts {
		count += len(account.LinkedAccounts)
	}
	return count > 1
}

func (s *SSO) selectAccount(ctx context.Context, uid, sid, idToken string) error {
	headers := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*", "Authorization": "Bearer " + idToken}
	_, _, err := s.client.request(ctx, http.MethodPut, URLAccountCoreAPI+"/accounts/"+uid+"/selectedAccount", requestOptions{Headers: headers, JSON: map[string]string{"idsName": sid, "automatic": "false"}})
	return err
}

func (s *SSO) getSDKBuildNumber(ctx context.Context, apiKey string) (string, error) {
	if s.sdkBuildNumber != "" {
		return s.sdkBuildNumber, nil
	}
	_, body, err := s.client.request(ctx, http.MethodGet, URLGigyaJS, requestOptions{Params: map[string]string{"apiKey": apiKey}})
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`gigya\.build\s*=\s*{[\s\S]+"number"\s*:\s*(\d+),`)
	m := re.FindStringSubmatch(string(body))
	if len(m) < 2 {
		return "", &Error{Kind: ErrClient, Msg: "unable to find gigya sdk build number"}
	}
	s.sdkBuildNumber = m[1]
	return s.sdkBuildNumber, nil
}

func (s *SSO) cdcAPIRequest(ctx context.Context, endpoint string, samlParams, queryParams map[string]string) ([]byte, error) {
	u := strings.TrimRight(URLAccountCDCApi, "/") + "/" + endpoint

	queryParts := make([]string, 0, len(samlParams))
	for k, v := range samlParams {
		queryParts = append(queryParts, k+"="+v)
	}
	pageURL := url.QueryEscape(URLAccountSAMLProxy + "?" + strings.Join(queryParts, "&"))

	apiKey := samlParams["apiKey"]
	sdkBuild, err := s.getSDKBuildNumber(ctx, apiKey)
	if err != nil {
		return nil, err
	}
	params := map[string]string{
		"sdk":      "js_latest",
		"APIKey":   apiKey,
		"authMode": "cookie",
		"pageURL":  pageURL,
		"sdkBuild": sdkBuild,
		"format":   "json",
	}
	for k, v := range queryParams {
		params[k] = v
	}
	headers := map[string]string{"Origin": URLAccount, "Referer": URLAccount, "Accept": "*/*"}
	_, body, err := s.client.request(ctx, http.MethodGet, u, requestOptions{Params: params, Headers: headers})
	if err != nil {
		return nil, err
	}
	var out cdcBaseResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	if out.ErrorCode != 0 {
		return nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("CDC API Error: %d - %s", out.StatusCode, out.ErrorMessage)}
	}
	return body, nil
}

func cloneMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
