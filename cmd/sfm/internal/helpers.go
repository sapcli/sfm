package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"text/tabwriter"
	"time"

	sfm "github.com/sapcli/sfm"
)

var (
	Username     *string
	Password     *string
	Timeout      *time.Duration
	HTTPLogLevel *string
	DebugBodyMax *int
	OutputFormat *string
)

func MustClient() *sfm.Client {
	var opts []sfm.ClientOption
	opts = append(opts, sfm.WithTimeout(*Timeout), sfm.WithHTTPDebugBodyMax(*DebugBodyMax))

	// Enable cookie session persistence to avoid re-login on every command.
	if *Username != "" {
		if p, err := sfm.DefaultCookiePath(*Username); err == nil {
			opts = append(opts, sfm.WithCookiePersistence(p))
		}
	}

	if strings.TrimSpace(*HTTPLogLevel) != "" {
		level, err := ParseLogLevel(*HTTPLogLevel)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
		opts = append(opts, sfm.WithLogger(logger), sfm.WithHTTPLogLevel(level))
	}
	client, err := sfm.NewClient(*Username, *Password, opts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	return client
}

func Print(v any) {
	switch *OutputFormat {
	case "text":
		printText(v)
	case "table":
		printTable(v)
	default:
		printJSON(v)
	}
}

// ValidateOutputFormat returns an error if format is not supported.
var ValidOutputFormats = []string{"json", "text", "table"}

func ValidateOutputFormat(format string) error {
	for _, f := range ValidOutputFormats {
		if f == format {
			return nil
		}
	}
	return fmt.Errorf("invalid output format %q (expected %s)", format, strings.Join(ValidOutputFormats, "|"))
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func printText(v any) {
	switch val := v.(type) {
	case map[string]any:
		for k, v := range val {
			fmt.Printf("%s: %v\n", k, v)
		}
	default:
		if rows := toRows(v); rows != nil {
			printTextRows(rows)
		} else {
			fmt.Printf("%v\n", v)
		}
	}
}

func printTextRows(rows []any) {
	if len(rows) == 0 {
		return
	}

	// Extract headers in declaration order from original struct type.
	headers := structFieldNames(rows[0])

	mapped := make([]any, len(rows))
	for i, r := range rows {
		if m := structToMap(r); m != nil {
			mapped[i] = m
		} else {
			mapped[i] = r
		}
	}

	// Fall back to map key iteration for non-struct types.
	if headers == nil {
		if first, ok := mapped[0].(map[string]any); ok {
			for k := range first {
				headers = append(headers, k)
			}
		}
	}

	if len(headers) == 0 {
		return
	}

	for i, h := range headers {
		if i > 0 {
			fmt.Print(",")
		}
		fmt.Print(strings.ToUpper(h))
	}
	fmt.Println()

	for _, r := range mapped {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		for i, h := range headers {
			if i > 0 {
				fmt.Print(",")
			}
			fmt.Print(csvCell(m[h]))
		}
		fmt.Println()
	}
}

func csvCell(v any) string {
	s := fmt.Sprintf("%v", v)
	if strings.ContainsAny(s, ",\"\n") {
		s = strings.ReplaceAll(s, "\"", "\"\"")
		s = `"` + s + `"`
	}
	return s
}

func printTable(v any) {
	switch val := v.(type) {
	case map[string]any:
		if results, ok := val["results"].([]any); ok && len(results) > 0 {
			printTableRows(results)
		} else {
			printText(v)
		}
	default:
		if rows := toRows(v); rows != nil {
			printTableRows(rows)
		} else {
			printText(v)
		}
	}
}

func toRows(v any) []any {
	if rows := toSliceAny(v); rows != nil {
		return rows
	}
	// Pass the original struct so printTextRows/printTableRows can
	// extract ordered field names via structFieldNames before conversion.
	if structToMap(v) != nil {
		return []any{v}
	}
	return nil
}

func toSliceAny(v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return nil
	}
	n := rv.Len()
	out := make([]any, n)
	for i := range n {
		out[i] = rv.Index(i).Interface()
	}
	return out
}

func printTableRows(rows []any) {
	if len(rows) == 0 {
		return
	}

	// Extract headers in declaration order from original struct type.
	headers := structFieldNames(rows[0])

	mapped := make([]any, len(rows))
	for i, r := range rows {
		if m := structToMap(r); m != nil {
			mapped[i] = m
		} else {
			mapped[i] = r
		}
	}

	// Fall back to map key iteration for non-struct types.
	if headers == nil {
		if first, ok := mapped[0].(map[string]any); ok {
			for k := range first {
				headers = append(headers, k)
			}
		}
	}

	if len(headers) == 0 {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, strings.ToUpper(h))
	}
	fmt.Fprintln(w)

	for _, r := range mapped {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, m[h])
		}
		fmt.Fprintln(w)
	}
	w.Flush()
}

func structToMap(v any) map[string]any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	m := make(map[string]any, rv.NumField())
	t := rv.Type()
	for i := range rv.NumField() {
		ft := t.Field(i)
		if !ft.IsExported() {
			continue
		}
		name := ft.Name
		if tag, ok := ft.Tag.Lookup("json"); ok {
			if tag == "-" {
				continue
			}
			tag, _, _ = strings.Cut(tag, ",")
			name = tag
		}
		m[name] = rv.Field(i).Interface()
	}
	return m
}

// structFieldNames returns exported field names (with json tag support)
// in declaration order, or nil if v is not a struct or pointer to struct.
func structFieldNames(v any) []string {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}

	t := rv.Type()
	names := make([]string, 0, t.NumField())
	for i := range t.NumField() {
		ft := t.Field(i)
		if !ft.IsExported() {
			continue
		}
		name := ft.Name
		if tag, ok := ft.Tag.Lookup("json"); ok {
			if tag == "-" {
				continue
			}
			tag, _, _ = strings.Cut(tag, ",")
			if tag != "" {
				name = tag
			}
		}
		names = append(names, name)
	}
	return names
}

func ParseLogLevel(raw string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid -http-log-level %q (expected debug|info|warn|error)", raw)
	}
}
