package sfm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type BatchItem struct {
	Request   *http.Request
	ChangeSet []BatchItem
}

func BuildPayload(items []BatchItem, isChangeset bool) (string, string, error) {
	indicator := "batch"
	if isChangeset {
		indicator = "changeset"
	}
	boundary := fmt.Sprintf("%s_%s", indicator, uid())
	var b strings.Builder
	if isChangeset {
		b.WriteString(fmt.Sprintf("Content-Type: multipart/mixed;boundary=%s\n\n", boundary))
	}

	for _, item := range items {
		b.WriteString(fmt.Sprintf("--%s\n", boundary))
		if len(item.ChangeSet) > 0 {
			_, nested, err := BuildPayload(item.ChangeSet, true)
			if err != nil {
				return "", "", err
			}
			b.WriteString(nested)
			continue
		}
		if item.Request == nil {
			continue
		}

		actionPath := item.Request.URL.String()
		if i := strings.LastIndex(actionPath, "/"); i >= 0 {
			actionPath = actionPath[i+1:]
		}
		action := fmt.Sprintf("%s %s HTTP/1.1", item.Request.Method, actionPath)

		headers := make([]string, 0, len(item.Request.Header))
		for k, vals := range item.Request.Header {
			if len(vals) == 0 {
				continue
			}
			headers = append(headers, fmt.Sprintf("%s: %s", k, vals[0]))
		}

		var body string
		if item.Request.Body != nil {
			payload, err := io.ReadAll(item.Request.Body)
			if err != nil {
				return "", "", err
			}
			_ = item.Request.Body.Close()
			item.Request.Body = io.NopCloser(bytes.NewReader(payload))
			body = string(payload)
		}

		b.WriteString("Content-Type: application/http\n")
		b.WriteString("Content-Transfer-Encoding: binary\n\n")
		b.WriteString(action + "\n")
		b.WriteString(strings.Join(headers, "\n") + "\n\n")
		b.WriteString(body + "\n")
	}
	b.WriteString(fmt.Sprintf("--%s--\n\n", boundary))

	return boundary, b.String(), nil
}

func BuildBatchRequest(method, endpoint string, headers map[string]string, params map[string]string, jsonBody any) (*http.Request, error) {
	commonHeaders := map[string]string{
		"sap-cancel-on-close":   "true",
		"sap-contextid-accept":  "header",
		"Accept":                "application/json",
		"Accept-Language":       "en",
		"DataServiceVersion":    "2.0",
		"MaxDataServiceVersion": "2.0",
	}
	for k, v := range headers {
		commonHeaders[k] = v
	}

	base, err := url.JoinPath(URLLaunchpad, endpoint)
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, err
	}
	if len(params) > 0 {
		u.RawQuery = encodeParams(params)
	}

	var body io.Reader
	if jsonBody != nil {
		payload, err := json.Marshal(jsonBody)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequest(method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for k, v := range commonHeaders {
		req.Header.Set(k, v)
	}
	return req, nil
}

func ParseBatchResponse(res *http.Response, body []byte) ([]BatchResult, error) {
	contentType := res.Header.Get("Content-Type")
	const marker = "multipart/mixed; boundary="
	if !strings.HasPrefix(contentType, marker) {
		return nil, &Error{Kind: ErrParse, Msg: "invalid $batch response content-type"}
	}
	boundary := strings.TrimPrefix(contentType, marker)
	sections := strings.Split(string(body), "--"+boundary)
	if len(sections) <= 2 {
		return nil, nil
	}

	out := make([]BatchResult, 0, len(sections)-2)
	for _, sec := range sections[1 : len(sections)-1] {
		s := strings.TrimSpace(sec)
		if s == "" || s == "--" {
			continue
		}
		parts := strings.Split(s, "\r\n\r\n")
		if len(parts) < 3 {
			continue
		}
		responseMsg := parts[1]
		content := strings.TrimSpace(parts[2])
		lines := strings.Split(responseMsg, "\r\n")
		if len(lines) == 0 {
			continue
		}
		first := strings.SplitN(lines[0], " ", 3)
		if len(first) < 3 {
			continue
		}
		out = append(out, BatchResult{Code: first[1], Status: first[2], Content: content})
	}
	return out, nil
}

func uid() string {
	return fmt.Sprintf("%04x-%04x-%04x", time.Now().UnixNano()&0xffff, (time.Now().UnixNano()>>16)&0xffff, (time.Now().UnixNano()>>32)&0xffff)
}
