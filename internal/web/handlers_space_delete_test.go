// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// postDelete sends the delete form of space haus with name and returns status and body.
func postDelete(t *testing.T, ts *testServer, c *http.Client, slug, name string) (*http.Response, string) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/s/"+slug+"/delete")
	form := url.Values{"name": {name}, "csrf": {between(page, `<meta name="csrf" content="`, `"`)}}
	req, _ := http.NewRequest("POST", ts.URL+"/s/"+slug+"/delete", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestDeleteSpaceRights(t *testing.T) {
	ts, _, anna := setupHaus(t)
	if code := status(t, anna, ts.URL+"/s/haus/delete"); code != http.StatusForbidden {
		t.Errorf("editor GET: %d", code)
	}
	if resp, _ := postDelete(t, ts, anna, "haus", "Haus"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("editor POST: %d", resp.StatusCode)
	}
	ben := ts.client(t, "ben", "Ben", false)
	if code := status(t, ben, ts.URL+"/s/haus/delete"); code != http.StatusNotFound {
		t.Errorf("outsider: %d", code)
	}
}

func TestDeleteSpaceConfirmShowsCounts(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	ts.createPage(t, admin, "haus", "", "Keller")
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	ts.planMeal(t, ts.familyPlan(t), "2026-11-10", curry, "Curry", 4)
	resp, body := get(t, admin, ts.URL+"/s/haus/delete")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	for _, want := range []string{"2 Seiten, davon 1 Rezepte, 0 Anhänge", "1 Kochplan-Einträge", `name="name"`} {
		if !strings.Contains(htmlUnescape(body), want) {
			t.Errorf("confirmation lacks %q", want)
		}
	}
}

func TestDeleteSpaceWrongName(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	ts.createPage(t, admin, "haus", "", "Keller")
	resp, body := postDelete(t, ts, admin, "haus", "Hau")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "Der Name stimmt nicht.") {
		t.Errorf("wrong name: %d", resp.StatusCode)
	}
	if code := status(t, admin, ts.URL+"/s/haus"); code != http.StatusOK {
		t.Errorf("space gone: %d", code)
	}
}

func TestDeleteSpaceLocked(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Keller")
	get(t, anna, ts.URL+"/p/"+id+"/edit")
	resp, body := postDelete(t, ts, admin, "haus", "Haus")
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, "Anna bearbeitet gerade") {
		t.Errorf("locked: %d", resp.StatusCode)
	}
	if code := status(t, admin, ts.URL+"/s/haus"); code != http.StatusOK {
		t.Errorf("space gone: %d", code)
	}
}

func TestDeleteSpace(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Kellerregal")
	curry := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	plan := ts.familyPlan(t)
	ts.planMeal(t, plan, "2026-11-10", curry, "Curry", 4)
	resp, _ := postDelete(t, ts, admin, "haus", " Haus ")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/wiki?deleted=Haus" {
		t.Fatalf("delete: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	for _, u := range []string{"/s/haus", "/p/" + id} {
		if code := status(t, admin, ts.URL+u); code != http.StatusNotFound {
			t.Errorf("%s: %d", u, code)
		}
	}
	if _, body := get(t, admin, ts.URL+"/search?q=Kellerregal"); strings.Contains(body, `href="/p/`+id+`"`) {
		t.Error("search still finds the page")
	}
	_, week := get(t, anna, ts.URL+"/mealplan/"+plan+"?week=2026-W46")
	if !strings.Contains(week, "Curry") || strings.Contains(week, `href="/p/`+curry+`"`) {
		t.Error("meal plan: title missing or still linked")
	}
	_, home := get(t, admin, ts.URL+"/wiki?deleted=Haus")
	if !strings.Contains(htmlUnescape(home), "Bereich „Haus“ gelöscht.") || strings.Contains(home, `href="/s/haus"`) {
		t.Error("home: no notice or space still listed")
	}
}

func TestDeleteEmptySpaceNeedsNoName(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	if resp, _ := postDelete(t, ts, admin, "haus", ""); resp.StatusCode != http.StatusSeeOther {
		t.Errorf("empty space: %d", resp.StatusCode)
	}
}

func TestSettingsLinkToDelete(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	if _, body := get(t, admin, ts.URL+"/s/haus/settings"); !strings.Contains(body, `href="/s/haus/delete"`) {
		t.Error("settings lack the delete link")
	}
}
