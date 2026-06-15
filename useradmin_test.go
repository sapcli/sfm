package launchpad

import (
	"context"
	"testing"
)

func TestUserAdminSearchInvalidCustomerID(t *testing.T) {
	ua := &UserAdmin{}
	_, err := ua.Search(context.Background(), "alice@example.com", SearchOption{CustomerID: "abc123"})
	if err == nil {
		t.Fatalf("expected validation error for invalid customer id")
	}
	e, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected *Error type, got %T", err)
	}
	if e.Kind != ErrClient {
		t.Fatalf("expected ErrClient, got %s", e.Kind)
	}
}
