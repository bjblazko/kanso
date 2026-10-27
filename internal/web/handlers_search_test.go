// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/url"
	"strings"
	"testing"

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
	idx := buildIndex(ts.deps.Wiki)
	if hits := idx.Search("wasser", func(string) bool { return true }, 5); len(hits) != 1 {
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
