// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/ai"
)

const ldPage = `<html><head><script type="application/ld+json">{"@type":"Recipe","name":"Kartoffelcurry","recipeYield":"4","totalTime":"PT40M",
"recipeIngredient":["800 g Kartoffeln","2 Zwiebeln"],"recipeInstructions":[{"@type":"HowToStep","text":"Kochen."}]}</script></head><body></body></html>`

func recipeSite(t *testing.T, html string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/rezept"
}

func TestImportFromJSONLDThenCreate(t *testing.T) {
	ts, _, anna := setupHaus(t)
	site := recipeSite(t, ldPage)
	_, form := get(t, anna, ts.URL+"/recipes/import")
	if !strings.Contains(form, `value="haus"`) || !strings.Contains(form, "nur Seiten mit Rezeptdaten") {
		t.Fatal("import form")
	}
	resp, body := ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {site}, "space": {"haus"}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Kartoffelcurry") || !strings.Contains(body, "800 · g · Kartoffeln") {
		t.Fatalf("%d", resp.StatusCode)
	}
	form2 := url.Values{"space": {"haus"}, "title": {between(body, `name="title" value="`, `"`)}, "servings": {"4"}, "time": {"40"},
		"source": {site}, "body": {htmlUnescape(between(body, `name="body" rows="16">`, `</textarea>`))}}
	resp = ts.postForm(t, anna, "/recipes/import", "/recipes/import/create", form2)
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasSuffix(loc, "/edit") {
		t.Fatalf("%d %q", resp.StatusCode, loc)
	}
	p, _ := ts.deps.Wiki.ByID(strings.TrimSuffix(strings.TrimPrefix(loc, "/p/"), "/edit"))
	if !p.Meta.IsRecipe() || p.Meta.Servings != 4 || p.Meta.Time != 40 || p.Meta.Source != site || !strings.Contains(p.Body, "- 800 g Kartoffeln") {
		t.Fatalf("%+v\n%s", p.Meta, p.Body)
	}
}

func TestImportWithoutDataAndWithoutAI(t *testing.T) {
	ts, _, anna := setupHaus(t)
	site := recipeSite(t, "<html><body><p>500 g Mehl</p></body></html>")
	resp, body := ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {site}, "space": {"haus"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "keine Rezeptdaten") {
		t.Fatalf("%d", resp.StatusCode)
	}
	resp, body = ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {"ftp://example.org/x"}, "space": {"haus"}})
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "lässt sich nicht abrufen") {
		t.Fatalf("ftp %d", resp.StatusCode)
	}
}

func TestImportWithAI(t *testing.T) {
	ts, _, anna := setupHaus(t)
	base := fakeAI(t, "# Omas Apfelkuchen\nPortionen: 12\n\n## Zutaten\n\n- 500 g Äpfel\n\n## Zubereitung\n\n1. Backen.")
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: base, Model: "m"})
	_, form := get(t, anna, ts.URL+"/recipes/import")
	if !strings.Contains(form, "127.0.0.1") {
		t.Error("form does not say where text goes")
	}
	site := recipeSite(t, "<html><body><h1>Omas Apfelkuchen</h1><p>500 g Äpfel</p></body></html>")
	resp, body := ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {site}, "space": {"haus"}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Omas Apfelkuchen") || !strings.Contains(body, "500 · g · Äpfel") || !strings.Contains(body, "mit KI") {
		t.Fatalf("%d", resp.StatusCode)
	}
}

func TestImportCreateNeedsWrite(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	resp := ts.postForm(t, anna, "/recipes/import", "/recipes/import/create",
		url.Values{"space": {"geheim"}, "title": {"X"}, "body": {"## Zutaten\n- 1 Ei\n"}})
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("%d", resp.StatusCode)
	}
}
