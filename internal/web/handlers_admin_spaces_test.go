// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// otherAdmin is an admin who is no member of Haus (setupHaus makes admin its owner).
func otherAdmin(t *testing.T, ts *testServer) *http.Client {
	t.Helper()
	return ts.client(t, "boss", "Boss", true)
}

func TestAdminListsAllSpaces(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.createPage(t, admin, "haus", "", "Keller")
	boss := otherAdmin(t, ts)
	resp, body := get(t, boss, ts.URL+"/admin/spaces")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"Haus", "Sichtbar für Admin, Anna", "1 Seiten", `href="/s/haus/settings"`} {
		if !strings.Contains(body, want) {
			t.Errorf("list lacks %q", want)
		}
	}
	if strings.Contains(body, `href="/s/haus"`) {
		t.Error("list links into a space the admin cannot read")
	}
	if code := status(t, anna, ts.URL+"/admin/spaces"); code != http.StatusForbidden {
		t.Errorf("non-admin: %d", code)
	}
}

func TestAdminManagesSpaceWithoutReadingIt(t *testing.T) {
	ts, _, _ := setupHaus(t)
	boss := otherAdmin(t, ts)
	if _, body := get(t, boss, ts.URL+"/s/haus/settings"); !strings.Contains(body, `href="/admin/spaces"`) || strings.Contains(body, `href="/s/haus"`) {
		t.Error("settings lead back into a space the admin cannot read")
	}
	for path, want := range map[string]int{"/s/haus": http.StatusNotFound, "/s/haus/settings": http.StatusOK, "/s/haus/delete": http.StatusOK} {
		if code := status(t, boss, ts.URL+path); code != want {
			t.Errorf("%s: %d, want %d", path, code, want)
		}
	}
	ts.client(t, "oma", "Oma", false)
	resp := ts.postForm(t, boss, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Errorf("admin cannot change members: %d", resp.StatusCode)
	}
}

func TestAdminTakesOverSpaceNobodyCanRead(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	for range 2 { // admin (owner) and the group familie
		ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings", url.Values{"action": {"remove"}, "index": {"0"}})
	}
	boss := otherAdmin(t, ts)
	_, body := get(t, boss, ts.URL+"/admin/spaces")
	if !strings.Contains(body, "Niemand hat Zugriff") || !strings.Contains(body, "Mich als Besitzer eintragen") {
		t.Fatalf("orphan not offered:\n%s", between(body, "<main", "</main>"))
	}
	resp := ts.postForm(t, boss, "/admin/spaces", "/admin/spaces", url.Values{"action": {"claim"}, "space": {"haus"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("claim: %d", resp.StatusCode)
	}
	if code := status(t, boss, ts.URL+"/s/haus"); code != http.StatusOK {
		t.Errorf("after claiming: %d", code)
	}
}

func TestAdminCannotTakeOverSpaceWithMembers(t *testing.T) {
	ts, _, _ := setupHaus(t)
	boss := otherAdmin(t, ts)
	resp := ts.postForm(t, boss, "/admin/spaces", "/admin/spaces", url.Values{"action": {"claim"}, "space": {"haus"}})
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("claim on a space with members: %d", resp.StatusCode)
	}
	if code := status(t, boss, ts.URL+"/s/haus"); code != http.StatusNotFound {
		t.Errorf("admin reads it: %d", code)
	}
}
