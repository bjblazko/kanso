// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCookedToday(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, cook := get(t, anna, ts.URL+"/recipes/"+id+"/cook")
	if !strings.Contains(toolbarOf(cook), `action="/recipes/`+id+`/cooked"`) {
		t.Fatal("no Heute gekocht in the cook view")
	}
	for range 2 { // twice the same day counts once
		resp := ts.postForm(t, anna, "/recipes/"+id+"/cook", "/recipes/"+id+"/cooked", url.Values{})
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/recipes/"+id+"/cook" {
			t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
		}
	}
	today := time.Now().Format("2006-01-02")
	p, _ := ts.deps.Wiki.ByID(id)
	if len(p.Meta.Cooked) != 1 || p.Meta.Cooked[0] != today {
		t.Fatalf("%v", p.Meta.Cooked)
	}
	if _, cook = get(t, anna, ts.URL+"/recipes/"+id+"/cook"); !strings.Contains(toolbarOf(cook), "Heute gekocht ✓") {
		t.Error("no sign that today is noted")
	}
	_, page := get(t, anna, ts.URL+"/p/"+id)
	cooked := between(page, `<section class="cooked"`, `</section>`)
	if !strings.Contains(cooked, "1× gekocht") || !strings.Contains(cooked, time.Now().Format("2.1.2006")) {
		t.Fatalf("recipe page:\n%s", cooked)
	}
}

func TestCookedCountsPlanAndHand(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	plan := ts.familyPlan(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, plan, "2026-11-03", id, "Curry", 4)
	ts.planMeal(t, plan, "2099-01-05", id, "Curry", 4) // ahead: planned, not cooked
	ts.postForm(t, anna, "/recipes/"+id+"/cook", "/recipes/"+id+"/cooked", url.Values{})
	_, page := get(t, anna, ts.URL+"/p/"+id)
	cooked := between(page, `<section class="cooked"`, `</section>`)
	if !strings.Contains(cooked, "2× gekocht") || !strings.Contains(cooked, "geplant") || !strings.Contains(cooked, "von Hand") {
		t.Fatalf("%s", cooked)
	}
}

func TestCookedNeedsWrite(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ben := ts.client(t, "ben", "Ben", false) // signed in once, so the space can name him
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings", url.Values{"action": {"add"}, "kind": {"user"}, "who": {"ben"}, "role": {"viewer"}})
	if resp := ts.postForm(t, ben, "/recipes/"+id+"/cook", "/recipes/"+id+"/cooked", url.Values{}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("%d", resp.StatusCode)
	}
	if _, cook := get(t, ben, ts.URL+"/recipes/"+id+"/cook"); strings.Contains(cook, "/cooked") {
		t.Error("reader gets the button")
	}
}
