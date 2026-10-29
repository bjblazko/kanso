// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/url"
	"strings"
	"testing"
)

// The space, its pages and the space list say who else can read them: the
// viewer first, then the others by name (members of a group as people), then
// the admins, whom Kanso only knows at their sign-in.
func TestSpaceSaysWhoSeesIt(t *testing.T) {
	ts, admin, anna := setupHaus(t) // anna via group familie
	ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	id := ts.createPage(t, anna, "haus", "", "Garten")
	want := "Sichtbar für dich, Oma und Admins"
	for _, path := range []string{"/", "/s/haus", "/p/" + id} {
		if _, body := get(t, anna, ts.URL+path); !strings.Contains(body, want) {
			t.Errorf("%s lacks %q", path, want)
		}
	}
	if _, body := get(t, admin, ts.URL+"/s/haus"); !strings.Contains(body, "Sichtbar für Anna, Oma und Admins") {
		t.Error("admin view lacks the members")
	}
}

func TestSpaceWithoutMembersIsAdminsOnly(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Leer"}})
	if _, body := get(t, admin, ts.URL+"/s/leer"); !strings.Contains(body, "Sichtbar nur für Admins") {
		t.Error("empty space does not say admins only")
	}
}

func TestNotesSayOnlyYou(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	for _, path := range []string{"/notes", "/notes/new"} {
		if _, body := get(t, anna, ts.URL+path); !strings.Contains(body, "Nur für dich sichtbar") {
			t.Errorf("%s does not say only you", path)
		}
	}
}
