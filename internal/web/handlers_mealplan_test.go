// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/mealplan"
	"github.com/bjblazko/kanso/internal/store"
)

var planAuthor = store.Author{Name: "Admin", Email: "admin@example.org"}

// familyPlan makes a plan for the group familie (anna is in it, admin is not).
func (ts *testServer) familyPlan(t *testing.T) string {
	t.Helper()
	p, err := ts.deps.MealPlans.Create(planAuthor, "Familienküche", "familie")
	if err != nil {
		t.Fatal(err)
	}
	return p.ID
}

func (ts *testServer) planMeal(t *testing.T, plan, day, recipe, title string, servings int) string {
	t.Helper()
	start, _ := mealplan.ParseWeek("", mustDay(t, day))
	var id string
	err := ts.deps.MealPlans.Update(planAuthor, plan, mealplan.WeekKey(start), func(w *mealplan.Week) error {
		w.Add(day, mealplan.Entry{Recipe: recipe, Title: title, Servings: servings})
		id = w.Days[day][len(w.Days[day])-1].ID
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestMealPlanWeek(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 6)
	ts.planMeal(t, plan, "2026-11-12", "aaaaaaaaaaaa", "Geheimsuppe", 2)
	resp, body := get(t, anna, ts.URL+"/mealplan/"+plan+"?week=2026-W46")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"Familienküche", `href="/p/` + curry + `"`, "6 Portionen", "Geheimsuppe", `href="/mealplan/` + plan + `?week=2026-W45"`,
		`href="/mealplan/` + plan + `?week=2026-W47"`, `href="/mealplan/` + plan + `/add?day=2026-11-15"`, `name="day" value="2026-11-10"`, `href="/recipes"`} {
		if !strings.Contains(body, want) {
			t.Errorf("week lacks %q", want)
		}
	}
	if strings.Contains(body, `href="/p/aaaaaaaaaaaa"`) {
		t.Error("unreadable recipe linked")
	}
	if code := status(t, admin, ts.URL+"/mealplan/"+plan); code != http.StatusNotFound {
		t.Errorf("non-member: %d", code)
	}
	if code := status(t, anna, ts.URL+"/mealplan/"+plan+"?week=abc"); code != http.StatusOK {
		t.Errorf("bad week: %d", code)
	}
}

func TestMealPlanHome(t *testing.T) {
	ts, _, anna := setupHaus(t)
	if _, body := get(t, anna, ts.URL+"/mealplan"); !strings.Contains(body, "Noch kein Kochplan") {
		t.Error("no hint without a plan")
	}
	plan := ts.familyPlan(t)
	if resp, _ := get(t, anna, ts.URL+"/mealplan"); resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/mealplan/"+plan {
		t.Errorf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestMealPlanAddAndRemove(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, body := get(t, anna, ts.URL+"/mealplan/"+plan+"/add?day=2026-11-10")
	if !strings.Contains(body, `value="`+curry+`"`) {
		t.Fatal("recipe not offered")
	}
	resp := ts.postForm(t, anna, "/mealplan/"+plan+"/add?day=2026-11-10", "/mealplan/"+plan+"/add",
		url.Values{"day": {"2026-11-10"}, "recipe": {curry}, "servings": {""}})
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/mealplan/"+plan+"?week=2026-W46") {
		t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	w, _ := ts.deps.MealPlans.Week(plan, "2026-W46")
	e := w.Days["2026-11-10"]
	if len(e) != 1 || e[0].Servings != 4 || e[0].Title != "Curry" {
		t.Fatalf("%+v", e)
	}
	resp = ts.postForm(t, anna, "/mealplan/"+plan+"/remove?week=2026-W46&entry="+e[0].ID, "/mealplan/"+plan+"/remove",
		url.Values{"week": {"2026-W46"}, "entry": {e[0].ID}})
	if w, _ = ts.deps.MealPlans.Week(plan, "2026-W46"); resp.StatusCode != http.StatusSeeOther || len(w.Days) != 0 {
		t.Fatalf("remove %d %+v", resp.StatusCode, w)
	}
}

func TestPlanFromRecipe(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	if _, body := get(t, anna, ts.URL+"/p/"+curry); !strings.Contains(toolbarOf(body), `href="/recipes/`+curry+`/plan"`) {
		t.Error("no Einplanen action")
	}
	resp := ts.postForm(t, anna, "/recipes/"+curry+"/plan", "/recipes/"+curry+"/plan",
		url.Values{"plan": {plan}, "day": {"2026-11-11"}, "servings": {"2"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("%d", resp.StatusCode)
	}
	if w, _ := ts.deps.MealPlans.Week(plan, "2026-W46"); len(w.Days["2026-11-11"]) != 1 || w.Days["2026-11-11"][0].Servings != 2 {
		t.Fatalf("%+v", w)
	}
	if resp := ts.postForm(t, admin, "/recipes/"+curry+"/plan", "/recipes/"+curry+"/plan",
		url.Values{"plan": {plan}, "day": {"2026-11-11"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("non-member: %d", resp.StatusCode)
	}
}

func TestAdminMealPlans(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	resp := ts.postForm(t, admin, "/admin/mealplans", "/admin/mealplans", url.Values{"action": {"create"}, "name": {"Familienküche"}, "group": {"familie"}})
	if resp.StatusCode != http.StatusSeeOther || len(ts.deps.MealPlans.All()) != 1 {
		t.Fatalf("%d", resp.StatusCode)
	}
	if code := status(t, anna, ts.URL+"/admin/mealplans"); code != http.StatusForbidden {
		t.Errorf("anna: %d", code)
	}
	id := ts.deps.MealPlans.All()[0].ID
	ts.postForm(t, admin, "/admin/mealplans", "/admin/mealplans", url.Values{"action": {"delete_confirmed"}, "id": {id}})
	if len(ts.deps.MealPlans.All()) != 0 {
		t.Fatal("not deleted")
	}
}

func TestKitchenTabs(t *testing.T) {
	ts, _, anna := setupHaus(t)
	_, body := get(t, anna, ts.URL+"/recipes")
	if !strings.Contains(body, `href="/mealplan"`) || !strings.Contains(body, `href="/recipes" aria-current="page"`) {
		t.Error("kitchen tabs")
	}
}
