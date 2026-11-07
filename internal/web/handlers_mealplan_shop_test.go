// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestPlanShopPreview(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 4)
	ts.planMeal(t, plan, "2026-11-12", curry, "Curry", 4)
	ts.planMeal(t, plan, "2026-11-12", "aaaaaaaaaaaa", "Geheimsuppe", 2)
	resp, body := get(t, anna, ts.URL+"/mealplan/"+plan+"/shop?week=2026-W46&day=2026-11-10&day=2026-11-12")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"800 g Kartoffeln", "800 g Kartoffeln → 1,6 kg Kartoffeln", `name="take" value="e0-0" checked`, `name="take" value="e1-0" checked`, "Geheimsuppe"} {
		if !strings.Contains(body, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
	if strings.Count(body, `<section class="shop-meal"`) != 2 {
		t.Error("expected two meal sections")
	}
	if code := status(t, admin, ts.URL+"/mealplan/"+plan+"/shop?week=2026-W46&day=2026-11-10"); code != http.StatusNotFound {
		t.Errorf("non-member: %d", code)
	}
}

func shopDays(t *testing.T, ts *testServer, c *http.Client, plan string, extra url.Values) *http.Response {
	t.Helper()
	form := url.Values{"week": {"2026-W46"}, "day": {"2026-11-10", "2026-11-12"}, "list": {"new"}, "name": {"Woche"}, "fill": {"fillbbbbbbbb"}}
	for k, v := range extra {
		form[k] = v
	}
	return ts.postForm(t, c, "/mealplan/"+plan+"/shop?week=2026-W46&day=2026-11-10", "/mealplan/"+plan+"/shop", form)
}

func TestPlanShopApplyRemoveSubtract(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	first := ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 4)
	second := ts.planMeal(t, plan, "2026-11-12", curry, "Curry", 4)
	resp := shopDays(t, ts, anna, plan, url.Values{"take": {"e0-0", "e1-0"}})
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(resp.Header.Get("Location"), "/todos/") {
		t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	shopDays(t, ts, anna, plan, url.Values{"take": {"e0-0", "e1-0"}}) // a second tap adds nothing
	lists := ts.shoppingLists(t, "anna")
	if len(lists) != 1 || len(lists[0].Items) != 1 || lists[0].Items[0].Text != "1,6 kg Kartoffeln" || len(lists[0].Items[0].For) != 2 {
		t.Fatalf("%+v", lists)
	}
	w, _ := ts.deps.MealPlans.Week(plan, "2026-W46")
	e, _, _ := w.Find(second)
	if e.Fill == "" || e.List != lists[0].ID {
		t.Fatalf("entry not marked: %+v", e)
	}
	_, body := get(t, anna, ts.URL+"/mealplan/"+plan+"/remove?week=2026-W46&entry="+second)
	if !strings.Contains(body, "Entfernen und Zutaten abziehen") {
		t.Fatal("no offer to subtract")
	}
	ts.postForm(t, anna, "/mealplan/"+plan+"/remove?week=2026-W46&entry="+second, "/mealplan/"+plan+"/remove",
		url.Values{"week": {"2026-W46"}, "entry": {second}, "subtract": {"1"}})
	ts.postForm(t, anna, "/mealplan/"+plan+"/remove?week=2026-W46&entry="+first, "/mealplan/"+plan+"/remove",
		url.Values{"week": {"2026-W46"}, "entry": {first}})
	l := ts.shoppingLists(t, "anna")[0]
	if len(l.Items) != 1 || l.Items[0].Text != "800 g Kartoffeln" || len(l.Items[0].For) != 1 {
		t.Fatalf("after removing: %+v", l.Items)
	}
}

func TestPlanShopSkipsShopped(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 4)
	shopDays(t, ts, anna, plan, url.Values{"day": {"2026-11-10"}, "take": {"e0-0"}})
	_, body := get(t, anna, ts.URL+"/mealplan/"+plan+"/shop?week=2026-W46&day=2026-11-10")
	if strings.Contains(body, `<section class="shop-meal"`) || !strings.Contains(body, "schon auf") {
		t.Fatal("shopped meal offered again")
	}
}
