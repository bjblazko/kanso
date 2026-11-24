// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/url"
	"strings"
	"testing"
)

func TestFocusPage(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, anna, ts.URL+"/focus")
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	for _, want := range []string{`role="timer" aria-live="off">25:00<`, `id="focus-toggle"`, ">Start<", ">Überspringen<", ">Zurücksetzen<",
		`id="focus-minutes-focus" type="number" min="1" max="120" value="25"`, `js/focus.js`, `data-key="focus"><a href="/focus" aria-current="true"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestFocusBreakTipsAndParking(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/focus")
	for _, want := range []string{`<template id="focus-tips"><li data-title="Ans Fenster">`, `id="focus-break" hidden`, `action="/focus/note"`, `form="focus-park"`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	resp, _ := postTaskForm(t, ts, anna, "/focus/note", url.Values{"text": {"Zahnarzt anrufen"}}, "")
	loc := resp.Header.Get("Location")
	if !strings.HasPrefix(loc, "/focus?saved=") {
		t.Fatalf("redirect %q", loc)
	}
	n, err := ts.deps.Notes.Get("anna", strings.TrimPrefix(loc, "/focus?saved="))
	if err != nil || !strings.HasPrefix(n.Body, "Gedanken-Parkplatz – ") || !strings.Contains(n.Body, "Zahnarzt anrufen") {
		t.Fatalf("note %q %v", n.Body, err)
	}
	if _, body := get(t, anna, ts.URL+loc); !strings.Contains(body, `href="/notes/`+n.Meta.ID+`"`) {
		t.Error("no link to the saved note")
	}
	if resp, _ := postTaskForm(t, ts, anna, "/focus/note", url.Values{"text": {"  "}}, ""); resp.Header.Get("Location") != "/focus" {
		t.Error("empty parking saved")
	}
}
