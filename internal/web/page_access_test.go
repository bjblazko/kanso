// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// wishlist is the setup of page sharing: in space haus, "Wunschliste" with
// the child "Bücher" is shared with christina (role), "Geheim" is not.
type wishlist struct {
	ts                     *testServer
	admin, anna, christina *http.Client
	list, books, secret    string
}

func setupWishlist(t *testing.T, role access.Role) wishlist {
	t.Helper()
	ts, admin, anna := setupHaus(t)
	christina := ts.client(t, "christina", "Christina", false)
	w := wishlist{ts: ts, admin: admin, anna: anna, christina: christina}
	w.list = ts.createPage(t, anna, "haus", "", "Wunschliste")
	w.books = ts.createPage(t, anna, "haus", w.list, "Bücher")
	w.secret = ts.createPage(t, anna, "haus", "", "Geheim")
	a := store.Author{Name: "Anna", Email: "anna@example.org"}
	if err := ts.deps.Wiki.SetMembers(a, w.list, []access.Member{{User: "christina", Role: role}}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{w.list, w.secret} {
		if _, err := ts.deps.Wiki.AddAttachment(a, id, "zettel.txt", strings.NewReader("x"), 1<<20); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func status(t *testing.T, c *http.Client, u string) int {
	t.Helper()
	resp, _ := get(t, c, u)
	return resp.StatusCode
}

func TestShareGivesPageAndChildren(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	for _, p := range []string{"/p/" + w.list, "/p/" + w.books, "/p/" + w.list + "/attachments/zettel.txt", "/p/" + w.list + "/history"} {
		if code := status(t, w.christina, w.ts.URL+p); code != http.StatusOK {
			t.Errorf("%s: %d", p, code)
		}
	}
}

func TestShareHidesSiblingsAndSpace(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	for _, p := range []string{"/p/" + w.secret, "/p/" + w.secret + "/attachments/zettel.txt", "/p/" + w.secret + "/history", "/s/haus", "/s/haus/new"} {
		if code := status(t, w.christina, w.ts.URL+p); code != http.StatusNotFound {
			t.Errorf("%s: %d", p, code)
		}
	}
	if _, body := get(t, w.christina, w.ts.URL+"/p/"+w.list); strings.Contains(body, "Geheim") {
		t.Error("the page shows a sibling's title")
	}
}

func TestShareReadOnly(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.list+"/edit"); code != http.StatusForbidden {
		t.Fatalf("edit: %d", code)
	}
}

func TestShareEditorWritesInSubtree(t *testing.T) {
	w := setupWishlist(t, access.Editor)
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.list+"/edit"); code != http.StatusOK {
		t.Fatalf("edit: %d", code)
	}
	child := w.ts.createPage(t, w.christina, "haus", w.list, "Spiele")
	if code := status(t, w.christina, w.ts.URL+"/p/"+child); code != http.StatusOK {
		t.Fatalf("new child: %d", code)
	}
	if resp := w.ts.postForm(t, w.christina, "/p/"+w.list, "/s/haus/new", url.Values{"title": {"Oben"}, "parent": {""}}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("top-level page: %d", resp.StatusCode)
	}
	_, body := get(t, w.christina, w.ts.URL+"/p/"+child)
	if strings.Contains(body, `value="haus/`+w.secret+`"`) || strings.Contains(body, `value="haus/"`) {
		t.Fatalf("move targets outside the share:\n%s", body)
	}
	if !strings.Contains(body, `value="haus/`+w.books+`"`) {
		t.Fatal("move targets miss the shared subtree")
	}
	resp := w.ts.postForm(t, w.christina, "/p/"+child, "/p/"+child+"/move", url.Values{"parent": {w.secret}, "order": {"0"}})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("move outside: %d", resp.StatusCode)
	}
}

func TestSearchOnlyShared(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.christina, w.ts.URL+"/search?q=Geheim")
	if strings.Contains(body, `href="/p/`+w.secret+`"`) {
		t.Fatal("search finds a page outside the share")
	}
	_, body = get(t, w.christina, w.ts.URL+"/search?q=B%C3%BCcher")
	if !strings.Contains(body, `href="/p/`+w.books+`"`) {
		t.Fatalf("search misses the shared child:\n%s", body)
	}
}

func TestPromoteTargetsIncludeSharedWritable(t *testing.T) {
	w := setupWishlist(t, access.Editor)
	n, _, _ := w.ts.deps.Notes.Create("christina", "", "Idee\n", nil, 1<<20)
	_, body := get(t, w.christina, w.ts.URL+"/notes/"+n.Meta.ID+"/promote")
	if !strings.Contains(body, `value="haus/`+w.list+`"`) || strings.Contains(body, `value="haus/`+w.secret+`"`) || strings.Contains(body, `value="haus/"`) {
		t.Fatalf("promote targets:\n%s", body)
	}
}
