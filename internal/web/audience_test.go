// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
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
	want := "Sichtbar für dich, Admin, Oma</span>"
	for _, path := range []string{"/", "/s/haus", "/p/" + id} {
		if _, body := get(t, anna, ts.URL+path); !strings.Contains(body, want) {
			t.Errorf("%s lacks %q", path, want)
		}
	}
	if _, body := get(t, admin, ts.URL+"/s/haus"); !strings.Contains(body, "Sichtbar für dich, Anna, Oma</span>") {
		t.Error("admin view lacks the members")
	}
}

func TestNewSpaceIsOnlyForItsCreator(t *testing.T) {
	ts := newTestServer(t)
	admin := ts.client(t, "admin", "Admin", true)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Leer"}})
	_, body := get(t, admin, ts.URL+"/s/leer")
	if !strings.Contains(body, "Nur für dich sichtbar") || strings.Contains(body, "Admins") {
		t.Error("a new space does not say only you")
	}
	other := ts.client(t, "boss", "Boss", true)
	if code := status(t, other, ts.URL+"/s/leer"); code != http.StatusNotFound {
		t.Errorf("another admin reads it: %d", code)
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

// Where a page says who can see it, whoever may change that gets a link to do so.
func TestAudienceLinksToSharingForOwners(t *testing.T) {
	ts, admin, anna := setupHaus(t) // anna is editor via familie, admin is owner everywhere
	id := ts.createPage(t, anna, "haus", "", "Garten")
	links := map[string]string{"/s/haus": `tb-audience" href="/s/haus/settings"`, "/p/" + id: `tb-audience" href="/p/` + id + `/share"`}
	for _, path := range []string{"/s/haus", "/p/" + id} {
		if _, body := get(t, admin, ts.URL+path); !strings.Contains(body, links[path]) {
			t.Errorf("owner misses the link on %s", path)
		}
		if _, body := get(t, anna, ts.URL+path); strings.Contains(body, "Freigabe ändern") {
			t.Errorf("editor gets the link on %s", path)
		}
	}
}

func TestBoardAudienceLinksAdminsToBoards(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", true) // admin and member of familie
	ben := ts.client(t, "ben", "Ben", false)
	fam := familyBoardID(t, ts)
	if _, body := get(t, anna, ts.URL+"/tasks/b/"+fam); !strings.Contains(body, `tb-audience" href="/admin/groups"`) {
		t.Fatal("admin misses the link")
	}
	if _, body := get(t, ben, ts.URL+"/tasks/b/"+fam); strings.Contains(body, "Freigabe ändern") {
		t.Fatal("member without admin rights gets the link")
	}
	if _, body := get(t, anna, ts.URL+"/tasks"); strings.Contains(body, "Freigabe ändern") {
		t.Fatal("private board offers sharing")
	}
}
