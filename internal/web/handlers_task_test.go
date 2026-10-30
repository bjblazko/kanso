// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/tasks"
)

// postTaskForm sends a form with the CSRF token of the tasks page.
func postTaskForm(t *testing.T, ts *testServer, c *http.Client, path string, form url.Values, accept string) (*http.Response, string) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/tasks")
	form.Set("csrf", between(page, `<meta name="csrf" content="`, `"`))
	req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	buf := make([]byte, 1<<16)
	for {
		n, err := resp.Body.Read(buf)
		b.Write(buf[:n])
		if err != nil {
			break
		}
	}
	return resp, b.String()
}

func taskURL(tk tasks.Task) string { return "/tasks/b/" + tk.Board + "/" + tk.Meta.ID }

func editForm(tk tasks.Task, f url.Values) url.Values {
	base := url.Values{"title": {tk.Meta.Title}, "body": {tk.Body}, "status": {string(tk.Meta.Status)},
		"board": {tk.Board}, "version": {tk.Meta.Updated.Format(time.RFC3339Nano)}}
	for k, v := range f {
		base[k] = v
	}
	return base
}

func TestTaskViewRendersMarkdownSafely(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen", Body: "**mit Besen**\n\n<script>alert(1)</script>\n"})
	resp, body := get(t, anna, ts.URL+taskURL(tk))
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "<strong>mit Besen</strong>") || strings.Contains(body, "<script>alert") {
		t.Fatalf("%d\n%s", resp.StatusCode, body)
	}
}

func TestTaskViewNoteLink(t *testing.T) {
	ts := newTestServer(t)
	anna, ben := ts.client(t, "anna", "Anna", false), ts.client(t, "ben", "Ben", false)
	fam := familyBoardID(t, ts)
	n, _, err := ts.deps.Notes.Create("anna", "", "Kind anmelden\n", nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	tk := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Kind anmelden", Note: n.Meta.ID})
	if _, body := get(t, anna, ts.URL+taskURL(tk)); !strings.Contains(body, `href="/notes/`+n.Meta.ID+`"`) {
		t.Fatalf("author misses the note link:\n%s", body)
	}
	if _, body := get(t, ben, ts.URL+taskURL(tk)); strings.Contains(body, "/notes/"+n.Meta.ID) || strings.Contains(body, "Notiz gelöscht") {
		t.Fatal("ben sees anna's note")
	}
	if err := ts.deps.Notes.Delete("anna", n.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, anna, ts.URL+taskURL(tk)); !strings.Contains(body, "Notiz gelöscht") {
		t.Fatalf("deleted note not said:\n%s", body)
	}
}

func TestSaveTask(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Kind anmelden"})
	if _, body := get(t, anna, ts.URL+taskURL(tk)+"/edit"); !strings.Contains(body, `name="version"`) || !strings.Contains(body, "Kind anmelden") {
		t.Fatalf("edit page:\n%s", body)
	}
	f := editForm(tk, url.Values{"title": {"Kind zur Ferienbetreuung anmelden"}, "body": {"per Mail"}, "status": {"waiting"},
		"urgent": {"1"}, "important": {"1"}, "due": {"2026-11-06"}, "waiting_for": {"Antwort Hort"}})
	resp, _ := postTaskForm(t, ts, anna, taskURL(tk), f, "")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != taskURL(tk) {
		t.Fatalf("save: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	got, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID)
	m := got.Meta
	if m.Title != "Kind zur Ferienbetreuung anmelden" || got.Body != "per Mail" || m.Status != tasks.Waiting || !m.Urgent || !m.Important ||
		m.Due != "2026-11-06" || m.WaitingFor != "Antwort Hort" {
		t.Fatalf("saved %+v %q", m, got.Body)
	}
}

func TestSaveTaskConflict(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	if _, err := ts.deps.Tasks.SetStatus(taskActor(ts, "anna"), tk.Board, tk.Meta.ID, tasks.Doing); err != nil {
		t.Fatal(err)
	}
	resp, body := postTaskForm(t, ts, anna, taskURL(tk), editForm(tk, url.Values{"body": {"neu"}}), "application/json")
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, `"conflict"`) {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	resp, body = postTaskForm(t, ts, anna, taskURL(tk), editForm(tk, url.Values{"body": {"mein Text"}}), "")
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, "mein Text") {
		t.Fatalf("form conflict: %d", resp.StatusCode)
	}
	if got, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID); got.Body != "" {
		t.Fatalf("body overwritten: %q", got.Body)
	}
}

func TestSaveTaskInvalid(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	resp, body := postTaskForm(t, ts, anna, taskURL(tk), editForm(tk, url.Values{"title": {" "}, "body": {"mein Text"}}), "")
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "mein Text") {
		t.Fatalf("%d", resp.StatusCode)
	}
}

func TestSaveMovesToOtherBoard(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	resp, body := postTaskForm(t, ts, anna, taskURL(tk), editForm(tk, url.Values{"board": {fam}}), "application/json")
	want := "/tasks/b/" + fam + "/" + tk.Meta.ID
	var answer struct{ URL string }
	_ = json.Unmarshal([]byte(body), &answer)
	if resp.StatusCode != http.StatusOK || answer.URL != want {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if _, err := ts.deps.Tasks.Get(taskActor(ts, "anna"), fam, tk.Meta.ID); err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("git", "-C", ts.deps.Cfg.DataDir, "log", "-1", "--format=%s").Output()
	if !strings.Contains(string(out), "verschoben") {
		t.Fatalf("last commit %q", out)
	}
	carla := ownBoardID(t, ts, "carla")
	moved, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), fam, tk.Meta.ID)
	resp, _ = postTaskForm(t, ts, anna, want, editForm(moved, url.Values{"board": {carla}}), "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("move to foreign board: %d", resp.StatusCode)
	}
}

func TestClaimAndRelease(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	fam := familyBoardID(t, ts)
	tk := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Hof fegen"})
	if _, body := get(t, ben, ts.URL+taskURL(tk)); !strings.Contains(body, "Übernehmen") {
		t.Fatalf("no claim button:\n%s", body)
	}
	resp, _ := postTaskForm(t, ts, ben, taskURL(tk)+"/claim", url.Values{"claim": {"1"}}, "")
	if got, _ := ts.deps.Tasks.Get(taskActor(ts, "ben"), fam, tk.Meta.ID); resp.StatusCode != http.StatusSeeOther || got.Meta.Claimed != "ben" {
		t.Fatalf("claim: %d %q", resp.StatusCode, got.Meta.Claimed)
	}
	if _, body := get(t, anna, ts.URL+taskURL(tk)); !strings.Contains(body, "übernommen von Ben") {
		t.Fatal("anna does not see who took it")
	}
	postTaskForm(t, ts, ben, taskURL(tk)+"/claim", url.Values{"claim": {"0"}}, "")
	if got, _ := ts.deps.Tasks.Get(taskActor(ts, "ben"), fam, tk.Meta.ID); got.Meta.Claimed != "" {
		t.Fatal("not released")
	}
	priv := addTask(t, ts, "anna", tasks.New{Title: "privat"})
	if resp, _ := postTaskForm(t, ts, anna, taskURL(priv)+"/claim", url.Values{"claim": {"1"}}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("claim private: %d", resp.StatusCode)
	}
}

func TestDeleteTask(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	if _, body := get(t, anna, ts.URL+taskURL(tk)+"/delete"); !strings.Contains(body, "„Hof fegen“ löschen?") {
		t.Fatalf("confirm page:\n%s", body)
	}
	resp, _ := postTaskForm(t, ts, anna, taskURL(tk)+"/delete", url.Values{}, "")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/tasks/b/"+tk.Board {
		t.Fatalf("delete: %d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, err := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID); err == nil {
		t.Fatal("still there")
	}
}

func TestTaskOfStrangerIs404(t *testing.T) {
	ts := newTestServer(t)
	ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	for _, p := range []string{"", "/edit", "/delete"} {
		if resp, _ := get(t, ben, ts.URL+taskURL(tk)+p); resp.StatusCode != http.StatusNotFound {
			t.Errorf("GET %s: %d", p, resp.StatusCode)
		}
	}
	for _, p := range []string{"", "/claim", "/delete"} {
		if resp, _ := postTaskForm(t, ts, ben, taskURL(tk)+p, editForm(tk, url.Values{"claim": {"1"}}), ""); resp.StatusCode != http.StatusNotFound {
			t.Errorf("POST %s: %d", p, resp.StatusCode)
		}
	}
	if got, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID); got.Meta.Title != "Hof fegen" {
		t.Fatal("changed by a stranger")
	}
}

func TestTaskPreview(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	ref := tk.Board + "/" + tk.Meta.ID
	resp, body := postTaskForm(t, ts, anna, "/api/preview", url.Values{"task": {ref}, "body": {"**x**"}}, "")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "<strong>x</strong>") {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	if resp, _ := postTaskForm(t, ts, ben, "/api/preview", url.Values{"task": {ref}, "body": {"x"}}, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("foreign preview: %d", resp.StatusCode)
	}
}

func moveTask(t *testing.T, ts *testServer, c *http.Client, tk tasks.Task, f url.Values) (*http.Response, string) {
	t.Helper()
	return postTaskForm(t, ts, c, taskURL(tk)+"/move", f, "application/json")
}

func TestMoveStatus(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	resp, body := moveTask(t, ts, anna, tk, url.Values{"status": {"done"}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"status":"done"`) {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	got, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID)
	if got.Meta.Status != tasks.Done || got.Meta.DoneAt == nil {
		t.Fatalf("%+v", got.Meta)
	}
	out, _ := exec.Command("git", "-C", ts.deps.Cfg.DataDir, "log", "-1", "--format=%s").Output()
	if !strings.Contains(string(out), "→ done") {
		t.Fatalf("commit %q", out)
	}
}

func TestMoveQuadrant(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Lohnsteuer"})
	resp, body := moveTask(t, ts, anna, tk, url.Values{"urgent": {"1"}, "important": {"1"}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"urgent":true`) {
		t.Fatalf("%d %s", resp.StatusCode, body)
	}
	got, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID)
	if !got.Meta.Urgent || !got.Meta.Important {
		t.Fatalf("%+v", got.Meta)
	}
}

func TestMoveGoneTask(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	if err := ts.deps.Tasks.Delete(taskActor(ts, "anna"), tk.Board, tk.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if resp, _ := moveTask(t, ts, anna, tk, url.Values{"status": {"done"}}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if _, err := ts.deps.Tasks.Get(taskActor(ts, "anna"), tk.Board, tk.Meta.ID); err == nil {
		t.Fatal("task came back")
	}
}

func TestMoveBadValue(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	for _, f := range []url.Values{{"status": {"later"}}, {}} {
		if resp, _ := moveTask(t, ts, anna, tk, f); resp.StatusCode != http.StatusBadRequest {
			t.Errorf("%v: %d", f, resp.StatusCode)
		}
	}
}

func TestMoveNeedsCSRF(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	req, _ := http.NewRequest("POST", ts.URL+taskURL(tk)+"/move", strings.NewReader("status=done"))
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

func TestBoardCardsAreDraggable(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	tk := addTask(t, ts, "anna", tasks.New{Title: "Hof fegen"})
	_, body := get(t, anna, ts.URL+"/tasks/b/"+tk.Board)
	if !strings.Contains(body, `draggable="true"`) || !strings.Contains(body, "js/board.js") {
		t.Fatalf("board not draggable:\n%s", body)
	}
}
