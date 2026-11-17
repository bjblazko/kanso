// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// cookbookSpace makes space "Rezepte" (admin owns it, familie writes) with one recipe.
func cookbookSpace(t *testing.T, ts *testServer, admin, anna *http.Client) string {
	t.Helper()
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Rezepte"}})
	ts.postForm(t, admin, "/s/rezepte/settings", "/s/rezepte/settings",
		url.Values{"action": {"add"}, "kind": {"group"}, "who": {"familie"}, "role": {"editor"}})
	return ts.makeRecipe(t, anna, "rezepte", "Curry", curryBody, 4)
}

func TestCookbookAdoptedAtStart(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Keller")
	ts.makeRecipe(t, anna, "haus", "Hausbrot", curryBody, 2) // haus holds pages too: no cookbook
	ts.srv.adoptCookbook()
	if _, ok := ts.deps.Spaces.Cookbook(); ok {
		t.Fatal("a space with ordinary pages became the cookbook")
	}
	cookbookSpace(t, ts, admin, anna)
	ts.srv.adoptCookbook()
	if cb, ok := ts.deps.Spaces.Cookbook(); ok {
		t.Errorf("two spaces hold recipes, yet %q became the cookbook", cb.Slug)
	}
}

func TestCookbookAdoptsTheOnlyRecipeSpace(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Keller")
	cookbookSpace(t, ts, admin, anna)
	ts.srv.adoptCookbook()
	if cb, ok := ts.deps.Spaces.Cookbook(); !ok || cb.Slug != "rezepte" {
		t.Errorf("cookbook = %+v, %v", cb, ok)
	}
}

func TestCookbookHiddenFromWiki(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	cookbookSpace(t, ts, admin, anna)
	ts.srv.adoptCookbook()
	if _, body := get(t, anna, ts.URL+"/"); strings.Contains(body, `href="/s/rezepte"`) {
		t.Error("the wiki lists the cookbook")
	}
	if resp, _ := get(t, anna, ts.URL+"/s/rezepte"); resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/recipes" {
		t.Errorf("cookbook space: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if code := status(t, admin, ts.URL+"/s/rezepte/settings"); code != http.StatusOK {
		t.Errorf("settings: %d", code)
	}
	if _, body := get(t, admin, ts.URL+"/recipes"); !strings.Contains(body, `href="/s/rezepte/settings"`) {
		t.Error("the kitchen does not lead the owner to who sees the recipes")
	}
}

func TestNewRecipeGoesToCookbook(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	cookbookSpace(t, ts, admin, anna)
	ts.srv.adoptCookbook()
	if _, body := get(t, anna, ts.URL+"/recipes/new"); strings.Contains(body, `<select id="space"`) {
		t.Error("the form still asks for a space")
	}
	resp := ts.postForm(t, anna, "/recipes/new", "/recipes/new", url.Values{"title": {"Linsen"}, "space": {"haus"}})
	id := strings.TrimSuffix(strings.TrimPrefix(resp.Header.Get("Location"), "/p/"), "/edit")
	if pg, ok := ts.deps.Wiki.ByID(id); !ok || pg.Space != "rezepte" {
		t.Errorf("recipe went to %q (%d %q)", pg.Space, resp.StatusCode, resp.Header.Get("Location"))
	}
	ben := ts.client(t, "ben", "Ben", false)
	if _, body := get(t, ben, ts.URL+"/recipes/new"); !strings.Contains(body, "Du darfst im Kochbuch keine Rezepte anlegen.") {
		t.Error("someone without access is not told")
	}
}

func TestAdminCreatesCookbook(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	if _, body := get(t, anna, ts.URL+"/recipes/new"); !strings.Contains(body, "Noch kein Kochbuch") || strings.Contains(body, `action="/recipes/cookbook"`) {
		t.Error("anna is not told there is no cookbook, or may create one")
	}
	if _, body := get(t, admin, ts.URL+"/recipes/new"); !strings.Contains(body, `action="/recipes/cookbook"`) {
		t.Error("admin is not offered to create the cookbook")
	}
	if resp := ts.postForm(t, anna, "/recipes/new", "/recipes/cookbook", url.Values{}); resp.StatusCode != http.StatusForbidden {
		t.Errorf("anna creates a cookbook: %d", resp.StatusCode)
	}
	resp := ts.postForm(t, admin, "/recipes/new", "/recipes/cookbook", url.Values{})
	cb, ok := ts.deps.Spaces.Cookbook()
	if resp.StatusCode != http.StatusSeeOther || !ok || cb.Name != "Rezepte" {
		t.Errorf("create: %d %+v", resp.StatusCode, cb)
	}
}

func TestAdminMovesCookbook(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	cookbookSpace(t, ts, admin, anna)
	ts.srv.adoptCookbook()
	resp := ts.postForm(t, admin, "/admin/spaces", "/admin/spaces", url.Values{"action": {"cookbook"}, "space": {"haus"}})
	if cb, _ := ts.deps.Spaces.Cookbook(); resp.StatusCode != http.StatusSeeOther || cb.Slug != "haus" {
		t.Errorf("move: %d %q", resp.StatusCode, cb.Slug)
	}
}
