// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bufio"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/config"
	"github.com/bjblazko/kanso/internal/i18n"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
)

// testServer runs the whole application with the dev login.
type testServer struct {
	*httptest.Server
	deps Deps
}

func newTestServer(t *testing.T) *testServer {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	ts := &testServer{}
	ts.Server = httptest.NewUnstartedServer(nil)
	cfg := config.Config{DataDir: t.TempDir(), StateDir: t.TempDir(), BaseURL: "http://" + ts.Listener.Addr().String(),
		AdminGroup: "kanso-admin", DevLogin: true}
	ts.deps = buildTestDeps(t, cfg)
	srv, err := New(ts.deps)
	if err != nil {
		t.Fatal(err)
	}
	ts.Config.Handler = srv.Handler()
	ts.Start()
	t.Cleanup(ts.Close)
	return ts
}

func buildTestDeps(t *testing.T, cfg config.Config) Deps {
	t.Helper()
	repo, err := store.OpenRepo(cfg.DataDir)
	if err != nil {
		t.Fatal(err)
	}
	us, _ := users.NewStore(cfg.DataDir, repo)
	sess, _ := auth.NewSessions(cfg.StateDir)
	a, err := auth.NewAuth(cfg, nil, sess, us)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _ := i18n.Load()
	return Deps{Cfg: cfg, Auth: a, I18n: bundle, Users: us, Repo: repo}
}

// client signs in through the dev login and returns a client with cookies.
func (ts *testServer) client(t *testing.T, sub, name string, admin bool) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	c := &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	form := url.Values{"sub": {sub}, "name": {name}}
	if admin {
		form.Set("admin", "on")
	}
	req, _ := http.NewRequest("POST", ts.URL+"/auth/dev", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("dev login: %v %v", resp.StatusCode, err)
	}
	return c
}

func get(t *testing.T, c *http.Client, u string) (*http.Response, string) {
	t.Helper()
	resp, err := c.Get(u)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b strings.Builder
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		b.WriteString(sc.Text() + "\n")
	}
	return resp, b.String()
}

func TestSecurityHeaders(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := get(t, http.DefaultClient, ts.URL+"/healthz")
	want := map[string]string{
		"Content-Security-Policy": "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'",
		"X-Content-Type-Options":  "nosniff",
		"Referrer-Policy":         "same-origin",
	}
	for k, v := range want {
		if got := resp.Header.Get(k); got != v {
			t.Errorf("%s = %q", k, got)
		}
	}
}

func TestStartRedirectsToLogin(t *testing.T) {
	ts := newTestServer(t)
	c := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, _ := get(t, c, ts.URL+"/")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/auth/login") {
		t.Errorf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestStartPageShowsUserAndCSRF(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, c, ts.URL+"/")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Anna") || !strings.Contains(body, `<meta name="csrf" content="`) {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
}

func TestUnknownRouteIs404Page(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, c, ts.URL+"/nope/nothing")
	if resp.StatusCode != http.StatusNotFound || !strings.Contains(body, "Diese Seite gibt es nicht.") {
		t.Errorf("%d %s", resp.StatusCode, body)
	}
}

func TestLanguageSwitch(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	_, body := get(t, c, ts.URL+"/")
	token := between(body, `<meta name="csrf" content="`, `"`)
	req, _ := http.NewRequest("POST", ts.URL+"/me/lang", strings.NewReader(url.Values{"lang": {"en"}, "csrf": {token}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil || resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("switch: %v %v", resp.StatusCode, err)
	}
	_, body = get(t, c, ts.URL+"/about/licenses")
	if !strings.Contains(body, "Licenses and thanks") || !strings.Contains(body, `lang="en"`) {
		t.Errorf("not English: %s", body)
	}
}

func between(s, start, end string) string {
	_, after, _ := strings.Cut(s, start)
	v, _, _ := strings.Cut(after, end)
	return v
}

func TestStaticAssetsAreCached(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := get(t, http.DefaultClient, ts.URL+"/static/css/tokens.css")
	if resp.StatusCode != http.StatusOK || !strings.Contains(resp.Header.Get("Cache-Control"), "immutable") {
		t.Errorf("%d %q", resp.StatusCode, resp.Header.Get("Cache-Control"))
	}
}

func TestLicensesComplete(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	_, body := get(t, c, ts.URL+"/about/licenses")
	for _, mod := range requiredModules(t) {
		if !strings.Contains(body, mod) {
			t.Errorf("licenses page lacks Go module %s", mod)
		}
	}
	files, _ := filepath.Glob("../../web/static/licenses/*.txt")
	for _, f := range files {
		if !strings.Contains(body, "/static/licenses/"+filepath.Base(f)) {
			t.Errorf("licenses page does not link %s", filepath.Base(f))
		}
	}
	if len(files) == 0 {
		t.Error("no license files")
	}
}

func requiredModules(t *testing.T) []string {
	t.Helper()
	b, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	var mods []string
	inBlock := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "require (":
			inBlock = true
		case line == ")":
			inBlock = false
		case strings.HasPrefix(line, "require "):
			mods = append(mods, strings.Fields(line)[1])
		case inBlock && line != "":
			mods = append(mods, strings.Fields(line)[0])
		}
	}
	return mods
}

func TestStaticHasNoDirectoryListing(t *testing.T) {
	ts := newTestServer(t)
	resp, _ := get(t, http.DefaultClient, ts.URL+"/static/licenses/")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("directory listing: %d", resp.StatusCode)
	}
}
