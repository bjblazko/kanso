// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
)

func saveForm(t *testing.T, ts *testServer, c *http.Client, id string, extra url.Values) int {
	t.Helper()
	_, csrf, lock := openEditor(t, ts, c, id)
	form := url.Values{"title": {"Curry"}, "body": {curryBody}, "lock": {lock}, "csrf": {csrf}}
	for k, v := range extra {
		form[k] = v
	}
	req, _ := http.NewRequest("POST", ts.URL+"/p/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func TestEditorShowsRecipeFields(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "deftig", "süß")
	body, _, _ := openEditor(t, ts, admin, id)
	for _, want := range []string{`name="meta" value="1"`, `id="recipe" name="recipe" value="1" checked`,
		`<fieldset class="recipe-fields"`, `name="servings" value="4"`, `name="tags" value="deftig, süß"`, `name="time" value="40"`, `name="source"`} {
		if !strings.Contains(body, want) {
			t.Errorf("editor lacks %s", want)
		}
	}
	plain := ts.createPage(t, admin, "haus", "", "Heizung")
	body, _, _ = openEditor(t, ts, admin, plain)
	if strings.Contains(body, `name="recipe" value="1" checked`) || !strings.Contains(body, `id="recipe"`) {
		t.Error("plain page: checkbox missing or checked")
	}
}

func TestSaveRecipeFields(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Curry")
	code := saveForm(t, ts, admin, id, url.Values{"meta": {"1"}, "recipe": {"1"}, "servings": {"6"},
		"tags": {"Deftig, süß"}, "time": {"30"}, "source": {"Oma"}})
	p, _ := ts.deps.Wiki.ByID(id)
	if code != http.StatusSeeOther || !p.Meta.IsRecipe() || p.Meta.Servings != 6 || p.Meta.Time != 30 ||
		p.Meta.Source != "Oma" || !slices.Equal(p.Meta.Tags, []string{"deftig", "süß"}) {
		t.Fatalf("%d %+v", code, p.Meta)
	}
}

func TestSaveUncheckRecipe(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "deftig")
	saveForm(t, ts, admin, id, url.Values{"meta": {"1"}, "servings": {"4"}, "tags": {"deftig"}})
	p, _ := ts.deps.Wiki.ByID(id)
	if p.Meta.Type != "" || p.Meta.Servings != 0 || !slices.Equal(p.Meta.Tags, []string{"deftig"}) || p.Meta.Time != 0 || p.Body != curryBody { // tags belong to the page
		t.Fatalf("%+v", p.Meta)
	}
}

func TestSaveWithoutMetaMarker(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4, "deftig")
	saveForm(t, ts, admin, id, nil)
	p, _ := ts.deps.Wiki.ByID(id)
	if !p.Meta.IsRecipe() || p.Meta.Servings != 4 {
		t.Fatalf("%+v", p.Meta)
	}
}

func TestSaveBadNumbers(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Curry")
	code := saveForm(t, ts, admin, id, url.Values{"meta": {"1"}, "recipe": {"1"}, "servings": {"abc"}, "time": {"-5"}})
	p, _ := ts.deps.Wiki.ByID(id)
	if code != http.StatusSeeOther || !p.Meta.IsRecipe() || p.Meta.Servings != 0 || p.Meta.Time != 0 {
		t.Fatalf("%d %+v", code, p.Meta)
	}
}

func TestRecipePreviewShowsParsed(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Brot")
	md := "## Zutaten\n- 200 g Mehl\n- Salz\n"
	_, plain := postPreview(t, ts, admin, id, md)
	if strings.Contains(plain, "recipe-parsed") {
		t.Error("plain preview shows recognition")
	}
	_, body := postPreviewWith(t, ts, admin, url.Values{"page": {id}, "body": {md}, "recipe": {"1"}})
	if !strings.Contains(body, `<section class="recipe-parsed"`) || !strings.Contains(body, "200 · g · Mehl") ||
		!strings.Contains(body, `class="parsed-none"`) || !strings.Contains(body, "ohne Menge") {
		t.Fatalf("%s", body)
	}
}
