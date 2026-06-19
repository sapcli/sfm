package sfm

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type UserAdmin struct {
	core      *core
	csrfToken string
}

type userAdminCustomerRow struct {
	Kunnr string `json:"Kunnr"`
	Name1 string `json:"Name1"`
}

type userAdminDepartmentRow struct {
	DepartmentID   string `json:"DepartmentId"`
	DepartmentName string `json:"DepartmentName"`
}

type userAdminCreatePayload struct {
	Ipadr        string `json:"Ipadr"`
	Namev        string `json:"Namev"`
	Name1        string `json:"Name1"`
	Anred        string `json:"Anred"`
	Kunnr        string `json:"Kunnr"`
	ParlaExt     string `json:"ParlaExt"`
	Department   string `json:"Department"`
	DepartmentID string `json:"DepartmentId"`
}

type userAdminIsEmailDuplicate struct {
	BoolValue bool `json:"boolValue"`
}

type userAdminIsEmailDuplicateResponse struct {
	IsEmailDuplicate userAdminIsEmailDuplicate `json:"IsEmailDuplicate"`
}

const (
	StatusOpen     = "O"
	StatusRejected = "R"
	StatusApproved = "A"

	RequestTypeExpiry = "Expiry_date"
	RequestTypeAuth   = "Authorization"
)

func (u *UserAdmin) Users(ctx context.Context) ([]User, error) {
	users := make([]User, 0)
	skip := 0
	for {
		payload, err := u.request(ctx, "/UserSet", requestOptions{Params: map[string]string{"$top": "1000", "$skip": strconv.Itoa(skip)}})
		if err != nil {
			return nil, err
		}
		results, err := decodeODataResults[User](payload)
		if err != nil {
			return nil, err
		}
		users = append(users, results...)
		if len(results) < 1000 {
			break
		}
		skip += 1000
	}
	return users, nil
}

func (u *UserAdmin) RequestedUsers(ctx context.Context) ([]RequestedUser, error) {
	payload, err := u.request(ctx, "/RequestedUsersSet", requestOptions{})
	if err != nil {
		return nil, err
	}
	return decodeODataResults[RequestedUser](payload)
}

func (u *UserAdmin) Customers(ctx context.Context) ([]Customer, error) {
	payload, err := u.request(ctx, "/CustomerSet", requestOptions{})
	if err != nil {
		return nil, err
	}
	rows, err := decodeODataResults[userAdminCustomerRow](payload)
	if err != nil {
		return nil, err
	}
	out := make([]Customer, 0, len(rows))
	for _, r := range rows {
		out = append(out, Customer{ID: r.Kunnr, Name: r.Name1})
	}
	return out, nil
}

func (u *UserAdmin) Departments(ctx context.Context) ([]Department, error) {
	payload, err := u.request(ctx, "/DepartmentNewSet", requestOptions{})
	if err != nil {
		return nil, err
	}
	rows, err := decodeODataResults[userAdminDepartmentRow](payload)
	if err != nil {
		return nil, err
	}
	out := make([]Department, 0, len(rows))
	for _, r := range rows {
		out = append(out, Department{ID: r.DepartmentID, Name: r.DepartmentName})
	}
	return out, nil
}

func (u *UserAdmin) GetUser(ctx context.Context, userID string) (*User, error) {
	payload, err := u.request(ctx, fmt.Sprintf("/UserSet('%s')", userID), requestOptions{})
	if err != nil {
		return nil, err
	}
	v, err := decodeODataSingle[User](payload)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (u *UserAdmin) Search(ctx context.Context, keyword string, opt SearchOption) ([]User, error) {
	field := opt.Field
	if field == "" {
		field = "Ipadr"
	}
	if !isSimpleODataIdentifier(field) {
		return nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("invalid search field: %q", field)}
	}
	match := fmt.Sprintf("substringof('%s',%s)", escapeODataString(keyword), field)
	if opt.CustomerID != "" {
		id, err := strconv.Atoi(strings.TrimSpace(opt.CustomerID))
		if err != nil {
			return nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("invalid customer id: %q", opt.CustomerID), Err: err}
		}
		condition := fmt.Sprintf("Kunnr eq '%010d'", id)
		match = condition + " and " + match
	}
	payload, err := u.request(ctx, "/UserSet", requestOptions{Params: map[string]string{"$filter": match}})
	if err != nil {
		return nil, err
	}
	return decodeODataResults[User](payload)
}

func (u *UserAdmin) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
	if req.Salutation == "" {
		req.Salutation = "Mr"
	}
	if req.Language == "" {
		req.Language = "EN"
	}

	departmentID := req.DepartmentID
	departmentName := ""
	if departmentID != "" {
		dep, err := u.GetDepartment(ctx, departmentID)
		if err != nil {
			return nil, err
		}
		departmentName = dep.Name
	}

	customerID := req.CustomerID
	if customerID == "" {
		adminUser, err := u.GetUser(ctx, u.core.username)
		if err != nil {
			return nil, err
		}
		customerID = adminUser.Kunnr
	}

	dup, err := u.IsEmailDuplicated(ctx, req.Email, customerID)
	if err != nil {
		return nil, err
	}
	if dup {
		return nil, &Error{Kind: ErrClient, Msg: fmt.Sprintf("email %s is duplicated in %s", req.Email, customerID)}
	}

	payload, err := u.request(ctx, "/UserSet", requestOptions{JSON: userAdminCreatePayload{
		Ipadr:        req.Email,
		Namev:        req.FirstName,
		Name1:        req.LastName,
		Anred:        req.Salutation,
		Kunnr:        customerID,
		ParlaExt:     req.Language,
		Department:   departmentName,
		DepartmentID: departmentID,
	}})
	if err != nil {
		return nil, err
	}
	v, err := decodeODataSingle[User](payload)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (u *UserAdmin) Delete(ctx context.Context, userID string) error {
	req, err := BuildBatchRequest(http.MethodDelete, fmt.Sprintf("/UserSet('%s')", userID), nil, nil, nil)
	if err != nil {
		return err
	}
	boundary, data, err := BuildPayload([]BatchItem{{ChangeSet: []BatchItem{{Request: req}}}}, false)
	if err != nil {
		return err
	}
	token, err := u.getCSRFToken(ctx)
	if err != nil {
		return err
	}
	_, _, err = u.core.batchRequest(ctx, URLUserAdmin, token, boundary, data)
	return err
}

func (u *UserAdmin) ExtendExpiryDate(ctx context.Context, userIDs []string, days int) ([]BatchResult, error) {
	ids := strings.Join(userIDs, ",")
	req, err := BuildBatchRequest(http.MethodGet, "ExtendUserDate", nil, map[string]string{"UserIds": "'" + ids + "'", "ExpDat": getNewExpiryDate(days)}, nil)
	if err != nil {
		return nil, err
	}
	boundary, data, err := BuildPayload([]BatchItem{{Request: req}}, false)
	if err != nil {
		return nil, err
	}
	token, err := u.getCSRFToken(ctx)
	if err != nil {
		return nil, err
	}
	res, body, err := u.core.batchRequest(ctx, URLUserAdmin, token, boundary, data)
	if err != nil {
		return nil, err
	}
	return ParseBatchResponse(res, body)
}

func (u *UserAdmin) GetUserRequests(ctx context.Context, filter RequestFilter) ([]AuthRequest, error) {
	keywords := []string{}
	if filter.Status != "" {
		keywords = append(keywords, fmt.Sprintf("Status eq '%s'", escapeODataString(filter.Status)))
	}
	if filter.Type != "" {
		keywords = append(keywords, fmt.Sprintf("TypeOfRequest eq '%s'", escapeODataString(filter.Type)))
	}
	if filter.Requester != "" {
		keywords = append(keywords, fmt.Sprintf("UserId eq '%s'", escapeODataString(filter.Requester)))
	}
	if filter.Processor != "" {
		keywords = append(keywords, fmt.Sprintf("ChangedBy eq '%s'", escapeODataString(filter.Processor)))
	}
	params := map[string]string{}
	if len(keywords) > 0 {
		params["$filter"] = strings.Join(keywords, " and ")
	}
	payload, err := u.request(ctx, "/AuthorizationRequestSet", requestOptions{Params: params})
	if err != nil {
		return nil, err
	}
	return decodeODataResults[AuthRequest](payload)
}

func (u *UserAdmin) SetExpiryDateRequest(ctx context.Context, requestID, status, expiryDate string) error {
	_, err := u.request(ctx, fmt.Sprintf("/AuthorizationRequestSet('%s')", requestID), requestOptions{Headers: map[string]string{"x-http-method": "MERGE"}, JSON: map[string]string{"Status": status, "TypeOfRequest": RequestTypeExpiry, "ExpDate": expiryDate}})
	return err
}

func (u *UserAdmin) GetDepartment(ctx context.Context, departmentID string) (Department, error) {
	payload, err := u.request(ctx, fmt.Sprintf("/DepartmentNewSet('%s')", departmentID), requestOptions{})
	if err != nil {
		return Department{}, err
	}
	row, err := decodeODataSingle[userAdminDepartmentRow](payload)
	if err != nil {
		return Department{}, err
	}
	return Department{ID: row.DepartmentID, Name: row.DepartmentName}, nil
}

func (u *UserAdmin) GetCustomer(ctx context.Context, customerID string) (Customer, error) {
	payload, err := u.request(ctx, fmt.Sprintf("/CustomerNewSet('%s')", customerID), requestOptions{})
	if err != nil {
		return Customer{}, err
	}
	row, err := decodeODataSingle[userAdminCustomerRow](payload)
	if err != nil {
		return Customer{}, err
	}
	return Customer{ID: row.Kunnr, Name: row.Name1}, nil
}

func (u *UserAdmin) IsEmailDuplicated(ctx context.Context, email, customerID string) (bool, error) {
	payload, err := u.request(ctx, "/IsEmailDuplicate", requestOptions{Params: map[string]string{"Ipadr": "'" + escapeODataString(email) + "'", "Kunnr": "'" + escapeODataString(customerID) + "'"}})
	if err != nil {
		return false, err
	}
	row, err := decodeODataSingle[userAdminIsEmailDuplicateResponse](payload)
	if err != nil {
		return false, err
	}
	return row.IsEmailDuplicate.BoolValue, nil
}

func GetNewExpiryDate(days int) string {
	return getNewExpiryDate(days)
}

func GetDateFromTimestamp(ts string) (time.Time, error) {
	return getDateFromTimestamp(ts)
}

func ChunkUsers(users []string, n int) [][]string {
	return chunkStrings(users, n)
}

func isSimpleODataIdentifier(v string) bool {
	if v == "" {
		return false
	}
	for i := 0; i < len(v); i++ {
		c := v[i]
		isAlpha := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		isDigit := c >= '0' && c <= '9'
		if !isAlpha && !isDigit && c != '_' {
			return false
		}
	}
	return true
}

func (u *UserAdmin) getCSRFToken(ctx context.Context) (string, error) {
	if u.csrfToken != "" {
		return u.csrfToken, nil
	}
	res, _, err := u.core.request(ctx, http.MethodHead, URLUserAdmin, requestOptions{Headers: map[string]string{"x-csrf-token": "Fetch"}})
	if err != nil {
		return "", err
	}
	if res.Header.Get("com.sap.cloud.security.login") == "login-request" {
		if err := u.core.Login(ctx); err != nil {
			return "", err
		}
		u.csrfToken = ""
		return u.getCSRFToken(ctx)
	}
	u.csrfToken = res.Header.Get("x-csrf-token")
	return u.csrfToken, nil
}

func (u *UserAdmin) request(ctx context.Context, endpoint string, opt requestOptions) ([]byte, error) {
	headers := map[string]string{"Accept": "application/json"}
	for k, v := range opt.Headers {
		headers[k] = v
	}
	opt.Headers = headers

	if opt.JSON != nil || len(opt.Form) > 0 || len(opt.Body) > 0 {
		token, err := u.getCSRFToken(ctx)
		if err != nil {
			return nil, err
		}
		opt.Headers["x-csrf-token"] = token
	}
	rawURL := URLUserAdmin + endpoint
	method := http.MethodGet
	if opt.JSON != nil || len(opt.Form) > 0 || len(opt.Body) > 0 {
		method = http.MethodPost
	}
	res, payload, err := u.core.request(ctx, method, rawURL, opt)
	if err != nil {
		return nil, err
	}
	if res.Header.Get("com.sap.cloud.security.login") == "login-request" {
		u.csrfToken = ""
		if err := u.core.Login(ctx); err != nil {
			return nil, err
		}
		return u.request(ctx, endpoint, opt)
	}
	return payload, nil
}
