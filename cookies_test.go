package sfm

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Parallel()

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c := &core{
		jar:        jar,
		httpClient: &http.Client{Jar: jar},
	}

	u, _ := url.Parse(URLLaunchpad)
	c.jar.SetCookies(u, []*http.Cookie{
		{Name: "MYSAPSSO2", Value: "abc123", Domain: ".support.sap.com", Path: "/", Secure: true, HttpOnly: true},
	})

	dir := t.TempDir()
	path := filepath.Join(dir, "cookies.json")
	if err := c.SaveCookies(path); err != nil {
		t.Fatalf("SaveCookies: %v", err)
	}

	jar2, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	c2 := &core{jar: jar2, httpClient: &http.Client{Jar: jar2}}
	if err := c2.LoadCookies(path); err != nil {
		t.Fatalf("LoadCookies: %v", err)
	}

	cookies := c2.jar.Cookies(u)
	if len(cookies) != 1 {
		t.Fatalf("expected 1 cookie after load, got %d", len(cookies))
	}
	if cookies[0].Name != "MYSAPSSO2" || cookies[0].Value != "abc123" {
		t.Fatalf("unexpected cookie: Name=%q Value=%q", cookies[0].Name, cookies[0].Value)
	}
}

func TestLoadNonexistentFile(t *testing.T) {
	t.Parallel()

	jar, _ := cookiejar.New(nil)
	c := &core{jar: jar}

	err := c.LoadCookies("/nonexistent/path/cookies.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestSaveEmptyJar(t *testing.T) {
	t.Parallel()

	jar, _ := cookiejar.New(nil)
	c := &core{jar: jar, httpClient: &http.Client{Jar: jar}}

	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	if err := c.SaveCookies(path); err != nil {
		t.Fatalf("SaveCookies(empty): %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty JSON for empty cookies")
	}
}

func TestLoadMalformedFile(t *testing.T) {
	t.Parallel()

	jar, _ := cookiejar.New(nil)
	c := &core{jar: jar}

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(path, []byte("{{{"), 0600); err != nil {
		t.Fatal(err)
	}

	err := c.LoadCookies(path)
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestRemoveCookies(t *testing.T) {
	t.Parallel()

	jar, _ := cookiejar.New(nil)
	c := &core{jar: jar}

	dir := t.TempDir()
	path := filepath.Join(dir, "remove_me.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := c.RemoveCookies(path); err != nil {
		t.Fatalf("RemoveCookies: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expected file to be removed")
	}
}

func TestDefaultCookiePath(t *testing.T) {
	t.Parallel()

	p, err := DefaultCookiePath("S1234567890")
	if err != nil {
		t.Fatal(err)
	}
	home, _ := os.UserHomeDir()
	want := filepath.Join(home, ".config", "sfm", "cookies-S1234567890.json")
	if p != want {
		t.Fatalf("DefaultCookiePath = %q, want %q", p, want)
	}
}
