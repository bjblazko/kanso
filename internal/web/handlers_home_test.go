// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
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
	return resp.StatusCode, htmlUnescape(between(body, `<section class="home-summary"`, `<h2 id="spaces"`))
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
