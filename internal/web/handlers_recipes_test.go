// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestCookViewScales(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	resp, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook?servings=6")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"1200 g Kartoffeln", "1½ Dose Kokosmilch", "3 Zwiebeln", "Salz", "Kartoffeln schälen und würfeln.", `data-tick="i0"`, `data-tick="s0"`} {
		if !strings.Contains(body, want) {
			t.Errorf("cook view lacks %q", want)
		}
	}
	if strings.Contains(body, "Kurz und mild") {
		t.Error("free text shown in the cook view")
	}
}

func TestCookViewServingsClamp(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	for raw, want := range map[string]string{"": "4", "0": "1", "-3": "1", "abc": "4", "1000": "99"} {
		_, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook?servings="+url.QueryEscape(raw))
		if got := between(body, `<output class="servings-n">`, `</output>`); got != want {
			t.Errorf("servings=%q: %q, want %s", raw, got, want)
		}
	}
}

func TestCookViewLinks(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	base := "/recipes/" + id + "/cook"
	_, body := get(t, admin, ts.URL+base+"?servings=6")
	if !strings.Contains(body, `href="`+base+`?servings=5"`) || !strings.Contains(body, `href="`+base+`?servings=7"`) ||
		!strings.Contains(body, `href="/p/`+id+`"`) {
		t.Fatalf("links missing")
	}
	if _, body = get(t, admin, ts.URL+base+"?servings=1"); strings.Contains(body, `?servings=0"`) {
		t.Error("minus below 1")
	}
	if _, body = get(t, admin, ts.URL+base+"?servings=99"); strings.Contains(body, `?servings=100"`) {
		t.Error("plus above 99")
	}
}

func TestCookViewNotRecipe(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plain := ts.createPage(t, admin, "haus", "", "Heizung")
	if code := status(t, admin, ts.URL+"/recipes/"+plain+"/cook"); code != http.StatusNotFound {
		t.Errorf("plain page: %d", code)
	}
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	secret := ts.makeRecipe(t, admin, "geheim", "Tresorsuppe", curryBody, 2)
	if code := status(t, anna, ts.URL+"/recipes/"+secret+"/cook"); code != http.StatusNotFound {
		t.Errorf("unreadable: %d", code)
	}
}

func TestCookViewNoServings(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 0)
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook")
	if !strings.Contains(body, `<output class="servings-n">1</output>`) || !strings.Contains(body, "ohne Portionsangabe") {
		t.Fatal("no hint for a recipe without servings")
	}
}

func TestCookViewNoIngredients(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Leer", "Nur Text.", 2)
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook")
	if !strings.Contains(body, "Kanso findet keine Zutaten") || !strings.Contains(body, `href="/p/`+id+`/edit"`) {
		t.Fatal("no hint with editor link")
	}
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"ben"}, "role": {"viewer"}})
	ben := ts.client(t, "ben", "Ben", false)
	if _, body = get(t, ben, ts.URL+"/recipes/"+id+"/cook"); strings.Contains(body, `/edit"`) {
		t.Error("reader gets an editor link")
	}
	_ = anna
}

func TestCookViewOffersReset(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook")
	if !strings.Contains(toolbarOf(body), `data-reset`) || !strings.Contains(body, "Häkchen entfernen") {
		t.Fatal("no reset action")
	}
}

func TestCookViewRendersInlineMarkdown(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	body := "## Zutaten\n\n- 60 g Röstzwiebeln, *frisch*\n\n## Zubereitung\n\n1. __Hefe auflösen__: Wasser **2 Min. / 37 °C** erwärmen.\n2. <script>alert(1)</script> Kneten.\n"
	id := ts.makeRecipe(t, admin, "haus", "Baguette", body, 1)
	_, page := get(t, admin, ts.URL+"/recipes/"+id+"/cook")
	steps := between(page, `<ol class="cook-steps">`, `</ol>`)
	if !strings.Contains(steps, "<strong>Hefe auflösen</strong>") || !strings.Contains(steps, "<strong>2 Min. / 37 °C</strong>") || strings.Contains(steps, "__Hefe") {
		t.Fatalf("steps:\n%s", steps)
	}
	if strings.Contains(steps, "<script>") || strings.Contains(steps, "<p>") {
		t.Fatalf("unsafe or block markup:\n%s", steps)
	}
	if !strings.Contains(between(page, `<ul class="cook-list">`, `</ul>`), "<em>frisch</em>") {
		t.Fatal("ingredient note not rendered")
	}
}
