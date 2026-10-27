// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestServiceWorkerHeaders(t *testing.T) {
	ts := newTestServer(t)
	resp, body := get(t, http.DefaultClient, ts.URL+"/sw.js")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("sw.js: %d", resp.StatusCode)
	}
	want := map[string]string{
		"Content-Type":           "text/javascript; charset=utf-8",
		"Cache-Control":          "no-cache",
		"Service-Worker-Allowed": "/",
	}
	for k, v := range want {
		if got := resp.Header.Get(k); got != v {
			t.Errorf("%s = %q", k, got)
		}
	}
	if !strings.Contains(body, `const VERSION = "`+ts.srv.version+`";`) {
		t.Error("service worker does not carry the asset version")
	}
}

// precached reads the PRECACHE list the server writes into sw.js.
func precached(t *testing.T, body string) []string {
	t.Helper()
	var list []string
	if err := json.Unmarshal([]byte(between(body, "const PRECACHE = ", ";\n")), &list); err != nil {
		t.Fatalf("PRECACHE: %v", err)
	}
	return list
}

func TestServiceWorkerListsAssets(t *testing.T) {
	ts := newTestServer(t)
	_, body := get(t, http.DefaultClient, ts.URL+"/sw.js")
	list := precached(t, body)
	for _, want := range []string{"/static/js/capture.js?v=", "/static/css/app.css?v=", "/static/fonts/IBMPlexSans-Regular-Latin1.woff2"} {
		found := false
		for _, u := range list {
			found = found || strings.HasPrefix(u, want)
		}
		if !found {
			t.Errorf("%s not precached", want)
		}
	}
	for _, u := range list {
		if resp, _ := get(t, http.DefaultClient, ts.URL+u); resp.StatusCode != http.StatusOK {
			t.Errorf("%s: %d", u, resp.StatusCode)
		}
	}
}

func TestManifest(t *testing.T) {
	ts := newTestServer(t)
	resp, body := get(t, http.DefaultClient, ts.URL+"/manifest.webmanifest")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/manifest+json" {
		t.Fatalf("manifest: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	var m struct {
		Name, ShortName, StartURL, Display string
		Icons                              []struct{ Src, Sizes string }
	}
	if err := json.Unmarshal([]byte(strings.NewReplacer(`"short_name"`, `"ShortName"`, `"start_url"`, `"StartURL"`).Replace(body)), &m); err != nil {
		t.Fatal(err)
	}
	if m.Name != "Hüpattl! Kanso" || m.ShortName != "Kanso" || m.StartURL != "/notes/new" || m.Display != "standalone" || len(m.Icons) < 2 {
		t.Errorf("manifest = %+v", m)
	}
	for _, icon := range m.Icons {
		if resp, _ := get(t, http.DefaultClient, ts.URL+icon.Src); resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" {
			t.Errorf("icon %s: %d", icon.Src, resp.StatusCode)
		}
	}
}

func TestLayoutLinksManifestAndIcon(t *testing.T) {
	ts := newTestServer(t)
	_, body := get(t, ts.client(t, "anna", "Anna", false), ts.URL+"/")
	for _, want := range []string{`<link rel="manifest" href="/manifest.webmanifest">`, `rel="apple-touch-icon"`, `<meta name="theme-color"`} {
		if !strings.Contains(body, want) {
			t.Errorf("layout lacks %s", want)
		}
	}
}

func TestTokenEndpoint(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, page := get(t, anna, ts.URL+"/notes/new")
	resp, body := get(t, anna, ts.URL+"/notes/token")
	want := between(page, `<meta name="csrf" content="`, `"`)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/json" || !strings.Contains(body, `"csrf":"`+want+`"`) {
		t.Errorf("token: %d %s", resp.StatusCode, body)
	}
	noRedirect := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if resp, _ := get(t, noRedirect, ts.URL+"/notes/token"); resp.StatusCode == http.StatusOK {
		t.Error("token without sign-in")
	}
}

func TestCapturePageLoadsOfflineScript(t *testing.T) {
	ts := newTestServer(t)
	_, body := get(t, ts.client(t, "anna", "Anna", false), ts.URL+"/notes/new")
	if !strings.Contains(body, `src="/static/js/capture.js?v=`) || !strings.Contains(body, `data-login-url="/auth/login?next=%2Fnotes%2Fnew"`) {
		t.Error("capture page lacks its script or login link")
	}
}
