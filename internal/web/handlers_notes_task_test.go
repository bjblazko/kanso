// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/notes"
)

func TestNoteToTask(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, err := ts.deps.Notes.Create("anna", "", "# Kind anmelden\nMail an den Hort\n", nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	own := ownBoardID(t, ts, "anna")
	if _, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID+"/task"); !strings.Contains(body, `value="`+own+`"`) {
		t.Fatalf("no board choice:\n%s", body)
	}
	resp, _ := postTaskForm(t, ts, anna, "/notes/"+n.Meta.ID+"/task", url.Values{"board": {own}}, "")
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/tasks/b/"+own+"/") || !strings.HasSuffix(loc, "/edit") {
		t.Fatalf("%d %s", resp.StatusCode, loc)
	}
	taskID := strings.TrimSuffix(strings.TrimPrefix(loc, "/tasks/b/"+own+"/"), "/edit")
	tk, err := ts.deps.Tasks.Get(taskActor(ts, "anna"), own, taskID)
	if err != nil || tk.Meta.Title != "Kind anmelden" || tk.Body != "Mail an den Hort\n" || tk.Meta.Note != n.Meta.ID {
		t.Fatalf("task %+v %q %v", tk.Meta, tk.Body, err)
	}
	got, _ := ts.deps.Notes.Get("anna", n.Meta.ID)
	if got.Meta.Task != own+"/"+taskID || got.Body != n.Body {
		t.Fatalf("note %+v", got.Meta)
	}
}

func TestNoteToTaskForeignNote(t *testing.T) {
	ts := newTestServer(t)
	ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Geheim\n", nil, 1<<20)
	if resp, _ := get(t, ben, ts.URL+"/notes/"+n.Meta.ID+"/task"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET %d", resp.StatusCode)
	}
	if resp, _ := postTaskForm(t, ts, ben, "/notes/"+n.Meta.ID+"/task", url.Values{}, ""); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("POST %d", resp.StatusCode)
	}
}

func TestNoteToTaskPhotoOnly(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, err := ts.deps.Notes.Create("anna", "", "", []notes.Upload{{Name: "zettel.png", R: bytes.NewReader([]byte("png"))}}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	postTaskForm(t, ts, anna, "/notes/"+n.Meta.ID+"/task", url.Values{}, "")
	got, _ := ts.deps.Notes.Get("anna", n.Meta.ID)
	board, id, _ := strings.Cut(got.Meta.Task, "/")
	tk, err := ts.deps.Tasks.Get(taskActor(ts, "anna"), board, id)
	if err != nil || !strings.HasPrefix(tk.Meta.Title, "Foto vom") {
		t.Fatalf("task %+v %v", tk.Meta, err)
	}
}

func TestNoteViewShowsTaskLink(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Hof fegen\n", nil, 1<<20)
	if _, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID); !strings.Contains(body, `href="/notes/`+n.Meta.ID+`/task"`) {
		t.Fatalf("no 'Als Aufgabe' link:\n%s", body)
	}
	postTaskForm(t, ts, anna, "/notes/"+n.Meta.ID+"/task", url.Values{}, "")
	got, _ := ts.deps.Notes.Get("anna", n.Meta.ID)
	_, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID)
	_, taskID, _ := strings.Cut(got.Meta.Task, "/")
	if !strings.Contains(body, `href="/tasks/t/`+taskID+`"`) || strings.Contains(body, `/notes/`+n.Meta.ID+`/task"`) {
		t.Fatalf("task link missing:\n%s", body)
	}
}

// turnIntoTask turns anna's note into a task on her own board and returns the task.
func turnIntoTask(t *testing.T, ts *testServer, anna *http.Client, n notes.Note) (string, string) {
	t.Helper()
	postTaskForm(t, ts, anna, "/notes/"+n.Meta.ID+"/task", url.Values{}, "")
	got, _ := ts.deps.Notes.Get("anna", n.Meta.ID)
	board, id, _ := strings.Cut(got.Meta.Task, "/")
	return board, id
}

func TestNoteTaskLinkFollowsMove(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Hof fegen\n", nil, 1<<20)
	board, id := turnIntoTask(t, ts, anna, n)
	if _, err := ts.deps.Tasks.Move(taskActor(ts, "anna"), board, id, fam); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID)
	if !strings.Contains(body, `href="/tasks/t/`+id+`"`) {
		t.Fatalf("note link:\n%s", body)
	}
	resp, _ := get(t, anna, ts.URL+"/tasks/t/"+id)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/tasks/b/"+fam+"/"+id {
		t.Fatalf("%d %s", resp.StatusCode, resp.Header.Get("Location"))
	}
	ben := ts.client(t, "carla", "Carla", false)
	if resp, _ := get(t, ben, ts.URL+"/tasks/t/"+id); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("stranger: %d", resp.StatusCode)
	}
}

func TestNoteOffersTaskAgainWhenGone(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Hof fegen\n", nil, 1<<20)
	board, id := turnIntoTask(t, ts, anna, n)
	if err := ts.deps.Tasks.Delete(taskActor(ts, "anna"), board, id); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID)
	if !strings.Contains(body, `href="/notes/`+n.Meta.ID+`/task"`) || strings.Contains(body, "/tasks/t/"+id) {
		t.Fatalf("note view:\n%s", body)
	}
}

func TestNoteToTaskDropsPhotoLinks(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, err := ts.deps.Notes.Create("anna", "", "Zettel vom Hort\nbitte ausfüllen\n", []notes.Upload{{Name: "zettel.png", R: bytes.NewReader([]byte("png"))}}, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	board, id := turnIntoTask(t, ts, anna, n)
	tk, _ := ts.deps.Tasks.Get(taskActor(ts, "anna"), board, id)
	if tk.Meta.Title != "Zettel vom Hort" || strings.Contains(tk.Body, "attachments/") || !strings.Contains(tk.Body, "bitte ausfüllen") {
		t.Fatalf("task %q %q", tk.Meta.Title, tk.Body)
	}
}

func TestNoteToTaskTooLong(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Langer Text\n"+strings.Repeat("x", 101<<10), nil, 1<<20)
	if resp, _ := postTaskForm(t, ts, anna, "/notes/"+n.Meta.ID+"/task", url.Values{}, ""); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
