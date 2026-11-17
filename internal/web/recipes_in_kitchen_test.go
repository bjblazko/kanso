// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

func TestRecipesHiddenFromWikiTree(t *testing.T) {
	ts, _, anna := setupHaus(t)
	page := ts.createPage(t, anna, "haus", "", "Keller")
	curry := ts.makeRecipe(t, anna, "haus", "Curry", curryBody, 4)
	under := ts.createPage(t, anna, "haus", curry, "Gewürzliste") // a plain page below a recipe
	_, body := get(t, anna, ts.URL+"/s/haus")
	nav := between(body, `class="sidenav"`, "</nav>")
	if strings.Contains(nav, `href="/p/`+curry+`"`) {
		t.Error("the spaces bar lists a recipe")
	}
	for _, id := range []string{page, under} {
		if !strings.Contains(nav, `href="/p/`+id+`"`) {
			t.Errorf("the spaces bar lacks page %s", id)
		}
	}
}

func TestRecipePageBelongsToKitchen(t *testing.T) {
	ts, _, anna := setupHaus(t)
	curry := ts.makeRecipe(t, anna, "haus", "Curry", curryBody, 4)
	for _, path := range []string{"/p/" + curry, "/p/" + curry + "/history"} {
		_, body := get(t, anna, ts.URL+path)
		if !strings.Contains(body, `href="/recipes" aria-current="true"`) || strings.Contains(body, `class="sidenav"`) {
			t.Errorf("%s is not shown in the kitchen", path)
		}
	}
	_, body := get(t, anna, ts.URL+"/p/"+curry)
	if crumbs := between(body, `class="breadcrumb"`, "</nav>"); !strings.Contains(crumbs, `href="/recipes"`) || strings.Contains(crumbs, `href="/s/haus"`) {
		t.Errorf("breadcrumb does not lead back to the recipes:\n%s", crumbs)
	}
	page := ts.createPage(t, anna, "haus", "", "Keller")
	if _, body := get(t, anna, ts.URL+"/p/"+page); !strings.Contains(body, `class="sidenav"`) {
		t.Error("an ordinary page lost the spaces bar")
	}
}

func TestSharedRecipeNotInSharedList(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ben := ts.client(t, "ben", "Ben", false)
	if err := ts.deps.Wiki.SetMembers(planAuthor, curry, []access.Member{{User: "ben", Role: access.Viewer}}); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, ben, ts.URL+"/p/"+curry)
	if !strings.Contains(body, "<h1>Curry</h1>") {
		t.Fatal("ben cannot read the shared recipe")
	}
	if _, body := get(t, ben, ts.URL+"/"); strings.Contains(between(body, `class="sidenav"`, "</nav>"), `href="/p/`+curry+`"`) {
		t.Error("a shared recipe is listed under the wiki's shared pages")
	}
}
