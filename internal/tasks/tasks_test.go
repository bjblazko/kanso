// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateWritesFrontmatterAndCommits(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	b := ownBoard(t, s, a)
	tk, created, err := s.Create(a, New{Board: b.ID, Title: "Lohnsteuer machen", Urgent: true, Due: "2026-11-30"})
	if err != nil || !created {
		t.Fatalf("Create = %v, %v", created, err)
	}
	rel := filepath.Join(Dir, b.ID, tasksDir, tk.Meta.ID+".md")
	raw, err := os.ReadFile(filepath.Join(s.dataDir, rel))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"status: open", "urgent: true", "due: \"2026-11-30\"", "author: anna", "title: Lohnsteuer machen"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("file lacks %q:\n%s", want, raw)
		}
	}
	if log := gitLog(t, s, rel); len(log) != 1 || !strings.HasPrefix(log[0], "Anna|") {
		t.Fatalf("log = %v", log)
	}
	if !idPattern.MatchString(tk.Meta.ID) || tk.Board != b.ID {
		t.Fatalf("task = %+v", tk)
	}
}

func TestCreateSameIDTwice(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	own := ownBoard(t, s, a)
	fb := familyBoard(t, s)
	n := New{ID: "abcdefghijk2", Board: own.ID, Title: "Hof fegen"}
	mustCreate(t, s, a, n)
	if _, err := s.Move(a, own.ID, n.ID, fb.ID); err != nil {
		t.Fatal(err)
	}
	tk, created, err := s.Create(a, n)
	if err != nil || created || tk.Board != fb.ID {
		t.Fatalf("second Create = %+v, %v, %v", tk, created, err)
	}
	all, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*", tasksDir, n.ID+".md"))
	if len(all) != 1 {
		t.Fatalf("files = %v", all)
	}
}

func TestCreateUnknownBoardFallsBackToOwn(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	own := ownBoard(t, s, a)
	foreign := ownBoard(t, s, actor("carla"))
	for _, board := range []string{"", "zzzzzzzzzzzz", foreign.ID, "../x"} {
		tk, _, err := s.Create(a, New{Board: board, Title: "x " + board})
		if err != nil || tk.Board != own.ID {
			t.Fatalf("board %q → %+v, %v", board, tk, err)
		}
	}
}

func TestCreateValidation(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	cases := []New{
		{Title: "  "},
		{Title: strings.Repeat("ä", 201)},
		{Title: "x", Due: "2026-13-01"},
		{Title: "x", Due: "morgen"},
		{Title: "x", Body: strings.Repeat("a", maxBody+1)},
		{ID: "BAD", Title: "x"},
	}
	for _, n := range cases {
		if _, _, err := s.Create(a, n); !errors.Is(err, ErrInvalid) && !errors.Is(err, ErrBadID) {
			t.Errorf("Create(%.30q, due %q, id %q) = %v", n.Title, n.Due, n.ID, err)
		}
	}
	if _, _, err := s.Create(a, New{Title: strings.Repeat("ä", 200)}); err != nil {
		t.Fatalf("200 runes = %v", err)
	}
}

func TestCreateBoardFull(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	b := ownBoard(t, s, a)
	dir := filepath.Join(s.dataDir, Dir, b.ID, tasksDir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for i := range maxTasks {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("%012d.md", i)), nil, 0o640); err != nil {
			t.Fatal(err)
		}
	}
	if _, _, err := s.Create(a, New{Board: b.ID, Title: "one more"}); !errors.Is(err, ErrFull) {
		t.Fatalf("Create = %v", err)
	}
}

func TestUpdateConflict(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Title: "Hof fegen"})
	e := Edit{Title: "Hof fegen", Body: "mit **Besen**", Status: Doing}
	saved, err := s.Update(a, tk.Board, tk.Meta.ID, e, tk.Meta.Updated)
	if err != nil || saved.Body != "mit **Besen**" || saved.Meta.Status != Doing {
		t.Fatalf("Update = %+v, %v", saved, err)
	}
	e.Body = "alt"
	if _, err := s.Update(a, tk.Board, tk.Meta.ID, e, tk.Meta.Updated); !errors.Is(err, ErrConflict) {
		t.Fatalf("stale Update = %v", err)
	}
	if got, _ := s.Get(a, tk.Board, tk.Meta.ID); got.Body != "mit **Besen**" {
		t.Fatalf("body = %q", got.Body)
	}
	e.Title = ""
	if _, err := s.Update(a, tk.Board, tk.Meta.ID, e, saved.Meta.Updated); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank title = %v", err)
	}
}

func TestDoneAt(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Title: "Hof fegen"})
	done, err := s.SetStatus(a, tk.Board, tk.Meta.ID, Done)
	if err != nil || done.Meta.DoneAt == nil {
		t.Fatalf("SetStatus done = %+v, %v", done.Meta, err)
	}
	back, err := s.Update(a, tk.Board, tk.Meta.ID, Edit{Title: "Hof fegen", Status: Open}, done.Meta.Updated)
	if err != nil || back.Meta.DoneAt != nil {
		t.Fatalf("Update open = %+v, %v", back.Meta, err)
	}
	again, err := s.Update(a, tk.Board, tk.Meta.ID, Edit{Title: "Hof fegen", Status: Done}, back.Meta.Updated)
	if err != nil || again.Meta.DoneAt == nil {
		t.Fatalf("Update done = %+v, %v", again.Meta, err)
	}
	if _, err := s.SetStatus(a, tk.Board, tk.Meta.ID, Status("later")); !errors.Is(err, ErrInvalid) {
		t.Fatalf("bad status = %v", err)
	}
}

func TestWaitingForKeptWhenLeavingWaiting(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Title: "Kind anmelden"})
	w, err := s.Update(a, tk.Board, tk.Meta.ID, Edit{Title: "Kind anmelden", Status: Waiting, WaitingFor: "Antwort Hort"}, tk.Meta.Updated)
	if err != nil {
		t.Fatal(err)
	}
	moved, err := s.SetStatus(a, tk.Board, tk.Meta.ID, Doing)
	if err != nil || moved.Meta.WaitingFor != "Antwort Hort" {
		t.Fatalf("SetStatus = %+v, %v (from %v)", moved.Meta, err, w.Meta.Status)
	}
	if _, err := s.Update(a, tk.Board, tk.Meta.ID, Edit{Title: "x", Status: Waiting, WaitingFor: strings.Repeat("x", 121)}, moved.Meta.Updated); !errors.Is(err, ErrInvalid) {
		t.Fatalf("long waiting_for = %v", err)
	}
}

func TestSetQuadrant(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Title: "Lohnsteuer"})
	got, err := s.SetQuadrant(a, tk.Board, tk.Meta.ID, false, true)
	if err != nil || got.Meta.Urgent || !got.Meta.Important || QuadrantOf(got) != Plan {
		t.Fatalf("SetQuadrant = %+v, %v", got.Meta, err)
	}
}

func TestClaimOnlyOnGroupBoard(t *testing.T) {
	s := newStore(t)
	anna, ben := actor("anna"), actor("ben")
	priv := mustCreate(t, s, anna, New{Title: "privat"})
	if _, err := s.Claim(anna, priv.Board, priv.Meta.ID, true); !errors.Is(err, ErrPrivate) {
		t.Fatalf("claim private = %v", err)
	}
	fb := familyBoard(t, s)
	tk := mustCreate(t, s, anna, New{Board: fb.ID, Title: "Hof fegen"})
	if got, err := s.Claim(anna, fb.ID, tk.Meta.ID, true); err != nil || got.Meta.Claimed != "anna" {
		t.Fatalf("claim = %+v, %v", got.Meta, err)
	}
	if got, err := s.Claim(ben, fb.ID, tk.Meta.ID, true); err != nil || got.Meta.Claimed != "ben" {
		t.Fatalf("re-claim = %+v, %v", got.Meta, err)
	}
	if got, err := s.Claim(ben, fb.ID, tk.Meta.ID, false); err != nil || got.Meta.Claimed != "" {
		t.Fatalf("release = %+v, %v", got.Meta, err)
	}
}

func TestDeleteGoneThenSetStatus(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Board: familyBoard(t, s).ID, Title: "Hof fegen"})
	if err := s.Delete(a, tk.Board, tk.Meta.ID); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(s.dataDir, Dir, tk.Board, tasksDir, tk.Meta.ID+".md")
	if _, err := s.SetStatus(a, tk.Board, tk.Meta.ID, Done); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetStatus after delete = %v", err)
	}
	if _, err := s.Claim(a, tk.Board, tk.Meta.ID, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Claim after delete = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file came back: %v", err)
	}
	if log := gitLog(t, s, filepath.Join(Dir, tk.Board, tasksDir, tk.Meta.ID+".md")); len(log) != 2 {
		t.Fatalf("log = %v", log)
	}
}

func TestMoveNeedsBothVisible(t *testing.T) {
	s := newStore(t)
	anna := actor("anna")
	tk := mustCreate(t, s, anna, New{Title: "Hof fegen"})
	carla := ownBoard(t, s, actor("carla"))
	if _, err := s.Move(anna, tk.Board, tk.Meta.ID, carla.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Move to foreign = %v", err)
	}
	fb := familyBoard(t, s)
	moved, err := s.Move(anna, tk.Board, tk.Meta.ID, fb.ID)
	if err != nil || moved.Board != fb.ID {
		t.Fatalf("Move = %+v, %v", moved, err)
	}
	if _, err := s.Get(anna, fb.ID, tk.Meta.ID); err != nil {
		t.Fatal(err)
	}
}

func TestBadIDs(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	tk := mustCreate(t, s, a, New{Title: "x"})
	for _, ids := range [][2]string{{"..", tk.Meta.ID}, {tk.Board, "../" + tk.Meta.ID}, {strings.ToUpper(tk.Board), tk.Meta.ID}, {tk.Board, ""}} {
		if _, err := s.Get(a, ids[0], ids[1]); !errors.Is(err, ErrNotFound) {
			t.Errorf("Get(%q, %q) = %v", ids[0], ids[1], err)
		}
	}
	if _, err := s.Tasks(a, "../.."); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Tasks(../..) = %v", err)
	}
}

func TestTasksListsBoard(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	mustCreate(t, s, a, New{Title: "a"})
	tk := mustCreate(t, s, a, New{Title: "b"})
	ts, err := s.Tasks(a, tk.Board)
	if err != nil || len(ts) != 2 {
		t.Fatalf("Tasks = %v, %v", ts, err)
	}
}
