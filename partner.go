package sfm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type PartnerUser struct {
	core        *core
	api         string
	partnerData *PartnerData
	csrfToken   string
}

type partnerCreatePCF struct {
	PartnerType     string `json:"PartnerType"`
	PartnerFunction string `json:"PartnerFunction"`
	Action          string `json:"Action"`
}

type partnerCreatePayload struct {
	PrmConpID                      string             `json:"PrmConpId"`
	PrmBpID                        string             `json:"PrmBpId"`
	Email                          string             `json:"Email"`
	FirstName                      string             `json:"Firstname"`
	Function                       string             `json:"Function"`
	LastName                       string             `json:"Lastname"`
	CCPhone                        string             `json:"CCPhone"`
	OPhone                         string             `json:"OPhone"`
	UserExpiry                     string             `json:"UserExpiry"`
	ContactToPCF                   []partnerCreatePCF `json:"ContactToPCF"`
	ContactToRelationshipAttribute []any              `json:"ContactToRelationshipAttribute"`
	ContactToPcfDelimitNav         []any              `json:"ContactToPcfDelimitNav"`
	ContactID                      string             `json:"ContactId,omitempty"`
}

type partnerLockResponse struct {
	IsLockedFlag string `json:"IsLockedFlag"`
}

var (
	reSignature = regexp.MustCompile(`signature=(.*?);`)
	reLocation  = regexp.MustCompile(`location="(.*)"`)
)

func (p *PartnerUser) Auth(ctx context.Context) error {
	active, err := p.core.Auth().IsSessionActive(ctx)
	fmt.Println(active, err)
	if err != nil {
		return err
	}
	if !active {
		p.core.Logout()
		if err := p.core.Login(ctx); err != nil {
			return err
		}
	}

	_, body, err := p.core.request(ctx, http.MethodGet, URLPartnerEdge+"/index.html", requestOptions{})
	if err != nil {
		return err
	}
	txt := string(body)
	sigMatch := reSignature.FindStringSubmatch(txt)
	locMatch := reLocation.FindStringSubmatch(txt)
	if len(sigMatch) < 2 || len(locMatch) < 2 {
		return &Error{Kind: ErrClient, Msg: "unable to parse partner login metadata"}
	}

	edgeURL, _ := url.Parse(URLPartnerEdge)
	_ = p.core.setCookie(URLPartnerEdge, &http.Cookie{Name: "signature", Value: sigMatch[1], Domain: edgeURL.Host})
	_ = p.core.setCookie(URLPartnerEdge, &http.Cookie{Name: "locationAfterLogin", Value: "%2Findex.html", Domain: edgeURL.Host})
	_ = p.core.setCookie(URLPartnerEdge, &http.Cookie{Name: "fragmentAfterLogin", Value: "", Domain: edgeURL.Host})

	_, body2, err := p.core.request(ctx, http.MethodGet, locMatch[1], requestOptions{})
	if err != nil {
		return err
	}
	loginURL := parseFirstAnchorHref(string(body2))
	if loginURL == "" {
		return &Error{Kind: ErrClient, Msg: "unable to find partner login URL"}
	}

	edp, meta, err := p.core.Auth().getSSOEndpointMeta(ctx, loginURL, requestOptions{})
	if err != nil {
		return err
	}
	if _, _, err := p.core.request(ctx, http.MethodPost, edp, requestOptions{Form: meta}); err != nil {
		return err
	}

	_, body3, err := p.core.request(ctx, http.MethodGet, URLPartnerEdge+"/index.html", requestOptions{})
	if err != nil {
		return err
	}
	if parseTitle(string(body3)) != "Manage My Users" {
		return &Error{Kind: ErrClient, Msg: "authentication failed"}
	}
	p.csrfToken = ""
	return nil
}

func (p *PartnerUser) Users(ctx context.Context) ([]PartnerContact, error) {
	_, body, err := p.request(ctx, http.MethodGet, "ContactsSet", requestOptions{})
	if err != nil {
		return nil, err
	}
	return decodeODataResults[PartnerContact](body)
}

func (p *PartnerUser) Create(ctx context.Context, req CreatePartnerUserRequest) (*PartnerContact, error) {
	if req.CheckDuplicate {
		users, err := p.Search(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if len(users) > 0 {
			return nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("SUser already exists for %s", req.Email)}
		}
	}

	expireDate := req.ExpireDateMS
	if expireDate == 0 {
		expireDate = time.Now().Add(730 * 24 * time.Hour).UnixMilli()
	}

	partner, err := p.GetPartnerData(ctx)
	if err != nil {
		return nil, err
	}
	country := partner.Country
	phone := partner.Telephone
	partnerID := partner.PrmBpID

	locked, err := p.IsPartnerLocked(ctx, partnerID)
	if err != nil {
		return nil, err
	}
	if locked {
		return nil, &Error{Kind: ErrPartnerLocked, Msg: fmt.Sprintf("partner %s is currently locked from editing", partnerID)}
	}

	payload := partnerCreatePayload{
		PrmConpID:  "",
		PrmBpID:    partnerID,
		Email:      req.Email,
		FirstName:  req.FirstName,
		Function:   "",
		LastName:   req.LastName,
		CCPhone:    country,
		OPhone:     phone,
		UserExpiry: fmt.Sprintf("/Date(%d)/", expireDate),
		ContactToPCF: []partnerCreatePCF{{
			PartnerType:     "",
			PartnerFunction: "YPBASIC",
			Action:          "01",
		}},
		ContactToRelationshipAttribute: []any{},
		ContactToPcfDelimitNav:         []any{},
	}
	if req.ContactID != "" {
		payload.ContactID = req.ContactID
	}

	_, body, err := p.request(ctx, http.MethodPost, "ContactsSet", requestOptions{Headers: map[string]string{"Content-Type": "application/json"}, JSON: payload})
	if err != nil {
		return nil, err
	}
	v, err := decodeODataSingle[PartnerContact](body)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (p *PartnerUser) GetPartnerFunctions(ctx context.Context, user PartnerContact) ([]PartnerFunction, error) {
	partnerID := user.PrmBpID
	userID := user.PrmConpID
	endpoint := fmt.Sprintf("ContactsSet(PrmBpId='%s',PrmConpId='%s')/ContactToPCF", partnerID, userID)
	_, body, err := p.request(ctx, http.MethodGet, endpoint, requestOptions{})
	if err != nil {
		return nil, err
	}
	var payload struct {
		Results []PartnerFunction `json:"results"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Results, nil
}

func (p *PartnerUser) DeleteByEmail(ctx context.Context, email string) (bool, error) {
	partner, err := p.GetPartnerData(ctx)
	if err != nil {
		return false, err
	}
	partnerID := partner.PrmBpID

	results, err := p.Search(ctx, email)
	if err != nil {
		return false, err
	}
	if len(results) == 0 {
		return false, nil
	}
	userID := results[0].PrmConpID
	if err := p.Delete(ctx, userID, partnerID); err != nil {
		return false, err
	}
	return true, nil
}

func (p *PartnerUser) Delete(ctx context.Context, conpID, bpID string) error {
	endpoint := fmt.Sprintf("ContactsSet(PrmConpId='%s',PrmBpId='%s')", conpID, bpID)
	_, _, err := p.request(ctx, http.MethodDelete, endpoint, requestOptions{})
	return err
}

func (p *PartnerUser) IsPartnerLocked(ctx context.Context, partnerID string) (bool, error) {
	_, body, err := p.request(ctx, http.MethodGet, fmt.Sprintf("PartnerLockSet('%s')", partnerID), requestOptions{})
	if err != nil {
		return false, err
	}
	v, err := decodeODataSingle[partnerLockResponse](body)
	if err != nil {
		return false, err
	}
	return v.IsLockedFlag == "X", nil
}

func (p *PartnerUser) IsDuplicated(ctx context.Context, email string) (bool, error) {
	results, err := p.Search(ctx, email)
	if err != nil {
		return false, err
	}
	return len(results) > 0, nil
}

func (p *PartnerUser) Search(ctx context.Context, email string) ([]PartnerContact, error) {
	partner, err := p.GetPartnerData(ctx)
	if err != nil {
		return nil, err
	}
	partnerID := partner.PrmBpID

	params := map[string]string{
		"PrmBpId":   partnerID,
		"Email":     email,
		"Firstname": "",
		"Lastname":  "",
		"Function":  "",
		"Phone":     "",
	}
	parts := make([]string, 0, len(params))
	for k, v := range params {
		parts = append(parts, fmt.Sprintf("%s eq '%s'", k, escapeODataString(v)))
	}
	query := strings.Join(parts, " and ")
	endpoint := "ContactDupCheckSet?$filter=" + url.QueryEscape(query)

	_, body, err := p.request(ctx, http.MethodGet, endpoint, requestOptions{})
	if err != nil {
		return nil, err
	}
	results, err := decodeODataResults[PartnerContact](body)
	if err != nil {
		return nil, err
	}
	out := make([]PartnerContact, 0, len(results))
	for _, r := range results {
		if r.PrmConpID != "" {
			out = append(out, r)
		}
	}
	return out, nil
}

func (p *PartnerUser) GetPartnerData(ctx context.Context) (*PartnerData, error) {
	if p.partnerData != nil {
		copyVal := *p.partnerData
		return &copyVal, nil
	}
	_, body, err := p.request(ctx, http.MethodGet, "PartnerDataSet", requestOptions{})
	if err != nil {
		return nil, err
	}
	results, err := decodeODataResults[PartnerData](body)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, &Error{Kind: ErrClient, Msg: "empty partner data"}
	}
	p.partnerData = &results[0]
	copyVal := *p.partnerData
	return &copyVal, nil
}

func (p *PartnerUser) getXSRFToken(ctx context.Context) (string, error) {
	if p.csrfToken != "" {
		return p.csrfToken, nil
	}
	res, _, err := p.request(ctx, http.MethodHead, "", requestOptions{Headers: map[string]string{"x-csrf-token": "Fetch"}})
	if err != nil {
		return "", err
	}
	p.csrfToken = res.Header.Get("x-csrf-token")
	return p.csrfToken, nil
}

func (p *PartnerUser) request(ctx context.Context, method, endpoint string, opt requestOptions) (*http.Response, []byte, error) {
	headers := map[string]string{"Accept": "application/json"}
	for k, v := range opt.Headers {
		headers[k] = v
	}
	opt.Headers = headers

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete {
		token, err := p.getXSRFToken(ctx)
		if err != nil {
			return nil, nil, err
		}
		opt.Headers["x-csrf-token"] = token
	}

	rawURL := p.api + endpoint
	res, body, err := p.core.request(ctx, method, rawURL, opt)
	if err != nil {
		return nil, nil, err
	}
	if res.StatusCode == 401 || strings.Contains(string(body), "locationAfterLogin=") {
		if err := p.Auth(ctx); err != nil {
			return nil, nil, err
		}
		return p.request(ctx, method, endpoint, opt)
	}
	return res, body, nil
}
