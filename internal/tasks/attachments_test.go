// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readAttachment(t *testing.T, s *Store, a Actor, board, task, name string) string {
	t.Helper()
	f, err := s.OpenAttachment(a, board, task, name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()
	b, _ := io.ReadAll(f)
	return string(b)
}

func TestAttachmentAddOpen(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	name, err := s.AddAttachment(actor("anna"), tk.Board, tk.Meta.ID, "Bon.PDF", strings.NewReader("%PDF"), 1<<20)
	if err != nil || name != "bon.pdf" {
		t.Fatalf("add = %q, %v", name, err)
	}
	if got := readAttachment(t, s, actor("anna"), tk.Board, tk.Meta.ID, name); got != "%PDF" {
		t.Fatalf("read %q", got)
	}
	if log := gitLog(t, s, attachmentRel(tk.Board, tk.Meta.ID)); len(log) != 1 {
		t.Fatalf("log %v", log)
	}
}

func TestAttachmentTooLarge(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	if _, err := s.AddAttachment(actor("anna"), tk.Board, tk.Meta.ID, "a.txt", strings.NewReader("12345"), 3); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err %v", err)
	}
}

func TestAttachmentInvisibleBoard(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Privat"})
	name, _ := s.AddAttachment(actor("anna"), tk.Board, tk.Meta.ID, "a.txt", strings.NewReader("x"), 1<<20)
	if _, err := s.OpenAttachment(actor("ben"), tk.Board, tk.Meta.ID, name); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ben open: %v", err)
	}
	if _, err := s.AddAttachment(actor("ben"), tk.Board, tk.Meta.ID, "b.txt", strings.NewReader("x"), 1<<20); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ben add: %v", err)
	}
}

func TestAttachmentBadName(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	for _, n := range []string{"../board.yaml", "", "a/b.txt"} {
		if _, err := s.OpenAttachment(actor("anna"), tk.Board, tk.Meta.ID, n); !errors.Is(err, ErrNotFound) {
			t.Errorf("%q: %v", n, err)
		}
	}
}

func TestMoveTakesAttachments(t *testing.T) {
	s := newStore(t)
	fam := familyBoard(t, s)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	name, _ := s.AddAttachment(actor("anna"), tk.Board, tk.Meta.ID, "foto.jpg", strings.NewReader("jpg"), 1<<20)
	if _, err := s.Move(actor("anna"), tk.Board, tk.Meta.ID, fam.ID); err != nil {
		t.Fatal(err)
	}
	if got := readAttachment(t, s, actor("ben"), fam.ID, tk.Meta.ID, name); got != "jpg" {
		t.Fatalf("after move %q", got)
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, attachmentRel(tk.Board, tk.Meta.ID))); !os.IsNotExist(err) {
		t.Fatalf("old folder left: %v", err)
	}
}

func TestDeleteRemovesAttachments(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	_, _ = s.AddAttachment(actor("anna"), tk.Board, tk.Meta.ID, "a.txt", strings.NewReader("x"), 1<<20)
	if err := s.Delete(actor("anna"), tk.Board, tk.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, attachmentRel(tk.Board, tk.Meta.ID))); !os.IsNotExist(err) {
		t.Fatalf("folder left: %v", err)
	}
	out, _ := gitStatus(t, s)
	if out != "" {
		t.Fatalf("uncommitted: %q", out)
	}
}

func TestAddAttachmentsOneCommit(t *testing.T) {
	s := newStore(t)
	tk := mustCreate(t, s, actor("anna"), New{Title: "Tasche"})
	names, err := s.AddAttachments(actor("anna"), tk.Board, tk.Meta.ID,
		[]File{{Name: "a.jpg", R: strings.NewReader("1")}, {Name: "b.jpg", R: strings.NewReader("2")}}, 1<<20)
	if err != nil || strings.Join(names, ",") != "a.jpg,b.jpg" {
		t.Fatalf("names %v, %v", names, err)
	}
	if log := gitLog(t, s, attachmentRel(tk.Board, tk.Meta.ID)); len(log) != 1 {
		t.Fatalf("log %v", log)
	}
}
