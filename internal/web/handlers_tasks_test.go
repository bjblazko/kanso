// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/tasks"
)

// taskActor is what the store sees for sub in the test server.
func taskActor(ts *testServer, sub string) tasks.Actor {
	g, _ := access.LoadGroups(ts.deps.Cfg.DataDir)
	return tasks.Actor{Sub: sub, Author: store.Author{Name: sub, Email: sub + "@example.org"}, Groups: g}
}

func ownBoardID(t *testing.T, ts *testServer, sub string) string {
	t.Helper()
	bs, err := ts.deps.Tasks.Boards(taskActor(ts, sub))
	if err != nil {
		t.Fatal(err)
	}
	return bs[0].ID
}

// familyBoardID makes group "familie" (anna, ben) and its board.
func familyBoardID(t *testing.T, ts *testServer) string {
	t.Helper()
	if err := access.SaveGroups(ts.deps.Cfg.DataDir, access.Groups{"familie": {"anna", "ben"}}); err != nil {
		t.Fatal(err)
	}
	b, err := ts.deps.Tasks.CreateGroupBoard(store.Author{Name: "Admin", Email: "a@example.org"}, "Familie", "familie")
	if err != nil {
		t.Fatal(err)
	}
	return b.ID
}

func addTask(t *testing.T, ts *testServer, sub string, n tasks.New) tasks.Task {
	t.Helper()
	tk, _, err := ts.deps.Tasks.Create(taskActor(ts, sub), n)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

func inOrder(t *testing.T, body string, parts ...string) {
	t.Helper()
	at := 0
	for _, p := range parts {
		i := strings.Index(body[at:], p)
		if i < 0 {
			t.Fatalf("%q missing or out of order in:\n%s", p, body)
		}
		at += i + len(p)
	}
}

func day(offset int) string { return time.Now().AddDate(0, 0, offset).Format("2006-01-02") }

func TestTasksShowsOwnBoard(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, body := get(t, anna, ts.URL+"/tasks")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	for _, want := range []string{"<h1>Mein Board</h1>", "Noch keine Aufgaben.", `href="/tasks" aria-current="true"`, "Nur für dich sichtbar"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestBoardOfStrangerIs404(t *testing.T) {
	ts := newTestServer(t)
	ts.client(t, "anna", "Anna", false)
	ben, boss := ts.client(t, "ben", "Ben", false), ts.client(t, "boss", "Boss", true)
	id := ownBoardID(t, ts, "anna")
	for _, c := range []*http.Client{ben, boss} {
		if resp, _ := get(t, c, ts.URL+"/tasks/b/"+id); resp.StatusCode != http.StatusNotFound {
			t.Errorf("status %d", resp.StatusCode)
		}
	}
	if resp, _ := get(t, ben, ts.URL+"/tasks/b/nothing"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown board: %d", resp.StatusCode)
	}
}

func TestBoardTabs(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	id := ownBoardID(t, ts, "anna")
	_, body := get(t, anna, ts.URL+"/tasks/b/"+id+"?view=matrix")
	if !strings.Contains(body, `href="/tasks/b/`+id+`?view=matrix" aria-current="page"`) ||
		!strings.Contains(body, `href="/tasks/b/`+id+`?view=kanban">Kanban`) {
		t.Fatalf("tabs wrong:\n%s", body)
	}
	_, body = get(t, anna, ts.URL+"/tasks/b/"+id)
	if !strings.Contains(body, `href="/tasks/b/`+id+`?view=kanban" aria-current="page"`) {
		t.Fatalf("kanban is not the default:\n%s", body)
	}
}

func TestKanbanColumnsAndOrder(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	id := ownBoardID(t, ts, "anna")
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Ohne Datum"})
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Später fällig", Due: day(5)})
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Bald fällig", Due: day(2)})
	w := addTask(t, ts, "anna", tasks.New{Board: id, Title: "Wartendes"})
	if _, err := ts.deps.Tasks.SetStatus(taskActor(ts, "anna"), id, w.Meta.ID, tasks.Waiting); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/tasks/b/"+id)
	inOrder(t, body, "Offen", "Bald fällig", "Später fällig", "Ohne Datum", "In Arbeit", "Wartet", "Wartendes", "Erledigt")
}

func TestKanbanDoneFilter(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	id := ownBoardID(t, ts, "anna")
	old := addTask(t, ts, "anna", tasks.New{Board: id, Title: "Altes Erledigtes"})
	if _, err := ts.deps.Tasks.SetStatus(taskActor(ts, "anna"), id, old.Meta.ID, tasks.Done); err != nil {
		t.Fatal(err)
	}
	backdate(t, ts, id, old.Meta.ID, -15)
	_, body := get(t, anna, ts.URL+"/tasks/b/"+id)
	if strings.Contains(body, "Altes Erledigtes") || !strings.Contains(body, "Ältere erledigte anzeigen") {
		t.Fatalf("old done shown or link missing:\n%s", body)
	}
	_, body = get(t, anna, ts.URL+"/tasks/b/"+id+"?view=kanban&done=all")
	if !strings.Contains(body, "Altes Erledigtes") {
		t.Fatalf("done=all hides it:\n%s", body)
	}
}

// backdate moves a task's done_at by days.
func backdate(t *testing.T, ts *testServer, board, id string, days int) {
	t.Helper()
	path := filepath.Join(ts.deps.Cfg.DataDir, tasks.Dir, board, "tasks", id+".md")
	d, err := store.ReadDoc[tasks.Meta](path)
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().AddDate(0, 0, days)
	d.Meta.DoneAt = &when
	if err := store.WriteDoc(path, d); err != nil {
		t.Fatal(err)
	}
}

func TestMatrixQuadrants(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	id := ownBoardID(t, ts, "anna")
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Ferienbetreuung", Urgent: true, Important: true})
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Lohnsteuer", Important: true})
	addTask(t, ts, "anna", tasks.New{Board: id, Title: "Hof fegen", Urgent: true})
	done := addTask(t, ts, "anna", tasks.New{Board: id, Title: "Erledigtes", Urgent: true})
	if _, err := ts.deps.Tasks.SetStatus(taskActor(ts, "anna"), id, done.Meta.ID, tasks.Done); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/tasks/b/"+id+"?view=matrix")
	inOrder(t, body, "Sofort", "Ferienbetreuung", "Planen", "Lohnsteuer", "Kurz erledigen", "Hof fegen", "Später")
	if strings.Contains(body, "Erledigtes") {
		t.Fatal("matrix shows a done task")
	}
}

func TestBoardSwitcher(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	own := ownBoardID(t, ts, "anna")
	_, body := get(t, anna, ts.URL+"/tasks")
	if !strings.Contains(body, `href="/tasks/b/`+own+`"`) || !strings.Contains(body, `href="/tasks/b/`+fam+`">Familie`) {
		t.Fatalf("switcher missing:\n%s", body)
	}
}

func TestCardShowsDueAndClaim(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ts.client(t, "ben", "Ben", false)
	fam := familyBoardID(t, ts)
	addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Zu spät", Due: day(-1)})
	c := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Hof fegen"})
	if _, err := ts.deps.Tasks.Claim(taskActor(ts, "ben"), fam, c.Meta.ID, true); err != nil {
		t.Fatal(err)
	}
	mine := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Meins"})
	if _, err := ts.deps.Tasks.Claim(taskActor(ts, "anna"), fam, mine.Meta.ID, true); err != nil {
		t.Fatal(err)
	}
	w := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Anmelden"})
	if _, err := ts.deps.Tasks.Update(taskActor(ts, "anna"), fam, w.Meta.ID,
		tasks.Edit{Title: "Anmelden", Status: tasks.Waiting, WaitingFor: "Antwort Hort"}, w.Meta.Updated); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/tasks/b/"+fam)
	for _, want := range []string{"überfällig", "übernommen von Ben", "übernommen von dir", "Antwort Hort"} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q", want)
		}
	}
}

func TestBoardAudience(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ts.client(t, "ben", "Ben", false)
	fam := familyBoardID(t, ts)
	_, body := get(t, anna, ts.URL+"/tasks/b/"+fam)
	if !strings.Contains(body, "Sichtbar für dich, Ben") {
		t.Fatalf("audience missing:\n%s", body)
	}
}
