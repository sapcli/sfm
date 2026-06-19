package sfm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

type ClientOption func(*clientConfig)

type clientConfig struct {
	RetryAttempts int
	Timeout       time.Duration
	Headers       map[string]string
	DebugHTTP     bool
	DebugBodyMax  int
	Logger        *slog.Logger
	LogLevel      slog.Leveler
}

func WithRetryAttempts(v int) ClientOption {
	return func(c *clientConfig) {
		if v > 0 {
			c.RetryAttempts = v
		}
	}
}

func WithTimeout(v time.Duration) ClientOption {
	return func(c *clientConfig) {
		if v > 0 {
			c.Timeout = v
		}
	}
}

func WithHTTPDebug(enabled bool) ClientOption {
	return func(c *clientConfig) {
		c.DebugHTTP = enabled
	}
}

func WithHTTPDebugBodyMax(v int) ClientOption {
	return func(c *clientConfig) {
		if v > 0 {
			c.DebugBodyMax = v
		}
	}
}

func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *clientConfig) {
		if logger != nil {
			c.Logger = logger
		}
	}
}

func WithHTTPLogLevel(level slog.Leveler) ClientOption {
	return func(c *clientConfig) {
		c.LogLevel = level
	}
}

type Client struct {
	Username string
	Password string
	core     *core

	auth      *SSO
	useradmin *UserAdmin
	partner   *PartnerUser
}

type core struct {
	username       string
	password       string
	getCredentials func() (string, string)

	httpClient     *http.Client
	jar            http.CookieJar
	retry          int
	headers        map[string]string
	logger         *slog.Logger
	httpLogLevel   slog.Leveler
	httpLogBodyMax int
	auth           *SSO

	onLogout func()
}

func NewClient(username, password string, opts ...ClientOption) (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	cfg := clientConfig{
		RetryAttempts: DefaultHTTPRetryAttempts,
		Timeout:       60 * time.Second,
		DebugBodyMax:  2048,
		Logger:        slog.Default(),
		Headers: map[string]string{
			"User-Agent": UserAgentChrome,
		},
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	transport := http.DefaultTransport
	effectiveHTTPLogLevel := cfg.LogLevel
	if cfg.DebugHTTP || cfg.LogLevel != nil {
		level := effectiveHTTPLogLevel
		if level == nil {
			level = slog.LevelDebug
			effectiveHTTPLogLevel = level
		}
		transport = &loggingRoundTripper{
			next:       transport,
			logger:     cfg.Logger,
			level:      level,
			bodyMaxLen: cfg.DebugBodyMax,
		}
	}

	co := &core{
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Jar:       jar,
			Transport: transport,
		},
		jar:            jar,
		retry:          cfg.RetryAttempts,
		headers:        cfg.Headers,
		logger:         cfg.Logger,
		httpLogLevel:   effectiveHTTPLogLevel,
		httpLogBodyMax: cfg.DebugBodyMax,
	}
	co.auth = &SSO{client: co}

	c := &Client{Username: username, Password: password, core: co, auth: co.auth}
	co.getCredentials = func() (string, string) {
		return c.Username, c.Password
	}
	c.useradmin = &UserAdmin{core: co}
	c.partner = &PartnerUser{core: co, api: URLPartnerEdge + "/sap/opu/odata/sap/YMMU_SERVICE_SRV/"}
	co.onLogout = func() {
		if c.useradmin != nil {
			c.useradmin.csrfToken = ""
		}
		if c.partner != nil {
			c.partner.csrfToken = ""
			c.partner.partnerData = nil
		}
	}

	return c, nil
}

func (c *Client) Auth() *SSO {
	return c.auth
}

func (c *Client) UserAdmin() *UserAdmin {
	return c.useradmin
}

func (c *Client) Partner() *PartnerUser {
	return c.partner
}

func (c *Client) Login(ctx context.Context) error {
	return c.core.Login(ctx)
}

func (c *Client) Logout() {
	c.core.Logout()
}

func (c *Client) Cookie(name string) (string, bool) {
	return c.core.Cookie(name)
}

func (c *Client) request(ctx context.Context, method, rawURL string, opt requestOptions) (*http.Response, []byte, error) {
	return c.core.request(ctx, method, rawURL, opt)
}

func (c *Client) batchRequest(ctx context.Context, baseURL, token, boundary, data string) (*http.Response, []byte, error) {
	return c.core.batchRequest(ctx, baseURL, token, boundary, data)
}

func (c *Client) setCookie(rawURL string, cookie *http.Cookie) error {
	return c.core.setCookie(rawURL, cookie)
}

func (c *core) Auth() *SSO {
	return c.auth
}

func (c *core) Login(ctx context.Context) error {
	if c.getCredentials != nil {
		u, p := c.getCredentials()
		c.username = u
		c.password = p
	}
	c.Logout()
	return c.Auth().Login(ctx)
}

func (c *core) Logout() {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return
	}
	c.jar = jar
	c.httpClient.Jar = jar
	if c.onLogout != nil {
		c.onLogout()
	}
}

func (c *core) Cookie(name string) (string, bool) {
	u, err := url.Parse(URLLaunchpad)
	if err != nil {
		return "", false
	}
	for _, ck := range c.jar.Cookies(u) {
		if ck.Name == name {
			return ck.Value, true
		}
	}
	return "", false
}

type requestOptions struct {
	Headers        map[string]string
	Params         map[string]string
	Form           map[string]string
	JSON           any
	Body           []byte
	AllowRedirects *bool
	Timeout        time.Duration
}

func (c *core) request(ctx context.Context, method, rawURL string, opt requestOptions) (*http.Response, []byte, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, nil, err
	}
	if len(opt.Params) > 0 {
		q := encodeParams(opt.Params)
		if parsedURL.RawQuery == "" {
			parsedURL.RawQuery = q
		} else {
			parsedURL.RawQuery += "&" + q
		}
	}

	body := opt.Body
	headers := map[string]string{}
	for k, v := range c.headers {
		headers[k] = v
	}
	for k, v := range opt.Headers {
		headers[k] = v
	}
	if opt.JSON != nil {
		b, err := json.Marshal(opt.JSON)
		if err != nil {
			return nil, nil, err
		}
		body = b
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/json"
		}
	}
	if len(opt.Form) > 0 {
		body = []byte(encodeParams(opt.Form))
		if _, ok := headers["Content-Type"]; !ok {
			headers["Content-Type"] = "application/x-www-form-urlencoded"
		}
	}

	finalURL := parsedURL.String()
	client := c.httpClient
	if opt.AllowRedirects != nil {
		client = &http.Client{
			Timeout:   c.httpClient.Timeout,
			Jar:       c.jar,
			Transport: c.httpClient.Transport,
		}
		if !*opt.AllowRedirects {
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}
	if opt.Timeout > 0 {
		client = &http.Client{
			Timeout:   opt.Timeout,
			Jar:       c.jar,
			Transport: c.httpClient.Transport,
		}
		if opt.AllowRedirects != nil && !*opt.AllowRedirects {
			client.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			}
		}
	}

	for attempt := 0; attempt <= c.retry; attempt++ {
		reqCtx := context.WithValue(ctx, httpAttemptContextKey{}, attempt+1)
		req, err := http.NewRequestWithContext(reqCtx, method, finalURL, bytes.NewReader(body))
		if err != nil {
			return nil, nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		res, err := client.Do(req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
				return nil, nil, err
			}
			if attempt < c.retry && isRetryableNetError(err) {
				if sleepErr := sleepWithContext(ctx, backoff(attempt)); sleepErr != nil {
					return nil, nil, sleepErr
				}
				continue
			}
			return nil, nil, &Error{Kind: ErrHTTP, Msg: "request failed", URL: finalURL, Err: err}
		}

		payload, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return nil, nil, readErr
		}
		c.logHTTPResponse(ctx, attempt+1, res, payload)
		res.Body = io.NopCloser(bytes.NewReader(payload))

		if _, ok := retryStatusForcelist[res.StatusCode]; ok && attempt < c.retry {
			if sleepErr := sleepWithContext(ctx, backoff(attempt)); sleepErr != nil {
				return nil, nil, sleepErr
			}
			continue
		}
		if res.StatusCode >= 400 {
			msg := strings.TrimSpace(string(payload))
			if len(msg) > 512 {
				msg = msg[:512]
			}
			return res, payload, &Error{Kind: ErrHTTP, Msg: msg, Status: res.StatusCode, URL: finalURL}
		}

		return res, payload, nil
	}

	return nil, nil, &Error{Kind: ErrHTTP, Msg: "request retry exhausted", URL: finalURL}
}

func isRetryableNetError(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}
	return false
}

func backoff(attempt int) time.Duration {
	seconds := math.Pow(2, float64(attempt+1))
	return time.Duration(seconds * float64(time.Second))
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *core) batchRequest(ctx context.Context, baseURL, token, boundary, data string) (*http.Response, []byte, error) {
	headers := map[string]string{
		"Accept":               "multipart/mixed",
		"Content-Type":         fmt.Sprintf("multipart/mixed;boundary=%s", boundary),
		"Referer":              URLLaunchpad,
		"sap-cancel-on-close":  "true",
		"sap-contextid-accept": "header",
		"x-csrf-token":         token,
	}
	batchURL := strings.TrimRight(baseURL, "/") + "/$batch"
	return c.request(ctx, http.MethodPost, batchURL, requestOptions{Headers: headers, Body: []byte(data)})
}

func (c *core) setCookie(rawURL string, cookie *http.Cookie) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	c.jar.SetCookies(u, []*http.Cookie{cookie})
	return nil
}

type httpAttemptContextKey struct{}

type loggingRoundTripper struct {
	next       http.RoundTripper
	logger     *slog.Logger
	level      slog.Leveler
	bodyMaxLen int
}

func (rt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	logger := rt.logger
	if logger == nil {
		logger = slog.Default()
	}
	level := slog.LevelDebug
	if rt.level != nil {
		level = rt.level.Level()
	}
	if !logger.Enabled(req.Context(), level) {
		next := rt.next
		if next == nil {
			next = http.DefaultTransport
		}
		return next.RoundTrip(req)
	}

	attempt := 1
	if v, ok := req.Context().Value(httpAttemptContextKey{}).(int); ok {
		attempt = v
	}
	logger.Log(req.Context(), level, "http request",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.Int("attempt", attempt),
		slog.Any("headers", redactHeader(req.Header)),
		slog.String("body", bodyPreview(readRequestBody(req), rt.bodyMaxLen)),
	)

	next := rt.next
	if next == nil {
		next = http.DefaultTransport
	}
	res, err := next.RoundTrip(req)
	if err != nil {
		logger.Log(req.Context(), level, "http roundtrip error",
			slog.String("method", req.Method),
			slog.String("url", req.URL.String()),
			slog.Int("attempt", attempt),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	return res, nil
}

func readRequestBody(req *http.Request) []byte {
	if req == nil || req.Body == nil {
		return nil
	}
	if req.GetBody != nil {
		rc, err := req.GetBody()
		if err == nil {
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err == nil {
				return data
			}
		}
	}
	return nil
}

func (c *core) logHTTPResponse(ctx context.Context, attempt int, res *http.Response, payload []byte) {
	if c.httpLogLevel == nil {
		return
	}
	logger := c.logger
	if logger == nil {
		logger = slog.Default()
	}
	level := c.httpLogLevel.Level()
	if !logger.Enabled(ctx, level) {
		return
	}
	logger.Log(ctx, level, "http response",
		slog.Int("status_code", res.StatusCode),
		slog.String("url", res.Request.URL.String()),
		slog.Int("attempt", attempt),
		slog.Any("headers", redactHeader(res.Header)),
		slog.String("body", bodyPreview(payload, c.httpLogBodyMax)),
	)
}

func redactHeader(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vals := range h {
		v := ""
		if len(vals) > 0 {
			v = vals[0]
		}
		lk := strings.ToLower(k)
		switch lk {
		case "authorization", "cookie", "set-cookie", "x-csrf-token":
			out[k] = "<redacted>"
		default:
			out[k] = v
		}
	}
	return out
}

func bodyPreview(body []byte, max int) string {
	if len(body) == 0 {
		return ""
	}
	if max <= 0 || len(body) <= max {
		return string(body)
	}
	return string(body[:max]) + "...(truncated)"
}
