package sfm

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type capturedRecord struct {
	msg   string
	attrs map[string]any
}

type captureHandler struct {
	mu      sync.Mutex
	records []capturedRecord
	level   slog.Level
}

func (h *captureHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	attrs := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	h.mu.Lock()
	h.records = append(h.records, capturedRecord{msg: r.Message, attrs: attrs})
	h.mu.Unlock()
	return nil
}

func (h *captureHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func TestHTTPResponseLoggingDoesNotConsumeBody(t *testing.T) {
	t.Parallel()

	const expectedBody = `{"ok":true,"hello":"world"}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, expectedBody)
	}))
	defer ts.Close()

	h := &captureHandler{level: slog.LevelDebug}
	logger := slog.New(h)

	c, err := NewClient(
		"S1234567890",
		"secret",
		WithLogger(logger),
		WithHTTPLogLevel(slog.LevelDebug),
		WithHTTPDebugBodyMax(4096),
		WithRetryAttempts(0),
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	_, payload, err := c.request(context.Background(), http.MethodGet, ts.URL, requestOptions{})
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if string(payload) != expectedBody {
		t.Fatalf("payload mismatch: got=%q want=%q", payload, expectedBody)
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	responseLogs := 0
	for _, rec := range h.records {
		if rec.msg == "http response" {
			responseLogs++
			body, _ := rec.attrs["body"].(string)
			if body != expectedBody {
				t.Fatalf("logged body mismatch: got=%q want=%q", body, expectedBody)
			}
		}
	}
	if responseLogs != 1 {
		t.Fatalf("expected exactly 1 http response log, got %d", responseLogs)
	}
}
