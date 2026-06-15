package launchpad

import (
	"net/http"
	"strings"
	"testing"
)

func TestBuildPayloadIncludesRequestLine(t *testing.T) {
	req, err := BuildBatchRequest(http.MethodDelete, "/UserSet('S123')", nil, nil, nil)
	if err != nil {
		t.Fatalf("BuildBatchRequest failed: %v", err)
	}
	boundary, payload, err := BuildPayload([]BatchItem{{Request: req}}, false)
	if err != nil {
		t.Fatalf("BuildPayload failed: %v", err)
	}
	if boundary == "" {
		t.Fatalf("expected boundary")
	}
	if !strings.Contains(payload, "DELETE UserSet('S123') HTTP/1.1") {
		t.Fatalf("payload missing request line: %s", payload)
	}
}

func TestParseBatchResponse(t *testing.T) {
	body := "--batch_1\r\nContent-Type: application/http\r\n\r\nHTTP/1.1 200 OK\r\nHeader: x\r\n\r\n{\"ok\":true}\r\n--batch_1--\r\n"
	res := &http.Response{Header: http.Header{"Content-Type": []string{"multipart/mixed; boundary=batch_1"}}}
	results, err := ParseBatchResponse(res, []byte(body))
	if err != nil {
		t.Fatalf("ParseBatchResponse failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected one result, got %d", len(results))
	}
	if results[0].Code != "200" || results[0].Status != "OK" {
		t.Fatalf("unexpected result: %+v", results[0])
	}
}
