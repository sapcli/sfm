package sfm

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func decodeODataResults[T any](data []byte) ([]T, error) {
	var payload struct {
		D struct {
			Results []T `json:"results"`
		} `json:"d"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, err
	}
	return payload.D.Results, nil
}

func decodeODataSingle[T any](data []byte) (T, error) {
	var payload struct {
		D T `json:"d"`
	}
	var zero T
	if err := json.Unmarshal(data, &payload); err != nil {
		return zero, err
	}
	return payload.D, nil
}

func encodeParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	// deterministic order
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	parts := make([]string, 0, len(keys))
	repl := strings.NewReplacer(
		"+", "%20",
		"%24", "$",
		"%28", "(",
		"%29", ")",
		"%2C", ",",
		"%2F", "/",
		"%27", "'",
	)
	for _, k := range keys {
		ek := repl.Replace(url.QueryEscape(k))
		ev := repl.Replace(url.QueryEscape(params[k]))
		parts = append(parts, ek+"="+ev)
	}
	return strings.Join(parts, "&")
}

func getNewExpiryDate(days int) string {
	d := time.Now().AddDate(0, 0, days).Format("2006-01-02")
	return fmt.Sprintf("datetime'%sT00:00:00'", d)
}

var tsDigits = regexp.MustCompile(`\d+`)

func getDateFromTimestamp(ts string) (time.Time, error) {
	m := tsDigits.FindString(ts)
	if m == "" {
		return time.Time{}, fmt.Errorf("no timestamp digits in %q", ts)
	}
	v, err := strconv.ParseInt(m, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.UnixMilli(v), nil
}

func chunkStrings(items []string, n int) [][]string {
	if n <= 0 {
		n = 20
	}
	out := make([][]string, 0, (len(items)+n-1)/n)
	for i := 0; i < len(items); i += n {
		j := i + n
		if j > len(items) {
			j = len(items)
		}
		out = append(out, items[i:j])
	}
	return out
}

func escapeODataString(v string) string {
	return strings.ReplaceAll(v, "'", "''")
}
