// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// postForm sends a form with the CSRF token taken from page.
func (ts *testServer) postForm(t *testing.T, c *http.Client, page, action string, form url.Values) *http.Response {
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
	_ = resp.Body.Close()
	return resp
}

// setup: admin creates "Haus", group "familie" with anna, family = editor.
func setupHaus(t *testing.T) (*testServer, *http.Client, *http.Client) {
	t.Helper()
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	anna := ts.client(t, "anna", "Anna", false)
	resp := ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Haus"}, "description": {"Alles ums Haus"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/s/haus" {
		t.Fatalf("create space: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"create"}, "group": {"familie"}})
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"add"}, "group": {"familie"}, "sub": {"anna"}})
	resp = ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"group"}, "who": {"familie"}, "role": {"editor"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("add member: %d", resp.StatusCode)
	}
	return ts, admin, anna
}

func TestMemberSeesSpaceOutsiderDoesNot(t *testing.T) {
	ts, _, anna := setupHaus(t)
	resp, body := get(t, anna, ts.URL+"/s/haus")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Alles ums Haus") {
		t.Errorf("member: %d", resp.StatusCode)
	}
	_, home := get(t, anna, ts.URL+"/")
	if !strings.Contains(home, `href="/s/haus"`) {
		t.Error("home does not list the space")
	}
	outsider := ts.client(t, "ben", "Ben", false)
	if resp, _ := get(t, outsider, ts.URL+"/s/haus"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("outsider: %d", resp.StatusCode)
	}
	if _, home := get(t, outsider, ts.URL+"/"); strings.Contains(home, `href="/s/haus"`) {
		t.Error("outsider sees the space")
	}
}

func TestEditorCannotOpenSettings(t *testing.T) {
	ts, _, anna := setupHaus(t)
	if resp, _ := get(t, anna, ts.URL+"/s/haus/settings"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("settings for editor: %d", resp.StatusCode)
	}
}

func TestOwnerManagesMembers(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"anna"}, "role": {"owner"}})
	anna := ts.client(t, "anna", "Anna", false)
	resp := ts.postForm(t, anna, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("owner add: %d", resp.StatusCode)
	}
	oma := ts.client(t, "oma", "Oma", false)
	if resp, _ := get(t, oma, ts.URL+"/s/haus"); resp.StatusCode != http.StatusOK {
		t.Errorf("oma: %d", resp.StatusCode)
	}
}

func TestRemovedGroupMemberLosesAccessImmediately(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	if resp, _ := get(t, anna, ts.URL+"/s/haus"); resp.StatusCode != http.StatusOK {
		t.Fatalf("before: %d", resp.StatusCode)
	}
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"remove"}, "group": {"familie"}, "sub": {"anna"}})
	if resp, _ := get(t, anna, ts.URL+"/s/haus"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("after removal: %d", resp.StatusCode)
	}
}

func TestOnlyAdminsManageGroupsAndSpaces(t *testing.T) {
	ts, _, anna := setupHaus(t)
	resp := ts.postForm(t, anna, "/", "/admin/groups", url.Values{"action": {"create"}, "group": {"hack"}})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("groups: %d", resp.StatusCode)
	}
	resp = ts.postForm(t, anna, "/", "/admin/spaces/new", url.Values{"name": {"Hack"}})
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("spaces: %d", resp.StatusCode)
	}
}

func TestSettingsEscapesNames(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"<script>x</script>"}})
	_, body := get(t, admin, ts.URL+"/")
	if strings.Contains(body, "<script>x</script>") {
		t.Error("space name not escaped")
	}
}
