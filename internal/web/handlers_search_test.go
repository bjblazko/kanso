// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/search"
	"github.com/bjblazko/kanso/internal/store"
)

func TestSearchRespectsPermissions(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	if err := ts.deps.Wiki.Save(store.Author{Name: "Admin", Email: "a@x"}, id, "Heizung", "Der Zählerstand steht im Keller."); err != nil {
		t.Fatal(err)
	}
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	secret := ts.createPage(t, admin, "geheim", "", "Zählerstand Tresor")

	_, body := get(t, anna, ts.URL+"/search?q="+url.QueryEscape("zählerst"))
	if !strings.Contains(body, `href="/p/`+id+`"`) {
		t.Error("readable page not found")
	}
	if strings.Contains(body, secret) || strings.Contains(body, "Tresor") {
		t.Error("page from an unreadable space leaked into results")
	}
	_, body = get(t, admin, ts.URL+"/search?q=tresor")
	if !strings.Contains(body, `href="/p/`+secret+`"`) {
		t.Error("admin does not find the page")
	}
}

func TestSearchFindsPagesAfterRestart(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	ts.createPage(t, admin, "haus", "", "Wasserzähler")
	idx := buildIndex(ts.deps.Wiki, ts.deps.Notes)
	if hits := idx.Search("wasser", func(search.Doc) bool { return true }, 5); len(hits) != 1 {
		t.Errorf("rebuilt index: %d hits", len(hits))
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	ts, _, anna := setupHaus(t)
	resp, body := get(t, anna, ts.URL+"/search?q=")
	if resp.StatusCode != 200 || !strings.Contains(body, "<h1>") {
		t.Errorf("empty search: %d", resp.StatusCode)
	}
}

func TestSearchFindsOwnNotesOnly(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	n, _, err := ts.deps.Notes.Create("anna", "", "Zählerstand Keller\n", nil, maxAttachment)
	if err != nil {
		t.Fatal(err)
	}
	link := `href="/notes/` + n.Meta.ID + `"`
	_, body := get(t, anna, ts.URL+"/search?q="+url.QueryEscape("zählerstand"))
	if !strings.Contains(body, link) || !strings.Contains(body, "<h2>Notizen</h2>") {
		t.Error("own note not found")
	}
	ben := ts.client(t, "ben", "Ben", false)
	for name, c := range map[string]*http.Client{"ben": ben, "admin": admin} {
		if _, body := get(t, c, ts.URL+"/search?q=keller"); strings.Contains(body, n.Meta.ID) {
			t.Errorf("%s finds anna's note", name)
		}
	}
	if err := ts.deps.Notes.Delete("anna", n.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, anna, ts.URL+"/search?q=keller"); strings.Contains(body, link) {
		t.Error("deleted note still found")
	}
}
