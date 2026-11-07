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

func TestImportNeedsAWritableSpace(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	site := recipeSite(t, ldPage)
	if resp, _ := ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {site}, "space": {"geheim"}}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("%d", resp.StatusCode)
	}
}

var fakeJPEG = []byte("\xff\xd8\xff\xe0\x00\x10JFIF fake photo")

// photoSite serves a recipe page without recipe data that names a photo, and the photo.
func photoSite(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bild.jpeg" {
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(fakeJPEG)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(strings.Replace(ldPage, "<head>", `<head><meta property="og:image" content="/bild.jpeg">`, 1)))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/rezept"
}

func TestImportTakesThePhoto(t *testing.T) {
	ts, _, anna := setupHaus(t)
	site := photoSite(t)
	_, body := ts.postFormBody(t, anna, "/recipes/import", "/recipes/import", url.Values{"url": {site}, "space": {"haus"}})
	if !strings.Contains(body, `src="data:image/jpeg;base64,`) || !strings.Contains(body, `name="place_0"`) || !strings.Contains(body, `<option value="top" selected>`) {
		t.Fatal("no photo with its place in the preview")
	}
	form := url.Values{"space": {"haus"}, "title": {"Kartoffelcurry"}, "servings": {"4"}, "source": {site}, "place_0": {"step-1"},
		"photo_url_0": {htmlUnescape(between(body, `name="photo_url_0" value="`, `"`))},
		"body":        {htmlUnescape(between(body, `name="body" rows="16">`, `</textarea>`))}}
	resp := ts.postForm(t, anna, "/recipes/import", "/recipes/import/create", form)
	id := strings.TrimSuffix(strings.TrimPrefix(resp.Header.Get("Location"), "/p/"), "/edit")
	p, _ := ts.deps.Wiki.ByID(id)
	if p.Meta.Image == "" || !strings.Contains(p.Body, "   ![Kartoffelcurry](attachments/"+p.Meta.Image+")") {
		t.Fatalf("%+v\n%s", p.Meta, p.Body)
	}
	if files := ts.deps.Wiki.ListAttachments(id); len(files) != 1 || files[0] != p.Meta.Image {
		t.Fatalf("attachments %v", files)
	}
	_, list := get(t, anna, ts.URL+"/recipes")
	if !strings.Contains(list, `src="/p/`+id+`/attachments/`+p.Meta.Image+`"`) {
		t.Fatal("no thumbnail in the kitchen")
	}
	_, cook := get(t, anna, ts.URL+"/recipes/"+id+"/cook")
	if !strings.Contains(between(cook, `<ol class="cook-steps">`, `</ol>`), `src="/p/`+id+`/attachments/`+p.Meta.Image+`"`) {
		t.Fatal("no step photo in the cook view")
	}
}

func TestImportWithoutPhotoWhenSkipped(t *testing.T) {
	ts, _, anna := setupHaus(t)
	site := photoSite(t)
	form := url.Values{"space": {"haus"}, "title": {"Curry"}, "source": {site}, "place_0": {"skip"},
		"photo_url_0": {strings.Replace(site, "/rezept", "/bild.jpeg", 1)}, "body": {"## Zutaten\n- 1 Ei\n"}}
	resp := ts.postForm(t, anna, "/recipes/import", "/recipes/import/create", form)
	id := strings.TrimSuffix(strings.TrimPrefix(resp.Header.Get("Location"), "/p/"), "/edit")
	if p, _ := ts.deps.Wiki.ByID(id); p.Meta.Image != "" || len(ts.deps.Wiki.ListAttachments(id)) != 0 {
		t.Fatal("photo taken although skipped")
	}
}

func TestKitchenThumbnailFromBody(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", "![Teller](attachments/teller.jpg)\n\n"+curryBody, 4)
	_, list := get(t, anna, ts.URL+"/recipes")
	if !strings.Contains(list, `src="/p/`+id+`/attachments/teller.jpg"`) {
		t.Fatal("no thumbnail from the first photo in the text")
	}
}

func TestImportPhotoOnTopIsTheKitchenPhoto(t *testing.T) {
	ts, _, anna := setupHaus(t)
	site := photoSite(t)
	img := strings.Replace(site, "/rezept", "/bild.jpeg", 1)
	form := url.Values{"space": {"haus"}, "title": {"Curry"}, "source": {site}, "body": {"## Zutaten\n- 1 Ei\n\n## Zubereitung\n\n1. Kochen.\n"},
		"place_0": {"step-1"}, "photo_url_0": {img}, "place_1": {"top"}, "photo_url_1": {img + "?gross"}}
	resp := ts.postForm(t, anna, "/recipes/import", "/recipes/import/create", form)
	id := strings.TrimSuffix(strings.TrimPrefix(resp.Header.Get("Location"), "/p/"), "/edit")
	p, _ := ts.deps.Wiki.ByID(id)
	if !strings.HasPrefix(p.Body, "![Curry](attachments/"+p.Meta.Image+")") || len(ts.deps.Wiki.ListAttachments(id)) != 2 {
		t.Fatalf("%+v\n%s", p.Meta, p.Body)
	}
}

func TestAIMarkSaysWhereTextGoes(t *testing.T) {
	ts, _, anna := setupHaus(t)
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: "http://127.0.0.1:1/v1", Model: "m"})
	_, form := get(t, anna, ts.URL+"/recipes/import")
	mark := between(form, `<span class="ai-mark"`, `</span>`)
	if !strings.Contains(mark, "KI") || !strings.Contains(mark, "im eigenen Netz") || !strings.Contains(mark, "127.0.0.1") {
		t.Fatalf("mark: %q", mark)
	}
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: "https://8.8.8.8/v1", Model: "m"})
	_, form = get(t, anna, ts.URL+"/recipes/import")
	if mark = between(form, `<span class="ai-mark"`, `</span>`); !strings.Contains(mark, "extern") || !strings.Contains(mark, "8.8.8.8") {
		t.Fatalf("external mark: %q", mark)
	}
	_ = ts.deps.AI.Save(ai.Config{})
	if _, form = get(t, anna, ts.URL+"/recipes/import"); strings.Contains(form, `class="ai-mark"`) {
		t.Fatal("mark without AI")
	}
}
