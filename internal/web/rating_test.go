// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestRateRecipe(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, page := get(t, anna, ts.URL+"/p/"+id)
	if !strings.Contains(page, `action="/recipes/`+id+`/rating"`) || !strings.Contains(page, `name="stars" value="4"`) {
		t.Fatal("no stars to set")
	}
	resp := ts.postForm(t, anna, "/p/"+id, "/recipes/"+id+"/rating", url.Values{"stars": {"4"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/p/"+id {
		t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if p, _ := ts.deps.Wiki.ByID(id); p.Meta.Rating != 4 {
		t.Fatalf("rating %d", p.Meta.Rating)
	}
	_, page = get(t, anna, ts.URL+"/p/"+id)
	if !strings.Contains(page, `aria-label="4 von 5 Sternen"`) {
		t.Fatal("rating not shown")
	}
	ts.postForm(t, anna, "/p/"+id, "/recipes/"+id+"/rating", url.Values{"stars": {"4"}}) // the same star again clears
	if p, _ := ts.deps.Wiki.ByID(id); p.Meta.Rating != 0 {
		t.Fatalf("not cleared: %d", p.Meta.Rating)
	}
	if resp := ts.postForm(t, anna, "/p/"+id, "/recipes/"+id+"/rating", url.Values{"stars": {"9"}}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("9 stars: %d", resp.StatusCode)
	}
}

func TestRatingFilters(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	good := ts.makeRecipe(t, admin, "haus", "Gutes Curry", curryBody, 4)
	okay := ts.makeRecipe(t, admin, "haus", "Okayes Curry", curryBody, 4)
	plain := ts.makeRecipe(t, admin, "haus", "Ungetestetes Curry", curryBody, 4)
	ts.postForm(t, admin, "/p/"+good, "/recipes/"+good+"/rating", url.Values{"stars": {"5"}})
	ts.postForm(t, admin, "/p/"+okay, "/recipes/"+okay+"/rating", url.Values{"stars": {"3"}})
	_, body := get(t, anna, ts.URL+"/recipes?stars=4")
	list := listOf(body)
	if !strings.Contains(list, good) || strings.Contains(list, okay) || strings.Contains(list, plain) {
		t.Fatalf("kitchen 4+:\n%s", list)
	}
	if !strings.Contains(list, `aria-label="5 von 5 Sternen"`) {
		t.Error("no stars in the kitchen row")
	}
	if !strings.Contains(between(body, `<nav class="tag-filter"`, `</nav>`), `href="/recipes?stars=5"`) {
		t.Error("no star filter in the kitchen")
	}
	_, body = get(t, anna, ts.URL+"/search?q=curry&stars=3")
	main := between(body, "<main", "</main>")
	if !strings.Contains(main, good) || !strings.Contains(main, okay) || strings.Contains(main, plain) {
		t.Fatal("search 3+")
	}
	if !strings.Contains(main, `href="/search?q=curry&amp;stars=5"`) {
		t.Error("no star filter in the search")
	}
}
