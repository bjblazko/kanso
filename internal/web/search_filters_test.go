// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

// tagged makes a plain page with tags and text.
func (ts *testServer) tagged(t *testing.T, title, body string, tags ...string) string {
	t.Helper()
	id := ts.createPage(t, ts.client(t, "admin", "Admin", true), "haus", "", title)
	_ = ts.deps.Wiki.SaveWith(store.Author{Name: "T", Email: "t@x"}, id, title, body, func(m *wiki.PageMeta) { m.Tags = tags })
	return id
}

func TestSearchGroupsRecipesWithPhotos(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	curry := ts.makeRecipe(t, admin, "haus", "Kartoffelcurry", "![T](attachments/teller.jpg)\n\n"+curryBody, 4, "deftig")
	page := ts.createPage(t, admin, "haus", "", "Kartoffelkeller")
	_, body := get(t, anna, ts.URL+"/search?q=kartoffel")
	recipes := between(body, `<section class="results-recipes"`, `</section>`)
	pages := between(body, `<section class="results-pages"`, `</section>`)
	if !strings.Contains(recipes, `href="/p/`+curry+`"`) || !strings.Contains(recipes, `src="/p/`+curry+`/attachments/teller.jpg"`) ||
		!strings.Contains(recipes, `href="/recipes?q=kartoffel"`) {
		t.Fatalf("recipes:\n%s", recipes)
	}
	if !strings.Contains(pages, `href="/p/`+page+`"`) || strings.Contains(pages, curry) {
		t.Fatalf("pages:\n%s", pages)
	}
}

func TestSearchTagFilters(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	curry := ts.makeRecipe(t, admin, "haus", "Kartoffelcurry", curryBody, 4, "deftig", "vegan")
	eintopf := ts.makeRecipe(t, admin, "haus", "Kartoffeleintopf", curryBody, 4, "deftig")
	keller := ts.tagged(t, "Kartoffelkeller", "Lager", "keller")
	_, body := get(t, anna, ts.URL+"/search?q=kartoffel")
	filters := between(body, `<nav class="tag-filter"`, `</nav>`)
	for _, want := range []string{`href="/search?q=kartoffel&amp;tag=deftig"`, `href="/search?q=kartoffel&amp;tag=keller"`, `href="/search?q=kartoffel&amp;type=recipe"`} {
		if !strings.Contains(filters, want) {
			t.Errorf("filters lack %s", want)
		}
	}
	_, body = get(t, anna, ts.URL+"/search?q=kartoffel&tag=deftig&tag=vegan")
	if main := between(body, "<main", "</main>"); !strings.Contains(main, curry) || strings.Contains(main, eintopf) || strings.Contains(main, keller) {
		t.Fatalf("two tags:\n%s", main)
	}
	_, body = get(t, anna, ts.URL+"/search?tag=keller")
	if main := between(body, "<main", "</main>"); !strings.Contains(main, keller) || strings.Contains(main, curry) {
		t.Fatal("a tag alone does not list its pages")
	}
	_, body = get(t, anna, ts.URL+"/search?q=kartoffel&type=recipe")
	if main := between(body, "<main", "</main>"); strings.Contains(main, keller) || !strings.Contains(main, eintopf) {
		t.Fatal("only recipes")
	}
}

func TestPageTagsLinkToFilteredSearch(t *testing.T) {
	ts, _, anna := setupHaus(t)
	id := ts.tagged(t, "Heizung", "Filter", "wartung")
	_, body := get(t, anna, ts.URL+"/p/"+id)
	if !strings.Contains(body, `href="/search?tag=wartung"`) {
		t.Fatal("page tag does not filter the search")
	}
}
