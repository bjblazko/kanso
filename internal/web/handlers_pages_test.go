// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

// createPage creates a page as c and returns its id from the redirect.
func (ts *testServer) createPage(t *testing.T, c *http.Client, space, parent, title string) string {
	t.Helper()
	resp := ts.postForm(t, c, "/s/"+space, "/s/"+space+"/new", url.Values{"title": {title}, "parent": {parent}})
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/p/") {
		t.Fatalf("create page: %d %q", resp.StatusCode, loc)
	}
	return strings.TrimPrefix(loc, "/p/")
}

func TestPageLifecycleOverHTTP(t *testing.T) {
	ts, admin, anna := setupHaus(t) // anna is editor via group
	garten := ts.createPage(t, anna, "haus", "", "Garten")
	rasen := ts.createPage(t, anna, "haus", garten, "Rasen")

	resp, body := get(t, anna, ts.URL+"/p/"+rasen)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Rasen") || !strings.Contains(body, `href="/p/`+garten+`"`) {
		t.Errorf("view: %d", resp.StatusCode)
	}
	_, spaceBody := get(t, anna, ts.URL+"/s/haus")
	if !strings.Contains(spaceBody, `href="/p/`+rasen+`"`) {
		t.Error("space page lacks tree entry")
	}
	if r := ts.postForm(t, anna, "/p/"+garten, "/p/"+garten+"/delete", url.Values{}); r.StatusCode != http.StatusConflict {
		t.Errorf("delete with children: %d", r.StatusCode)
	}
	if r := ts.postForm(t, anna, "/p/"+rasen, "/p/"+rasen+"/move", url.Values{"parent": {""}, "order": {"1"}}); r.StatusCode != http.StatusSeeOther {
		t.Errorf("move: %d", r.StatusCode)
	}
	if r := ts.postForm(t, admin, "/p/"+garten, "/p/"+garten+"/delete", url.Values{}); r.StatusCode != http.StatusSeeOther {
		t.Errorf("delete leaf: %d", r.StatusCode)
	}
	if r, _ := get(t, anna, ts.URL+"/p/"+garten); r.StatusCode != http.StatusNotFound {
		t.Errorf("deleted page: %d", r.StatusCode)
	}
}

func TestViewerCannotCreateOutsiderCannotSee(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Geheim")
	oma := ts.client(t, "oma", "Oma", false)
	if r, _ := get(t, oma, ts.URL+"/p/"+id); r.StatusCode != http.StatusNotFound {
		t.Errorf("outsider view: %d", r.StatusCode)
	}
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if r, _ := get(t, oma, ts.URL+"/p/"+id); r.StatusCode != http.StatusOK {
		t.Errorf("viewer view: %d", r.StatusCode)
	}
	if r := ts.postForm(t, oma, "/s/haus", "/s/haus/new", url.Values{"title": {"X"}}); r.StatusCode != http.StatusForbidden {
		t.Errorf("viewer create: %d", r.StatusCode)
	}
	if r := ts.postForm(t, oma, "/p/"+id, "/p/"+id+"/delete", url.Values{}); r.StatusCode != http.StatusForbidden {
		t.Errorf("viewer delete: %d", r.StatusCode)
	}
}

func TestPageTitleIsEscaped(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "<img src=x onerror=alert(1)>")
	_, body := get(t, admin, ts.URL+"/p/"+id)
	if strings.Contains(body, "<img src=x") {
		t.Error("title not escaped")
	}
}

func TestPageViewRendersMarkdown(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	body := "# Wartung\n\n**Jährlich** prüfen.\n\n```mermaid\ngraph TD\n  A-->B\n```\n"
	if err := ts.deps.Wiki.Save(store.Author{Name: "Admin", Email: "a@x"}, id, "Heizung", body); err != nil {
		t.Fatal(err)
	}
	_, html := get(t, admin, ts.URL+"/p/"+id)
	if !strings.Contains(html, "<strong>Jährlich</strong>") || !strings.Contains(html, `<pre class="mermaid">`) {
		t.Errorf("not rendered: %s", html)
	}
}

func postPreview(t *testing.T, ts *testServer, c *http.Client, id, md string) (*http.Response, string) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/p/"+id)
	req, _ := http.NewRequest("POST", ts.URL+"/api/preview", strings.NewReader(url.Values{"page": {id}, "body": {md}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", between(page, `<meta name="csrf" content="`, `"`))
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestPreviewEndpoint(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Vorschau")
	resp, out := postPreview(t, ts, anna, id, "*kursiv* ![x](attachments/a.png)")
	if resp.StatusCode != http.StatusOK || !strings.Contains(out, "<em>kursiv</em>") || !strings.Contains(out, "/p/"+id+"/attachments/a.png") {
		t.Errorf("preview: %d %s", resp.StatusCode, out)
	}
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"admin"}, "role": {"viewer"}})
	oma := ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if resp, _ := postPreview(t, ts, oma, id, "x"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer preview: %d", resp.StatusCode)
	}
}
