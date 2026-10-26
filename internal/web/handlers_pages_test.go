// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
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
