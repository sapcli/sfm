package sfm

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"time"
)

// knownDomains is the set of domains whose cookies we persist.
// After login, cookies on these domains carry the session.
var knownDomains = []string{
	URLLaunchpad,
	URLAccount,
	URLAccountCDCApi,
	URLAccountCoreAPI,
	URLPartnerEdge,
}

// cookieData is a JSON-serializable cookie without Raw/Unparsed fields
// that would not round-trip cleanly.
type cookieData struct {
	Name     string        `json:"name"`
	Value    string        `json:"value"`
	Domain   string        `json:"domain,omitempty"`
	Path     string        `json:"path,omitempty"`
	Expires  time.Time     `json:"expires,omitempty"`
	MaxAge   int           `json:"maxAge,omitempty"`
	Secure   bool          `json:"secure"`
	HttpOnly bool          `json:"httpOnly"`
	SameSite http.SameSite `json:"sameSite,omitempty"`
}

type savedCookies struct {
	Cookies []cookieData `json:"cookies"`
}

// DefaultCookiePath returns the default cookie file path for the given
// username, scoped per user so switching S-IDs doesn't reuse stale sessions.
func DefaultCookiePath(username string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Sanitize username for use as a filename component.
	sanitized := username
	if sanitized == "" {
		sanitized = "default"
	}
	dir := filepath.Join(home, ".config", "sfm")
	return filepath.Join(dir, "cookies-"+sanitized+".json"), nil
}

// LoadCookies populates the core's cookie jar from the given file.
func (c *core) LoadCookies(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err // includes file-not-found for callers that check
	}
	var sc savedCookies
	if err := json.Unmarshal(data, &sc); err != nil {
		return err
	}
	for _, cd := range sc.Cookies {
		ck := &http.Cookie{
			Name:     cd.Name,
			Value:    cd.Value,
			Domain:   cd.Domain,
			Path:     cd.Path,
			Expires:  cd.Expires,
			MaxAge:   cd.MaxAge,
			Secure:   cd.Secure,
			HttpOnly: cd.HttpOnly,
			SameSite: cd.SameSite,
		}
		// Determine a URL for this cookie to place it in the jar.
		// If Domain starts with ".", strip it for url.Parse.
		domain := cd.Domain
		if len(domain) > 0 && domain[0] == '.' {
			domain = domain[1:]
		}
		scheme := "https"
		if !cd.Secure {
			scheme = "http"
		}
		cookieURL, err := url.Parse(scheme + "://" + domain + cd.Path)
		if err != nil {
			continue
		}
		c.jar.SetCookies(cookieURL, []*http.Cookie{ck})
	}
	return nil
}

// SaveCookies serializes all cookies for known domains to the given file.
func (c *core) SaveCookies(path string) error {
	seen := map[string]bool{}
	var all []cookieData
	for _, domain := range knownDomains {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		for _, ck := range c.jar.Cookies(u) {
			// Deduplicate by name+domain+path.
			key := ck.Name + "|" + ck.Domain + "|" + ck.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			all = append(all, cookieData{
				Name:     ck.Name,
				Value:    ck.Value,
				Domain:   ck.Domain,
				Path:     ck.Path,
				Expires:  ck.Expires,
				MaxAge:   ck.MaxAge,
				Secure:   ck.Secure,
				HttpOnly: ck.HttpOnly,
				SameSite: ck.SameSite,
			})
		}
	}
	// Sort for deterministic output.
	slices.SortFunc(all, func(a, b cookieData) int {
		if a.Name < b.Name {
			return -1
		}
		if a.Name > b.Name {
			return 1
		}
		return 0
	})

	data, err := json.MarshalIndent(savedCookies{Cookies: all}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

// RemoveCookies deletes the cookie file.
func (c *core) RemoveCookies(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
