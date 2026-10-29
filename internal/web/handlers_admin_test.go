// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

// Group names are written as people say them: capitals, umlauts, spaces.
func TestGroupNamesAsPeopleWriteThem(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	for _, name := range []string{"Familie", "Großeltern", "Kita Gruppe"} {
		resp := ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"create"}, "group": {name}})
		if resp.StatusCode != http.StatusSeeOther {
			t.Errorf("%q: %d", name, resp.StatusCode)
		}
	}
	g, _ := access.LoadGroups(ts.deps.Cfg.DataDir)
	if _, ok := g["Großeltern"]; !ok || len(g) != 3 {
		t.Fatalf("groups %v", g)
	}
}

// A refused change says why on the page instead of a bare "Bad Request".
func TestGroupErrorsSayWhy(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"create"}, "group": {"Familie"}})
	cases := map[string]string{
		"Familie":               "Diese Gruppe gibt es schon.",
		"  ":                    "Gib der Gruppe einen Namen",
		strings.Repeat("x", 41): "Gib der Gruppe einen Namen",
		"a/b":                   "Gib der Gruppe einen Namen",
	}
	for name, want := range cases {
		resp, body := ts.postFormBody(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"create"}, "group": {name}})
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, want) || !strings.Contains(body, `role="alert"`) {
			t.Errorf("%q: %d, want %q", name, resp.StatusCode, want)
		}
	}
}

func TestDeleteGroup(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	resp := ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"delete"}, "group": {"familie"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("delete: %d", resp.StatusCode)
	}
	if resp, _ := get(t, anna, ts.URL+"/s/haus"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("member of a deleted group still sees the space: %d", resp.StatusCode)
	}
	if _, body := get(t, admin, ts.URL+"/admin/groups"); strings.Contains(body, "<h2>familie") {
		t.Error("deleted group still listed")
	}
}

// postFormBody is postForm that also returns the answer's body.
func (ts *testServer) postFormBody(t *testing.T, c *http.Client, page, action string, form url.Values) (*http.Response, string) {
	t.Helper()
	_, body := get(t, c, ts.URL+page)
	form.Set("csrf", between(body, `<meta name="csrf" content="`, `"`))
	req, _ := http.NewRequest("POST", ts.URL+action, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b := new(strings.Builder)
	_, _ = io.Copy(b, resp.Body)
	return resp, b.String()
}
