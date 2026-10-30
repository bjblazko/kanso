// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func countOwnBoards(t *testing.T, s *Store, sub string) int {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*", boardFile))
	n := 0
	for _, f := range files {
		var b Board
		if err := readBoardFile(f, &b); err == nil && b.Owner.User == sub {
			n++
		}
	}
	return n
}

func TestOwnBoardCreatedOnce(t *testing.T) {
	s := newStore(t)
	a := actor("anna")
	first := ownBoard(t, s, a)
	if again := ownBoard(t, s, a); again.ID != first.ID {
		t.Fatalf("second call made another board: %s vs %s", again.ID, first.ID)
	}
	var wg sync.WaitGroup
	for range 10 {
		wg.Go(func() {
			if _, _, err := s.Create(actor("ben"), New{Title: "Hof fegen"}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if n := countOwnBoards(t, s, "ben"); n != 1 {
		t.Fatalf("ben has %d boards, want 1", n)
	}
	if n := countOwnBoards(t, s, "anna"); n != 1 {
		t.Fatalf("anna has %d boards, want 1", n)
	}
}

func TestStrangerCannotSeePrivateBoard(t *testing.T) {
	s := newStore(t)
	b := ownBoard(t, s, actor("anna"))
	tk := mustCreate(t, s, actor("anna"), New{Board: b.ID, Title: "Lohnsteuer"})
	ben := actor("ben")
	if _, err := s.Board(ben, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Board = %v", err)
	}
	if _, err := s.Tasks(ben, b.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Tasks = %v", err)
	}
	if _, err := s.Get(ben, b.ID, tk.Meta.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get = %v", err)
	}
}

func TestAdminHasNoAccess(t *testing.T) {
	s := newStore(t)
	fb := familyBoard(t, s)
	boss := actor("boss") // an admin is just a person to the store
	if _, err := s.Board(boss, fb.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Board = %v", err)
	}
	bs, err := s.Boards(boss)
	if err != nil || len(bs) != 1 {
		t.Fatalf("Boards = %v, %v; want only the own board", bs, err)
	}
}

func TestGroupMemberSeesGroupBoard(t *testing.T) {
	s := newStore(t)
	fb := familyBoard(t, s)
	got, err := s.Board(actor("ben"), fb.ID)
	if err != nil || got.Name != "Familie" || got.Private() {
		t.Fatalf("Board = %+v, %v", got, err)
	}
}

func TestBoardsOrder(t *testing.T) {
	s := newStore(t)
	if _, err := s.CreateGroupBoard(admin(), "Zuhause", "familie"); err != nil {
		t.Fatal(err)
	}
	familyBoard(t, s)
	bs, err := s.Boards(actor("anna"))
	if err != nil || len(bs) != 3 {
		t.Fatalf("Boards = %v, %v", bs, err)
	}
	if !bs[0].Private() || bs[1].Name != "Familie" || bs[2].Name != "Zuhause" {
		t.Fatalf("order = %+v", bs)
	}
}

func TestGroupBoardAdminFunctions(t *testing.T) {
	s := newStore(t)
	fb := familyBoard(t, s)
	if err := s.RenameBoard(admin(), fb.ID, "Haushalt"); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Board(actor("anna"), fb.ID); got.Name != "Haushalt" {
		t.Fatalf("name = %q", got.Name)
	}
	own := ownBoard(t, s, actor("anna"))
	if err := s.RenameBoard(admin(), own.ID, "X"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("rename private = %v", err)
	}
	if err := s.DeleteBoard(admin(), own.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("delete private = %v", err)
	}
	if gb := s.GroupBoards(); len(gb) != 1 || gb[0].ID != fb.ID {
		t.Fatalf("GroupBoards = %+v", gb)
	}
	tk := mustCreate(t, s, actor("anna"), New{Board: fb.ID, Title: "Hof fegen"})
	if err := s.DeleteBoard(admin(), fb.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, Dir, fb.ID)); !os.IsNotExist(err) {
		t.Fatalf("board folder still there: %v", err)
	}
	if log := gitLog(t, s, filepath.Join(Dir, fb.ID, tasksDir, tk.Meta.ID+".md")); len(log) < 2 {
		t.Fatalf("history lost: %v", log)
	}
	if _, err := s.CreateGroupBoard(admin(), " ", "familie"); !errors.Is(err, ErrInvalid) {
		t.Fatalf("blank name = %v", err)
	}
}
