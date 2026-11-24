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
	for _, want := range []string{"Was brauchst du gerade?", `href="/care/between"`, `href="/care/me"`, `href="/care/sos"`,
		`href="tel:08001110111"`, `href="/care/settings"`, `data-key="care"><a href="/care" aria-current="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, "Akku") {
		t.Error("the battery check is gone")
	}
	_, sos := get(t, anna, ts.URL+"/care/sos")
	help, now := strings.Index(sos, `class="care-help"`), strings.Index(sos, `class="care-now"`)
	if help < 0 || help > now || !strings.Contains(sos, `href="/care/sos?after=`) || !strings.Contains(sos, `href="/care/breathe"`) {
		t.Error("SOS: help first, then one calming step and breathing")
	}
	_, me := get(t, anna, ts.URL+"/care/me")
	for _, want := range []string{`href="/care/m/arrive"`, `href="/care/sounds"`, `href="/care/senses"`, "10 Minuten"} {
		if !strings.Contains(me, want) {
			t.Errorf("me: missing %q", want)
		}
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
	if resp.Header.Get("Location") != "/care/settings#care-own" {
		t.Fatalf("add: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, body := get(t, anna, ts.URL+"/care/settings"); !strings.Contains(body, "<strong>Hund knuddeln</strong>") {
		t.Fatal("own idea not listed")
	}
	if _, body := get(t, ben, ts.URL+"/care/settings"); strings.Contains(body, "<strong>Hund knuddeln</strong>") {
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

func TestCareBetween(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/care/between")
	for _, want := range []string{"Was hast du gerade geschafft?", `id="between-tips"`, `href="/focus"`, "js/between.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("between: missing %q", want)
		}
	}
	if _, body := get(t, anna, ts.URL+"/care/settings"); !strings.Contains(body, `id="nudge-every"`) {
		t.Error("reminder setting missing")
	}
	resp, _ := postTaskForm(t, ts, anna, "/care/between/note", url.Values{"a0": {"Steuer fertig"}, "a1": {"Einkaufen"}}, "")
	loc := resp.Header.Get("Location")
	n, err := ts.deps.Notes.Get("anna", strings.TrimPrefix(loc, "/notes/"))
	if err != nil || !strings.HasPrefix(n.Body, "Kurze Pause – ") || !strings.Contains(n.Body, "Steuer fertig") {
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
	if _, body := get(t, anna, ts.URL+"/care/me"); !strings.Contains(body, `href="/care/sounds"`) {
		t.Error("sounds not linked from /care/me")
	}
	_, body := get(t, anna, ts.URL+"/care/sounds")
	for _, want := range []string{`name="noise" value="waves"`, `id="sleep-timer"`, `id="sounds-meter"`, "js/sounds.js", `data-running="false"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}
