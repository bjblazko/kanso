// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

func TestRecipeToolbarHasCook(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	for _, c := range []*http.Client{admin, anna} {
		_, body := get(t, c, ts.URL+"/p/"+id)
		if !strings.Contains(toolbarOf(body), `href="/recipes/`+id+`/cook"`) {
			t.Errorf("no cook action:\n%s", toolbarOf(body))
		}
	}
	plain := ts.createPage(t, admin, "haus", "", "Heizung")
	if _, body := get(t, admin, ts.URL+"/p/"+plain); strings.Contains(body, "/cook") {
		t.Error("plain page offers cooking")
	}
}

func TestRecipeMetaLine(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "süß", "deftig")
	_ = ts.deps.Wiki.SaveWith(store.Author{Name: "T", Email: "t@x"}, id, "Curry", curryBody, func(m *wiki.PageMeta) {
		m.Source = "https://beispiel.de/curry"
	})
	_, body := get(t, admin, ts.URL+"/p/"+id)
	meta := between(body, `<p class="recipe-meta">`, `</p>`)
	for _, want := range []string{"4 Portionen", "40 Min.", `href="https://beispiel.de/curry" rel="noopener noreferrer"`,
		`href="/recipes?tag=s%C3%BC%C3%9F"`, `href="/recipes?tag=deftig"`} {
		if !strings.Contains(meta, want) {
			t.Errorf("meta lacks %s:\n%s", want, meta)
		}
	}
	_ = ts.deps.Wiki.SaveWith(store.Author{Name: "T", Email: "t@x"}, id, "Curry", curryBody, func(m *wiki.PageMeta) { m.Source = "Oma Erna" })
	_, body = get(t, admin, ts.URL+"/p/"+id)
	if meta = between(body, `<p class="recipe-meta">`, `</p>`); !strings.Contains(meta, "Oma Erna") || strings.Contains(meta, `href="Oma`) {
		t.Errorf("text source:\n%s", meta)
	}
}

func TestSearchFindsTag(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "kinderessen")
	_, body := get(t, admin, ts.URL+"/search?q="+url.QueryEscape("kinderessen"))
	if !strings.Contains(body, `href="/p/`+id+`"`) {
		t.Error("tag not searchable")
	}
}
