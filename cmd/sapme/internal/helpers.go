package internal

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
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
	case []any:
		for i, item := range val {
			fmt.Printf("[%d] %v\n", i, item)
		}
	default:
		fmt.Printf("%v\n", v)
	}
}

func printTable(v any) {
	switch val := v.(type) {
	case map[string]any:
		if results, ok := val["results"].([]any); ok && len(results) > 0 {
			printTableRows(results)
		} else {
			printText(v)
		}
	case []any:
		printTableRows(val)
	default:
		printText(v)
	}
}

func printTableRows(rows []any) {
	if len(rows) == 0 {
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	first := rows[0]
	switch row := first.(type) {
	case map[string]any:
		headers := make([]string, 0, len(row))
		for k := range row {
			headers = append(headers, k)
		}
		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, strings.ToUpper(h))
		}
		fmt.Fprintln(w)

		for _, r := range rows {
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
	default:
		for _, r := range rows {
			fmt.Fprintf(w, "%v\n", r)
		}
	}
	w.Flush()
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
