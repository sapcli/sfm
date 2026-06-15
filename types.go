package launchpad

type User struct {
	Susid        string `json:"Susid"`
	Ipadr        string `json:"Ipadr"`
	Kunnr        string `json:"Kunnr"`
	Namev        string `json:"Namev"`
	Name1        string `json:"Name1"`
	Anred        string `json:"Anred"`
	ParlaExt     string `json:"ParlaExt"`
	Department   string `json:"Department"`
	DepartmentID string `json:"DepartmentId"`
}

type RequestedUser struct {
	RequestID     string `json:"RequestId"`
	UserID        string `json:"UserId"`
	Status        string `json:"Status"`
	TypeOfRequest string `json:"TypeOfRequest"`
	ChangedBy     string `json:"ChangedBy"`
	ExpDate       string `json:"ExpDate"`
	Ipadr         string `json:"Ipadr"`
	Susid         string `json:"Susid"`
}

type Department struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Customer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type AuthRequest struct {
	RequestID     string `json:"RequestId"`
	UserID        string `json:"UserId"`
	Status        string `json:"Status"`
	TypeOfRequest string `json:"TypeOfRequest"`
	ChangedBy     string `json:"ChangedBy"`
	ExpDate       string `json:"ExpDate"`
	Ipadr         string `json:"Ipadr"`
	Susid         string `json:"Susid"`
}

type PartnerContact struct {
	PrmConpID string `json:"PrmConpId"`
	PrmBpID   string `json:"PrmBpId"`
	SUser     string `json:"SUser"`
	Email     string `json:"Email"`
	FirstName string `json:"Firstname"`
	LastName  string `json:"Lastname"`
	ContactID string `json:"ContactId"`
}

type PartnerData struct {
	SUser                                    string `json:"SUser"`
	PrmBpID                                  string `json:"PrmBpId"`
	Country                                  string `json:"Country"`
	Telephone                                string `json:"Telephone"`
	PartnerName                              string `json:"PartnerName"`
	IsExternalUser                           string `json:"IsExternalUser"`
	PSM                                      string `json:"PSM"`
	PSMHasAgreedUponRolesAndResponsabilities string `json:"PSMhasAgreedUponRolesAndResponsabilities"`
	IsCentralBlocked                         string `json:"IsCentralBlocked"`
	IsMarkedForDeletion                      string `json:"IsMarkedForDeletion"`
	IsNotReleased                            string `json:"IsNotReleased"`
}

type PartnerFunction struct {
	PartnerType     string `json:"PartnerType"`
	PartnerFunction string `json:"PartnerFunction"`
	Action          string `json:"Action"`
}

type BatchResult struct {
	Code    string
	Status  string
	Content string
}

type SearchOption struct {
	Field      string
	CustomerID string
}

type RequestFilter struct {
	Status    string
	Type      string
	Requester string
	Processor string
}

type CreateUserRequest struct {
	Email        string
	FirstName    string
	LastName     string
	CustomerID   string
	Salutation   string
	DepartmentID string
	Language     string
}

type CreatePartnerUserRequest struct {
	Email          string
	FirstName      string
	LastName       string
	ExpireDateMS   int64
	ContactID      string
	CheckDuplicate bool
}
