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
