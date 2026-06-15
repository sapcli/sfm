package launchpad

import (
	"context"
	"strings"
	"testing"
)

func TestHasSSOTerminalMeta(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		meta map[string]string
		want bool
	}{
		{name: "nil", meta: nil, want: false},
		{name: "empty", meta: map[string]string{}, want: false},
		{name: "saml response", meta: map[string]string{"SAMLResponse": "x"}, want: true},
		{name: "login hint", meta: map[string]string{"login_hint": "x"}, want: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := hasSSOTerminalMeta(tt.meta)
			if got != tt.want {
				t.Fatalf("hasSSOTerminalMeta()=%v want=%v", got, tt.want)
			}
		})
	}
}

func TestStepTraverseSAMLStopsAtHopLimit(t *testing.T) {
	t.Parallel()

	calls := 0
	s := &SSO{
		client: &core{username: "S1234567890", password: "secret"},
		getEndpointMetaFn: func(_ context.Context, endpoint string, _ requestOptions) (string, map[string]string, error) {
			calls++
			return endpoint, map[string]string{}, nil
		},
	}

	err := s.stepTraverseSAML(context.Background(), &authFlowContext{endpoint: URLLaunchpad, meta: map[string]string{}})
	if err == nil {
		t.Fatalf("expected hop limit error")
	}
	if !strings.Contains(err.Error(), "hop limit") {
		t.Fatalf("expected hop limit error, got: %v", err)
	}
	if calls != maxSSOHops {
		t.Fatalf("expected %d hops, got %d", maxSSOHops, calls)
	}
}

func TestStepTraverseSAMLEarlyTerminal(t *testing.T) {
	t.Parallel()

	calls := 0
	s := &SSO{
		client: &core{username: "S1234567890", password: "secret"},
		getEndpointMetaFn: func(_ context.Context, endpoint string, _ requestOptions) (string, map[string]string, error) {
			calls++
			return endpoint, map[string]string{}, nil
		},
	}

	err := s.stepTraverseSAML(context.Background(), &authFlowContext{endpoint: URLLaunchpad, meta: map[string]string{"SAMLResponse": "already-there"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("expected 0 calls when terminal meta is preset, got %d", calls)
	}
}
