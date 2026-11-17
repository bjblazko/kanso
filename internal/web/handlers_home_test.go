// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// summary is the start page's summary section, unescaped.
func summary(t *testing.T, c *http.Client, u string) (int, string) {
	t.Helper()
	resp, body := get(t, c, u)
	return resp.StatusCode, htmlUnescape(between(body, `<header class="screen-head"`, "</main>"))
}

func TestHomeShowsOthersChanges(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Keller")
	code, sum := summary(t, admin, ts.URL+"/")
	if code != http.StatusOK {
		t.Fatalf("%d", code)
	}
	for _, want := range []string{"Keller", "1 neue Seite", `id="report-data"`, "seit deinem letzten Besuch", "Anna"} {
		if !strings.Contains(sum, want) {
			t.Errorf("summary lacks %q", want)
		}
	}
}

func TestHomeHidesOwnUnlessAsked(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Keller")
	if _, sum := summary(t, anna, ts.URL+"/"); strings.Contains(sum, "Keller") || !strings.Contains(sum, "nichts passiert") {
		t.Error("own change shown by default")
	}
	if _, sum := summary(t, anna, ts.URL+"/?own=1"); !strings.Contains(sum, "Keller") || !strings.Contains(sum, "Eigene ausblenden") {
		t.Error("own change missing with own=1")
	}
}

func TestHomeNeverShowsUnreadable(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Tagebuch")
	ts.note(t, "anna", "Geheim")
	boss := otherAdmin(t, ts)
	for _, u := range []string{"/", "/?range=30d&own=1"} {
		_, sum := summary(t, boss, ts.URL+u)
		if strings.Contains(sum, "Tagebuch") || strings.Contains(sum, "Geheim") || strings.Contains(sum, "Anna") {
			t.Errorf("%s: unreadable change shows", u)
		}
		if !strings.Contains(sum, "nichts passiert") {
			t.Errorf("%s: unreadable changes counted", u)
		}
	}
}

func TestHomeUnknownRangeFallsBack(t *testing.T) {
	ts, _, anna := setupHaus(t)
	code, sum := summary(t, anna, ts.URL+"/?range=1y&own=yes")
	if code != http.StatusOK || !strings.Contains(sum, `<option value="" selected>seit deinem letzten Besuch`) || !strings.Contains(sum, "Eigene einblenden") {
		t.Errorf("%d: no fallback", code)
	}
}

func TestHomeRangeDoesNotMoveVisit(t *testing.T) {
	ts, _, anna := setupHaus(t)
	path := filepath.Join(ts.deps.Cfg.StateDir, "visits.json")
	old := time.Now().Add(-72 * time.Hour).UTC().Format(time.RFC3339)
	before := `{"anna":{"previous":"` + old + `","last":"` + old + `"}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	get(t, anna, ts.URL+"/?range=24h")
	if b, _ := os.ReadFile(path); string(b) != before {
		t.Errorf("range moved the visit: %s", b)
	}
	get(t, anna, ts.URL+"/")
	if b, _ := os.ReadFile(path); string(b) == before {
		t.Error("a plain visit did not move it")
	}
}

func TestHomeGitErrorStillRenders(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.srv.scanRepo = func() (store.Scan, error) { return store.Scan{}, errors.New("broken") }
	code, sum := summary(t, anna, ts.URL+"/")
	if code != http.StatusOK || !strings.Contains(sum, "Historie gerade nicht lesbar") {
		t.Errorf("git error: %d", code)
	}
}

// today is the start page's "Heute" section, unescaped.
func today(t *testing.T, c *http.Client, u string) string {
	t.Helper()
	_, body := get(t, c, u)
	return htmlUnescape(between(body, `<section class="home-today"`, "</section>"))
}

func TestHomeTodayShowsMealAndDueTask(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	plan := ts.familyPlan(t)
	ts.planMeal(t, plan, day(0), "", "Linsensuppe", 4)
	ts.planMeal(t, plan, day(1), "", "Morgensuppe", 4)
	createTask(t, ts, anna, url.Values{"title": {"Hof fegen"}, "board": {fam}, "due": {day(0)}})
	createTask(t, ts, anna, url.Values{"title": {"Steuer"}, "board": {fam}, "due": {day(-2)}})
	createTask(t, ts, anna, url.Values{"title": {"Später"}, "board": {fam}, "due": {day(3)}})
	sec := today(t, anna, ts.URL+"/")
	for _, want := range []string{"Linsensuppe", `href="/mealplan/` + plan, "Hof fegen", "Steuer", "überfällig"} {
		if !strings.Contains(sec, want) {
			t.Errorf("today lacks %q", want)
		}
	}
	if strings.Contains(sec, "Morgensuppe") || strings.Contains(sec, "Später") {
		t.Error("today shows tomorrow's meal or a later task")
	}
	if strings.Index(sec, "Steuer") > strings.Index(sec, "Hof fegen") {
		t.Error("overdue task not first")
	}
}

func TestHomeTodayHidesOtherBoards(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	admin := ts.client(t, "admin", "Admin", true)
	_, a := createTask(t, ts, admin, url.Values{"title": {"Geheimsache"}, "due": {day(0)}})
	if a.ID == "" {
		t.Fatalf("admin task not created: %+v", a)
	}
	if _, body := get(t, anna, ts.URL+"/"); strings.Contains(body, "Geheimsache") || strings.Contains(body, `class="home-today"`) {
		t.Error("anna sees admin's private task or an empty today section")
	}
}

func TestStartPageIsOverall(t *testing.T) {
	ts, _, anna := setupHaus(t)
	_, body := get(t, anna, ts.URL+"/")
	expectHead(t, body, "Was passiert ist", `name="range"`, "Eigene einblenden")
	rail := between(body, `<nav class="rail"`, "</nav>")
	if !strings.Contains(rail, `<a href="/" aria-current="true">`) || !strings.Contains(rail, "<span>Start</span>") {
		t.Errorf("rail: %s", rail)
	}
	if main := between(body, "<main", "</main>"); strings.Contains(main, `href="/s/haus"`) || strings.Contains(body, `class="sidenav"`) {
		t.Error("the start page shows the wiki")
	}
}

func TestWikiListsSpacesOnly(t *testing.T) {
	ts, _, anna := setupHaus(t)
	_, body := get(t, anna, ts.URL+"/wiki")
	if !strings.Contains(body, "<h1>Wiki-Bereiche</h1>") {
		t.Error("/wiki has another title")
	}
	if !strings.Contains(body, `href="/s/haus"`) || strings.Contains(body, "home-summary") || strings.Contains(body, "d3.min.js") {
		t.Error("/wiki is not just the spaces")
	}
	if b, _ := os.ReadFile(filepath.Join(ts.deps.Cfg.StateDir, "visits.json")); strings.Contains(string(b), `"anna"`) {
		t.Errorf("/wiki recorded a visit: %s", b)
	}
	if rail := between(body, `<nav class="rail"`, "</nav>"); !strings.Contains(rail, `<a href="/wiki" aria-current="true">`) {
		t.Errorf("rail: %s", rail)
	}
}
