// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

func listOf(body string) string { return between(body, `<ul class="recipe-list`, `</ul>`) }

func TestKitchenRail(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	_, body := get(t, admin, ts.URL+"/notes")
	rail := between(body, `<nav class="rail"`, `</nav>`)
	if !strings.Contains(rail, `href="/recipes"`) || !strings.Contains(rail, "Küche") {
		t.Fatalf("rail:\n%s", rail)
	}
	_, body = get(t, admin, ts.URL+"/recipes")
	if rail = between(body, `<nav class="rail"`, `</nav>`); !strings.Contains(rail, `href="/recipes" aria-current="true"`) {
		t.Fatalf("not current:\n%s", rail)
	}
}

func TestRecipesListVisibleOnly(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	a := store.Author{Name: "Anna", Email: "anna@example.org"}
	for _, id := range []string{w.list, w.secret} {
		_ = w.ts.deps.Wiki.SaveWith(a, id, map[string]string{w.list: "Wunschkuchen", w.secret: "Geheimsuppe"}[id], curryBody,
			func(m *wiki.PageMeta) { m.Type = wiki.TypeRecipe })
	}
	_, body := get(t, w.christina, w.ts.URL+"/recipes")
	list := listOf(body)
	if !strings.Contains(list, `href="/p/`+w.list+`"`) || strings.Contains(list, w.secret) || strings.Contains(list, w.books) {
		t.Fatalf("christina:\n%s", list)
	}
	_, body = get(t, w.anna, w.ts.URL+"/recipes")
	if list = listOf(body); !strings.Contains(list, w.secret) || strings.Contains(list, w.books) {
		t.Fatalf("anna:\n%s", list)
	}
}

func TestRecipesFilterTag(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "deftig", "süß")
	eintopf := ts.makeRecipe(t, admin, "haus", "Eintopf", curryBody, 4, "deftig")
	kuchen := ts.makeRecipe(t, admin, "haus", "Kuchen", "## Zutaten\n- 200 g Mehl\n", 4, "süß")
	_, body := get(t, admin, ts.URL+"/recipes?tag=deftig")
	list := listOf(body)
	if !strings.Contains(list, curry) || !strings.Contains(list, eintopf) || strings.Contains(list, kuchen) {
		t.Fatalf("deftig:\n%s", list)
	}
	_, body = get(t, admin, ts.URL+"/recipes?tag=deftig&tag=s%C3%BC%C3%9F")
	if list = listOf(body); !strings.Contains(list, curry) || strings.Contains(list, eintopf) || strings.Contains(list, kuchen) {
		t.Fatalf("both:\n%s", list)
	}
	filters := between(body, `<nav class="tag-filter"`, `</nav>`)
	if !strings.Contains(filters, `aria-pressed="true"`) || !strings.Contains(filters, `href="/recipes?tag=deftig"`) {
		t.Fatalf("filters:\n%s", filters)
	}
}

func TestRecipesSearchIngredient(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "kinderessen")
	kuchen := ts.makeRecipe(t, admin, "haus", "Kuchen", "## Zutaten\n- 200 g Mehl\n", 4)
	for _, q := range []string{"kartoff", "KARTOFFELN", "curr", "kinder"} {
		_, body := get(t, admin, ts.URL+"/recipes?q="+url.QueryEscape(q))
		if list := listOf(body); !strings.Contains(list, curry) || strings.Contains(list, kuchen) {
			t.Errorf("q=%s:\n%s", q, list)
		}
	}
}

func TestRecipesEmpty(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	_, body := get(t, admin, ts.URL+"/recipes")
	if !strings.Contains(body, "Noch keine Rezepte") || !strings.Contains(body, `href="/recipes/new"`) {
		t.Fatal("empty state")
	}
}

func TestNewRecipe(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	_, body := get(t, anna, ts.URL+"/recipes/new")
	if !strings.Contains(body, `value="haus"`) || strings.Contains(body, `value="geheim"`) {
		t.Fatalf("spaces:\n%s", body)
	}
	resp := ts.postForm(t, anna, "/recipes/new", "/recipes/new", url.Values{"space": {"haus"}, "title": {"Curry"}})
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasSuffix(loc, "/edit") {
		t.Fatalf("%d %q", resp.StatusCode, loc)
	}
	p, _ := ts.deps.Wiki.ByID(strings.TrimSuffix(strings.TrimPrefix(loc, "/p/"), "/edit"))
	if !p.Meta.IsRecipe() || p.Meta.Servings != 4 || !strings.Contains(p.Body, "## Zutaten") || !strings.Contains(p.Body, "## Zubereitung") {
		t.Fatalf("%+v\n%s", p.Meta, p.Body)
	}
	if resp = ts.postForm(t, anna, "/recipes/new", "/recipes/new", url.Values{"space": {"geheim"}, "title": {"X"}}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("no write right: %d", resp.StatusCode)
	}
	if resp = ts.postForm(t, anna, "/recipes/new", "/recipes/new", url.Values{"space": {"haus"}, "title": {" "}}); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("empty title: %d", resp.StatusCode)
	}
}

func TestKitchenTagsFromFilesAreNormalised(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "Deftig")
	_, body := get(t, admin, ts.URL+"/recipes?tag=Deftig")
	if !strings.Contains(listOf(body), id) || !strings.Contains(between(body, `<nav class="tag-filter"`, `</nav>`), `aria-pressed="true">deftig`) {
		t.Fatalf("hand-written tag:\n%s", body)
	}
}
