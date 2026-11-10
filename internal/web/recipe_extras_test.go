// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

func TestMealPlanShowsRecipePhoto(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", "![Teller](attachments/teller.jpg)\n\n"+curryBody, 4)
	ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 4)
	_, body := get(t, anna, ts.URL+"/mealplan/"+plan+"?week=2026-W46")
	if !strings.Contains(between(body, `<ul class="week-meals">`, `</ul>`), `src="/p/`+curry+`/attachments/teller.jpg"`) {
		t.Fatal("no photo in the meal plan")
	}
}

func TestRecipePageShowsCookedDays(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, plan, "2026-11-03", curry, "Curry", 4)
	ts.planMeal(t, plan, "2026-11-12", curry, "Curry", 2)
	_, body := get(t, anna, ts.URL+"/p/"+curry)
	cooked := between(body, `<section class="cooked"`, `</section>`)
	list := between(cooked, "<ul", "</ul>")
	if strings.Index(list, "12.11.2026") > strings.Index(list, "3.11.2026") {
		t.Fatalf("not latest first:\n%s", list)
	}
	if !strings.Contains(cooked, "1× gekocht, zuletzt am 3.11.2026") || !strings.Contains(cooked, `href="/mealplan/`+plan+`?week=2026-W46#day-2026-11-12"`) ||
		!strings.Contains(cooked, `href="/mealplan/`+plan+`?week=2026-W45#day-2026-11-03"`) {
		t.Fatalf("cooked:\n%s", cooked)
	}
	if _, body = get(t, admin, ts.URL+"/p/"+curry); strings.Contains(body, `<section class="cooked"`) {
		t.Error("someone outside the group sees the plan")
	}
}

func TestTagsOnEveryPage(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	body, _, _ := openEditor(t, ts, admin, id)
	if !strings.Contains(body, `name="tags"`) || strings.Contains(between(body, `<fieldset class="recipe-fields"`, `</fieldset>`), `name="tags"`) {
		t.Fatal("tags field missing or only for recipes")
	}
	saveForm(t, ts, admin, id, url.Values{"meta": {"1"}, "tags": {"Wartung, Keller"}})
	p, _ := ts.deps.Wiki.ByID(id)
	if p.Meta.IsRecipe() || !slices.Equal(p.Meta.Tags, []string{"wartung", "keller"}) {
		t.Fatalf("%+v", p.Meta)
	}
	_, view := get(t, admin, ts.URL+"/p/"+id)
	if !strings.Contains(view, `href="/search?q=wartung"`) {
		t.Fatal("tags of a page do not lead to the search")
	}
	// unticking "Ist ein Rezept" keeps the tags
	_ = ts.deps.Wiki.SaveWith(store.Author{Name: "T", Email: "t@x"}, id, "Heizung", "", func(m *wiki.PageMeta) { m.Type = wiki.TypeRecipe })
	saveForm(t, ts, admin, id, url.Values{"meta": {"1"}, "tags": {"wartung"}})
	if p, _ = ts.deps.Wiki.ByID(id); p.Meta.IsRecipe() || !slices.Equal(p.Meta.Tags, []string{"wartung"}) {
		t.Fatalf("after unticking: %+v", p.Meta)
	}
}
