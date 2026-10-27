// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// noteFolder writes a note-like folder with one photo below the data dir.
func noteFolder(t *testing.T, w *Wiki) string {
	t.Helper()
	src := filepath.Join(w.dataDir, "notes", "anna", "abcdefghijkl")
	if err := os.MkdirAll(filepath.Join(src, "attachments"), 0o750); err != nil {
		t.Fatal(err)
	}
	note := "---\nid: abcdefghijkl\ncreated: 2026-10-27T10:00:00Z\nupdated: 2026-10-27T10:00:00Z\n---\nText\n\n![](attachments/a.png)\n"
	if err := os.WriteFile(filepath.Join(src, "index.md"), []byte(note), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "attachments", "a.png"), []byte("png"), 0o640); err != nil {
		t.Fatal(err)
	}
	return src
}

func TestAdoptMovesFolderAndCommits(t *testing.T) {
	w := newWiki(t)
	parent := mustCreate(t, w, "", "Heizung")
	src := noteFolder(t, w)
	p, err := w.Adopt(anna, "haus", parent.Meta.ID, "Ablesen", src)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := w.ByID(p.Meta.ID)
	if !ok || got.Body != "Text\n\n![](attachments/a.png)\n" || got.Meta.Title != "Ablesen" || got.Meta.ID == "abcdefghijkl" {
		t.Fatalf("page = %+v %v", got, ok)
	}
	if anc := w.Ancestors(p.Meta.ID); len(anc) != 1 || anc[0].ID != parent.Meta.ID {
		t.Errorf("ancestors = %+v", anc)
	}
	if !slices.Equal(w.ListAttachments(p.Meta.ID), []string{"a.png"}) {
		t.Errorf("attachments = %v", w.ListAttachments(p.Meta.ID))
	}
	if _, err := os.Stat(src); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("note folder still there: %v", err)
	}
	revs, err := w.History(p.Meta.ID, 10)
	if err != nil || len(revs) != 1 || revs[0].Message != "Seite aus Notiz angelegt: Ablesen" {
		t.Errorf("history = %+v, %v", revs, err)
	}
}

func TestAdoptNeedsTitleAndLeavesNoteAlone(t *testing.T) {
	w := newWiki(t)
	src := noteFolder(t, w)
	if _, err := w.Adopt(anna, "haus", "", "  ", src); !errors.Is(err, ErrNoTitle) {
		t.Errorf("err = %v", err)
	}
	if _, err := w.Adopt(anna, "haus", "unknownparen", "X", src); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown parent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(src, "index.md")); err != nil {
		t.Errorf("note folder touched: %v", err)
	}
}
