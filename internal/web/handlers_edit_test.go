// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// openEditor loads the edit page and returns its body, CSRF and lock token.
func openEditor(t *testing.T, ts *testServer, c *http.Client, id string) (string, string, string) {
	t.Helper()
	resp, body := get(t, c, ts.URL+"/p/"+id+"/edit")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit: %d", resp.StatusCode)
	}
	return body, between(body, `<meta name="csrf" content="`, `"`), between(body, `name="lock" value="`, `"`)
}

func saveJSON(t *testing.T, ts *testServer, c *http.Client, id, csrf, lock, title, body string) (int, map[string]string) {
	t.Helper()
	form := url.Values{"title": {title}, "body": {body}, "lock": {lock}}
	req, _ := http.NewRequest("POST", ts.URL+"/p/"+id, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	out := map[string]string{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode, out
}

func TestEditSaveWithLock(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	body, csrf, lock := openEditor(t, ts, anna, id)
	if lock == "" || !strings.Contains(body, "<textarea") {
		t.Fatalf("editor: lock %q", lock)
	}
	code, _ := saveJSON(t, ts, anna, id, csrf, lock, "Heizung", "Filter wechseln.\n")
	if code != http.StatusOK {
		t.Fatalf("save: %d", code)
	}
	p, _ := ts.deps.Wiki.ByID(id)
	if p.Body != "Filter wechseln.\n" || p.Meta.Author != "Anna" {
		t.Errorf("saved %+v", p)
	}
}

func TestSecondEditorSeesLockAndOwnerCanBreak(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	_, csrf, lock := openEditor(t, ts, anna, id)

	_, body := get(t, admin, ts.URL+"/p/"+id+"/edit")
	if !strings.Contains(body, "Anna bearbeitet diese Seite gerade.") || strings.Contains(body, "<textarea") {
		t.Fatal("second editor did not see the lock")
	}
	if !strings.Contains(body, `action="/p/`+id+`/lock/break"`) {
		t.Error("owner is not offered to break the lock")
	}
	if r := ts.postForm(t, admin, "/p/"+id, "/p/"+id+"/lock/break", url.Values{}); r.StatusCode != http.StatusSeeOther {
		t.Fatalf("break: %d", r.StatusCode)
	}
	code, out := saveJSON(t, ts, anna, id, csrf, lock, "Heizung", "überschreiben\n")
	if code != http.StatusConflict || out["error"] != "lock_lost" {
		t.Errorf("save after break: %d %v", code, out)
	}
	if p, _ := ts.deps.Wiki.ByID(id); p.Body != "" {
		t.Errorf("text was overwritten: %q", p.Body)
	}
}

func TestEditorCannotBreakLock(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	openEditor(t, ts, admin, id)
	_, body := get(t, anna, ts.URL+"/p/"+id+"/edit")
	if strings.Contains(body, "/lock/break") {
		t.Error("editor offered to break the lock")
	}
	if r := ts.postForm(t, anna, "/p/"+id, "/p/"+id+"/lock/break", url.Values{}); r.StatusCode != http.StatusForbidden {
		t.Errorf("editor break: %d", r.StatusCode)
	}
}

func TestHeartbeat(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	_, csrf, lock := openEditor(t, ts, anna, id)
	req, _ := http.NewRequest("POST", ts.URL+"/p/"+id+"/lock/heartbeat", strings.NewReader(url.Values{"lock": {lock}}.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := anna.Do(req)
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Errorf("heartbeat: %v %v", resp.StatusCode, err)
	}
}

func TestViewerCannotEdit(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	oma := ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if r, _ := get(t, oma, ts.URL+"/p/"+id+"/edit"); r.StatusCode != http.StatusForbidden {
		t.Errorf("viewer edit: %d", r.StatusCode)
	}
}

func TestEditorPageLoadsScriptAndPreview(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	body, _, _ := openEditor(t, ts, admin, id)
	for _, want := range []string{`<script type="module" src="/static/js/editor.js?v=`, `id="preview"`, `role="tablist"`, `type="file"`} {
		if !strings.Contains(body, want) {
			t.Errorf("editor page lacks %s", want)
		}
	}
}
