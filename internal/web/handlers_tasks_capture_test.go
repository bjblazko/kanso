// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/tasks"
)

type createAnswer struct {
	ID, Board, Status, Error string
}

func createTask(t *testing.T, ts *testServer, c *http.Client, f url.Values) (*http.Response, createAnswer) {
	t.Helper()
	resp, body := postTaskForm(t, ts, c, "/tasks", f, "application/json")
	var a createAnswer
	_ = json.Unmarshal([]byte(body), &a)
	return resp, a
}

func TestCaptureFormDefaults(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	resp, body := get(t, anna, ts.URL+"/tasks/new?board="+fam)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	for _, want := range []string{`name="title" required maxlength="200" autofocus`, `value="` + fam + `" checked`, `name="due" value="" checked`,
		"Mein Board", "Familie", `type="checkbox" name="urgent" value="1">`, `type="checkbox" name="important" value="1">`, "js/task-capture.js"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
	if strings.Contains(body, `name="urgent" value="1" checked`) {
		t.Error("urgent is preset")
	}
}

func TestCreateTaskForm(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	resp, _ := postTaskForm(t, ts, anna, "/tasks", url.Values{"title": {"Hof fegen"}, "board": {fam}, "urgent": {"1"}, "due": {day(0)}}, "")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/tasks/new?board="+fam+"&saved="+fam {
		t.Fatalf("%d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	ts2, _ := ts.deps.Tasks.Tasks(taskActor(ts, "anna"), fam)
	if len(ts2) != 1 || !ts2[0].Meta.Urgent || ts2[0].Meta.Due != day(0) || ts2[0].Meta.Status != tasks.Open {
		t.Fatalf("tasks %+v", ts2)
	}
	if _, body := get(t, anna, ts.URL+"/tasks/new?board="+fam+"&saved="+fam); !strings.Contains(body, "Gespeichert in Familie") {
		t.Fatal("no saved message")
	}
}

func TestCreateTaskJSONTwice(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	f := url.Values{"id": {"abcdefghijk2"}, "title": {"Lohnsteuer"}}
	resp, a := createTask(t, ts, anna, f)
	if resp.StatusCode != http.StatusCreated || a.ID != "abcdefghijk2" || a.Status != "created" {
		t.Fatalf("first: %d %+v", resp.StatusCode, a)
	}
	resp, b := createTask(t, ts, anna, f)
	if resp.StatusCode != http.StatusOK || b.Status != "exists" || b.Board != a.Board {
		t.Fatalf("second: %d %+v", resp.StatusCode, b)
	}
	files, _ := filepath.Glob(filepath.Join(ts.deps.Cfg.DataDir, tasks.Dir, "*", "tasks", "*.md"))
	if len(files) != 1 {
		t.Fatalf("files %v", files)
	}
}

func TestCreateTaskUnknownBoard(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, a := createTask(t, ts, anna, url.Values{"title": {"Hof fegen"}, "board": {"zzzzzzzzzzzz"}})
	if resp.StatusCode != http.StatusCreated || a.Board != ownBoardID(t, ts, "anna") {
		t.Fatalf("%d %+v", resp.StatusCode, a)
	}
}

func TestCreateTaskInvalid(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	for _, f := range []url.Values{{"title": {"  "}}, {"title": {"x"}, "due": {"morgen"}}, {"title": {strings.Repeat("x", 201)}}, {"title": {"x"}, "id": {"../.."}}} {
		if resp, a := createTask(t, ts, anna, f); resp.StatusCode != http.StatusBadRequest || a.Error != "invalid" {
			t.Errorf("%v: %d %+v", f, resp.StatusCode, a)
		}
	}
	resp, body := postTaskForm(t, ts, anna, "/tasks", url.Values{"title": {" "}}, "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "Betreff") {
		t.Fatalf("form: %d", resp.StatusCode)
	}
}

func TestCreateTaskNeedsCSRF(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	req, _ := http.NewRequest("POST", ts.URL+"/tasks", strings.NewReader("title=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := anna.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestServiceWorkerPrecachesCapture(t *testing.T) {
	ts := newTestServer(t)
	_, body := get(t, http.DefaultClient, ts.URL+"/sw.js")
	if !strings.Contains(body, "js/task-capture.js") || !strings.Contains(body, "'/tasks/new'") {
		t.Fatalf("sw.js:\n%s", body)
	}
}
