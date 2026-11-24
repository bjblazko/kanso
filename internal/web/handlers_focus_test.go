// SPDX-License-Identifier: Apache-2.0

package web

import (
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
