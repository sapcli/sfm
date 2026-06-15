package launchpad

import "fmt"

type ErrorKind string

const (
	ErrClient        ErrorKind = "client"
	ErrPartnerLocked ErrorKind = "partner_locked"
	ErrInvalidSID    ErrorKind = "invalid_sid"
	ErrParse         ErrorKind = "parse"
	ErrHTTP          ErrorKind = "http"
)

type Error struct {
	Kind   ErrorKind
	Msg    string
	Status int
	URL    string
	Err    error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	base := fmt.Sprintf("%s: %s", e.Kind, e.Msg)
	if e.Status > 0 {
		base = fmt.Sprintf("%s (status=%d)", base, e.Status)
	}
	if e.URL != "" {
		base = fmt.Sprintf("%s [%s]", base, e.URL)
	}
	if e.Err != nil {
		base = fmt.Sprintf("%s: %v", base, e.Err)
	}
	return base
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}
