package sfm

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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

// entry maps a set-cookie URL to the cookies that were set for it,
// so that loading uses the same URL (host) that was used during save.
type entry struct {
	URL     string       `json:"url"`
	Cookies []cookieData `json:"cookies"`
}

type savedCookies struct {
	Entries []entry `json:"entries"`
}

// DefaultCookiePath returns the default cookie file path for the given
// username, scoped per user so switching S-IDs doesn't reuse stale sessions.
func DefaultCookiePath(username string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
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
		return err
	}
	var sc savedCookies
	if err := json.Unmarshal(data, &sc); err != nil {
		return err
	}
	for _, e := range sc.Entries {
		u, err := url.Parse(e.URL)
		if err != nil {
			continue
		}
		for _, cd := range e.Cookies {
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
			c.jar.SetCookies(u, []*http.Cookie{ck})
		}
	}
	return nil
}

// SaveCookies serializes all cookies for known domains to the given file.
func (c *core) SaveCookies(path string) error {
	seen := map[string]bool{}
	var entries []entry
	for _, domain := range knownDomains {
		u, err := url.Parse(domain)
		if err != nil {
			continue
		}
		cookies := c.jar.Cookies(u)
		if len(cookies) == 0 {
			continue
		}
		var list []cookieData
		for _, ck := range cookies {
			key := ck.Name + "|" + ck.Domain + "|" + ck.Path
			if seen[key] {
				continue
			}
			seen[key] = true
			list = append(list, cookieData{
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
		if len(list) > 0 {
			entries = append(entries, entry{URL: u.String(), Cookies: list})
		}
	}

	data, err := json.MarshalIndent(savedCookies{Entries: entries}, "", "  ")
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
