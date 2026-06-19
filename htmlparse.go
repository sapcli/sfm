package sfm

import (
	"html"
	"regexp"
	"strings"
)

var (
	reForm  = regexp.MustCompile(`(?is)<form[^>]*action\s*=\s*["']?([^"'>\s]+)[^>]*>(.*?)</form>`)
	reInput = regexp.MustCompile(`(?is)<input[^>]*>`)
	reAttr  = regexp.MustCompile(`([a-zA-Z_:][-a-zA-Z0-9_:.]*)\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	reA     = regexp.MustCompile(`(?is)<a[^>]*href\s*=\s*["']([^"']+)["']`)
	reTitle = regexp.MustCompile(`(?is)<title>(.*?)</title>`)
)

func parseFormActionAndInputs(raw string) (string, map[string]string, error) {
	m := reForm.FindStringSubmatch(raw)
	if len(m) < 3 {
		return "", nil, &Error{Kind: ErrParse, Msg: "unable to find HTML form"}
	}
	action := html.UnescapeString(strings.TrimSpace(m[1]))
	inner := m[2]
	inputs := map[string]string{}
	for _, token := range reInput.FindAllString(inner, -1) {
		attrs := map[string]string{}
		for _, match := range reAttr.FindAllStringSubmatch(token, -1) {
			v := match[3]
			if v == "" {
				v = match[4]
			}
			if v == "" {
				v = match[5]
			}
			attrs[strings.ToLower(match[1])] = html.UnescapeString(v)
		}
		if attrs["type"] == "submit" {
			continue
		}
		name := attrs["name"]
		if name == "" {
			continue
		}
		inputs[name] = attrs["value"]
	}
	return action, inputs, nil
}

func parseTitle(raw string) string {
	m := reTitle.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(html.UnescapeString(m[1]))
}

func parseFirstAnchorHref(raw string) string {
	m := reA.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return html.UnescapeString(m[1])
}
