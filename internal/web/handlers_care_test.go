// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestCarePage(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, anna, ts.URL+"/care")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	for _, want := range []string{"Erste Hilfe – jetzt gleich", `href="/care?after=`, "Wie voll ist dein Akku?", `href="/care/senses"`,
		`href="tel:08001110111"`, `href="tel:112"`, `data-key="care"><a href="/care" aria-current="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	_, empty := get(t, anna, ts.URL+"/care?level=10")
	help, now := strings.Index(empty, `class="care-help"`), strings.Index(empty, `class="care-now"`)
	if help < 0 || help > now || !strings.Contains(empty, "Ein Glas Wasser") {
		t.Errorf("empty battery: help must come first and gentle steps follow")
	}
}

func TestCareExercises(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if _, body := get(t, anna, ts.URL+"/care/senses"); !strings.Contains(body, "<li>Fünf Dinge, die du siehst.</li>") {
		t.Error("senses steps missing")
	}
	if _, body := get(t, anna, ts.URL+"/care/breathe"); !strings.Contains(body, `id="breath-toggle"`) || !strings.Contains(body, "js/breath.js") {
		t.Error("breathing pacer missing")
	}
	if resp, _ := get(t, anna, ts.URL+"/care/nope"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown exercise: %d", resp.StatusCode)
	}
}

func TestCareNote(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, _ := postTaskForm(t, ts, anna, "/care/checkin/note", url.Values{"a0": {"halb voll"}, "a4": {"Nein zum Elternabend"}}, "")
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/notes/") {
		t.Fatalf("save: %d %q", resp.StatusCode, loc)
	}
	n, err := ts.deps.Notes.Get("anna", strings.TrimPrefix(loc, "/notes/"))
	if err != nil || !strings.HasPrefix(n.Body, "Check-in – ") || !strings.Contains(n.Body, "halb voll") ||
		!strings.Contains(n.Body, "Nein zum Elternabend") || strings.Contains(n.Body, "Was kostet dich") {
		t.Fatalf("note %q %v", n.Body, err)
	}
	if resp, _ := postTaskForm(t, ts, anna, "/care/checkin/note", url.Values{}, ""); resp.Header.Get("Location") != "/care/checkin" {
		t.Errorf("empty check-in saved a note: %q", resp.Header.Get("Location"))
	}
	if resp, _ := postTaskForm(t, ts, anna, "/care/senses/note", url.Values{"a0": {"x"}}, ""); resp.StatusCode != http.StatusNotFound {
		t.Errorf("note from a steps exercise: %d", resp.StatusCode)
	}
}

func TestCareMeditation(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if _, body := get(t, anna, ts.URL+"/care"); !strings.Contains(body, `href="/care/m/arrive"`) || !strings.Contains(body, "10 Minuten") {
		t.Error("meditations missing on /care")
	}
	_, body := get(t, anna, ts.URL+"/care/m/arrive")
	for _, want := range []string{`data-minutes="5"`, `value="lyra" checked`, `value="orion"`, `data-clip-lyra="/static/audio/meditation/lyra/de/arrive-01.mp3?v=`,
		`data-clip-orion="/static/audio/meditation/orion/de/arrive-13.mp3?v=`, ">05:00<", "js/meditation.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if resp, _ := get(t, anna, ts.URL+"/care/m/nope"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown meditation: %d", resp.StatusCode)
	}
}

func TestCareOwnIdeas(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	resp, _ := postTaskForm(t, ts, anna, "/care/own", url.Values{"title": {"Hund knuddeln"}, "body": {"Fünf Minuten"}}, "")
	if resp.Header.Get("Location") != "/care#care-own" {
		t.Fatalf("add: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, body := get(t, anna, ts.URL+"/care"); !strings.Contains(body, "<strong>Hund knuddeln</strong>") {
		t.Fatal("own idea not listed")
	}
	if _, body := get(t, ben, ts.URL+"/care"); strings.Contains(body, "<strong>Hund knuddeln</strong>") {
		t.Fatal("ben sees anna's idea")
	}
	if resp, _ := postTaskForm(t, ts, anna, "/care/own", url.Values{"title": {" "}}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Errorf("blank idea: %d", resp.StatusCode)
	}
	id := ts.deps.SelfCare.Own("anna")[0].ID
	postTaskForm(t, ts, anna, "/care/own/"+id+"/delete", url.Values{}, "")
	if len(ts.deps.SelfCare.Own("anna")) != 0 {
		t.Fatal("not deleted")
	}
}

func TestCareEveningReview(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if _, body := get(t, anna, ts.URL+"/care/evening"); !strings.Contains(body, "Wofür bin ich heute dankbar?") || !strings.Contains(body, `form="care-note"`) {
		t.Fatal("evening review missing")
	}
}

func TestCareWaysAndBetween(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/care")
	for _, want := range []string{`class="care-ways"`, `href="/care?level=10#care-help"`, `id="nudge-every"`, `href="/care?level=25#care-suggest"`} {
		if !strings.Contains(body, want) {
			t.Errorf("care: missing %q", want)
		}
	}
	if _, body := get(t, anna, ts.URL+"/care?level=50"); !strings.Contains(body, `href="/care/senses"`) || !strings.Contains(body, "Passt dazu:") {
		t.Error("no practice for half battery")
	}
	if _, body := get(t, anna, ts.URL+"/care?level=75"); !strings.Contains(body, `class="btn btn-secondary btn-sm" href="/care/m/arrive"`) {
		t.Error("no meditation for 75 %")
	}
	_, body = get(t, anna, ts.URL+"/care/between")
	for _, want := range []string{"Was hast du gerade geschafft?", `id="between-tips"`, `href="/focus"`, "js/between.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("between: missing %q", want)
		}
	}
	resp, _ := postTaskForm(t, ts, anna, "/care/between/note", url.Values{"a0": {"Steuer fertig"}, "a1": {"Einkaufen"}}, "")
	loc := resp.Header.Get("Location")
	n, err := ts.deps.Notes.Get("anna", strings.TrimPrefix(loc, "/notes/"))
	if err != nil || !strings.HasPrefix(n.Body, "Zwischendurch – ") || !strings.Contains(n.Body, "Steuer fertig") {
		t.Fatalf("note %q %v", n.Body, err)
	}
}

func TestNudgeBarOnEveryPage(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/notes")
	if !strings.Contains(body, `id="nudge" role="status" hidden`) || !strings.Contains(body, `js/nudge.js?v=`) {
		t.Fatal("nudge bar or script missing")
	}
}

func TestCareSounds(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if _, body := get(t, anna, ts.URL+"/care"); !strings.Contains(body, `href="/care/sounds"`) {
		t.Error("sounds not linked from /care")
	}
	_, body := get(t, anna, ts.URL+"/care/sounds")
	for _, want := range []string{`name="noise" value="waves"`, `id="sleep-timer"`, `id="sounds-meter"`, "js/sounds.js", `data-running="false"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}
