// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/config"
	"github.com/bjblazko/kanso/internal/devseed"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
)

const base = "http://localhost:8090"

func newAuth(t *testing.T, dev bool, p *Provider) *Auth {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	us, _ := users.NewStore(repo.Dir(), repo)
	sess, _ := NewSessions(t.TempDir())
	cfg := config.Config{BaseURL: base, AdminGroup: "kanso-admin", DevLogin: dev, StateDir: t.TempDir()}
	a, err := NewAuth(cfg, p, sess, us)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func protected(a *Auth) http.Handler {
	mux := http.NewServeMux()
	a.Routes(mux)
	mux.Handle("/", a.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, p := UserFrom(r.Context())
		_, _ = w.Write([]byte(u.Name + "|" + p.Sub + "|" + CSRFToken(r.Context())))
	})))
	return mux
}

func loggedIn(t *testing.T, a *Auth) (*http.Cookie, Session) {
	t.Helper()
	if _, err := a.users.Upsert(users.User{Sub: "anna", Name: "Anna", Email: "anna@example.org"}); err != nil {
		t.Fatal(err)
	}
	s := a.sessions.Create("anna", false)
	return &http.Cookie{Name: sessionCookie, Value: s.ID}, s
}

func TestRequireUserRedirectsGET(t *testing.T) {
	a := newAuth(t, true, nil)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, httptest.NewRequest("GET", "/s/haus?x=1", nil))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/auth/login?next="+url.QueryEscape("/s/haus?x=1") {
		t.Errorf("code %d location %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestRequireUserRejectsPOST(t *testing.T) {
	a := newAuth(t, true, nil)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, httptest.NewRequest("POST", "/x", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("code %d", rec.Code)
	}
}

func TestRequireUserPassesUser(t *testing.T) {
	a := newAuth(t, true, nil)
	c, s := loggedIn(t, a)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, req)
	if rec.Body.String() != "Anna|anna|"+s.CSRF {
		t.Errorf("body %q", rec.Body.String())
	}
}

func post(a *Auth, c *http.Cookie, token, origin string) int {
	form := url.Values{"csrf": {token}}
	req := httptest.NewRequest("POST", "/x", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	req.AddCookie(c)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, req)
	return rec.Code
}

func TestCSRF(t *testing.T) {
	a := newAuth(t, true, nil)
	c, s := loggedIn(t, a)
	if code := post(a, c, "", base); code != http.StatusForbidden {
		t.Errorf("no token: %d", code)
	}
	if code := post(a, c, s.CSRF, "https://evil.example"); code != http.StatusForbidden {
		t.Errorf("foreign origin: %d", code)
	}
	if code := post(a, c, s.CSRF, ""); code != http.StatusForbidden {
		t.Errorf("no origin: %d", code)
	}
	if code := post(a, c, s.CSRF, base); code != http.StatusOK {
		t.Errorf("valid: %d", code)
	}
}

func TestSafeNext(t *testing.T) {
	cases := map[string]string{
		"/s/haus":              "/s/haus",
		"https://evil.example": "/",
		"//evil.example":       "/",
		"/\\evil.example":      "/",
		"":                     "/",
	}
	for in, want := range cases {
		if got := safeNext(in); got != want {
			t.Errorf("safeNext(%q) = %q", in, got)
		}
	}
}

func TestDevRouteDisabled(t *testing.T) {
	a := newAuth(t, false, nil)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, httptest.NewRequest("GET", "/auth/dev", nil))
	if rec.Code == http.StatusOK {
		t.Errorf("dev login reachable: %d", rec.Code)
	}
}

func TestDevLoginCreatesSession(t *testing.T) {
	a := newAuth(t, true, nil)
	form := url.Values{"sub": {"ben"}, "name": {"Ben"}, "admin": {"on"}}
	req := httptest.NewRequest("POST", "/auth/dev", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", base)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("code %d: %s", rec.Code, rec.Body)
	}
	var sid string
	for _, c := range rec.Result().Cookies() {
		if c.Name == sessionCookie {
			sid = c.Value
		}
	}
	s, ok := a.sessions.Get(sid)
	if !ok || s.Sub != "ben" || !s.Admin {
		t.Errorf("session %+v %v", s, ok)
	}
}

func TestCallbackFlow(t *testing.T) {
	f := newFakeIdP(t)
	p, err := Discover(context.Background(), f.srv.URL, "kanso", "secret", base+"/auth/callback", f.srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	a := newAuth(t, false, p)
	h := protected(a)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/auth/login?next=/s/haus", nil))
	loc, _ := url.Parse(rec.Header().Get("Location"))
	state := loc.Query().Get("state")
	f.nonce = loc.Query().Get("nonce")
	authCookie := rec.Result().Cookies()[0]

	bad := httptest.NewRequest("GET", "/auth/callback?code=good-code&state=wrong", nil)
	bad.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, bad)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("state mismatch: %d", rec.Code)
	}

	good := httptest.NewRequest("GET", "/auth/callback?code=good-code&state="+state, nil)
	good.AddCookie(authCookie)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, good)
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/s/haus" {
		t.Fatalf("callback: %d %q %s", rec.Code, rec.Header().Get("Location"), rec.Body)
	}
	u, ok := a.users.Get("anna")
	if !ok || u.Name != "Anna" {
		t.Errorf("user %+v", u)
	}
}

func TestDevFormOffersEveryPerson(t *testing.T) {
	a := newAuth(t, true, nil)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, httptest.NewRequest("GET", "/auth/dev?next=/s/haus", nil))
	body := rec.Body.String()
	for _, p := range devseed.People {
		if !strings.Contains(body, `name="sub" value="`+p.Sub+`"`) || !strings.Contains(body, p.Name) {
			t.Errorf("no quick switch for %s", p.Sub)
		}
	}
	if !strings.Contains(body, `name="admin" value="on"`) {
		t.Error("admin person signs in without admin flag")
	}
	if strings.Count(body, `name="next" value="/s/haus"`) != len(devseed.People)+1 {
		t.Error("quick switch forgets where to go next")
	}
}

func TestDevLoginEndsPreviousSession(t *testing.T) {
	a := newAuth(t, true, nil)
	old, _ := loggedIn(t, a)
	form := url.Values{"sub": {"ben"}, "name": {"Ben"}}
	req := httptest.NewRequest("POST", "/auth/dev", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", base)
	req.AddCookie(old)
	rec := httptest.NewRecorder()
	protected(a).ServeHTTP(rec, req)
	if rec.Code != http.StatusSeeOther {
		t.Fatalf("code %d", rec.Code)
	}
	if _, ok := a.sessions.Get(old.Value); ok {
		t.Error("old session still valid after switching person")
	}
}
