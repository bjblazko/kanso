// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"
)

func TestRailItems(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/notes")
	last := 0
	for _, key := range []string{"start", "notes", "todos", "tasks", "wiki", "kitchen", "focus", "care"} {
		i := strings.Index(body, `data-key="`+key+`"`)
		if i < last {
			t.Fatalf("rail item %q missing or out of order", key)
		}
		last = i
	}
	for _, want := range []string{`class="rail-more"`, `href="/me/menu"`, `data-user="anna"`, `src="/static/js/menu.js?v=`} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	_, body = get(t, anna, ts.URL+"/focus")
	if !strings.Contains(body, `data-key="focus"><a href="/focus" aria-current="true"`) {
		t.Errorf("Fokus not current on /focus")
	}
}

func TestMenuPage(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, anna, ts.URL+"/me/menu")
	if resp.StatusCode != 200 || strings.Count(body, `<label class="choice"><input type="checkbox" checked`) != 8 ||
		!strings.Contains(body, `js/menu-settings.js`) || !strings.Contains(body, `aria-label="Wiki nach oben"`) {
		t.Fatalf("menu page %d:\n%s", resp.StatusCode, body)
	}
}
