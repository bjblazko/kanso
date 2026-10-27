// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

var anna = store.Author{Name: "Anna", Email: "anna@example.org"}

func newWiki(t *testing.T) *Wiki {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(repo.Dir(), "spaces", "haus"), 0o750); err != nil {
		t.Fatal(err)
	}
	w, err := NewWiki(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func mustCreate(t *testing.T, w *Wiki, parent, title string) Page {
	t.Helper()
	p, err := w.Create(anna, "haus", parent, title)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCreateUniqueSlugsAndVerbatimTitles(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "Über uns")
	b := mustCreate(t, w, "", "Über uns")
	if a.Dir != "ueber-uns" || b.Dir != "ueber-uns-2" {
		t.Errorf("dirs %q %q", a.Dir, b.Dir)
	}
	if a.Meta.Title != "Über uns" || a.Meta.ID == "" || a.Meta.ID == b.Meta.ID || a.Meta.Author != "Anna" {
		t.Errorf("meta %+v", a.Meta)
	}
}

func TestReloadFindsPagesByID(t *testing.T) {
	w := newWiki(t)
	parent := mustCreate(t, w, "", "Garten")
	child := mustCreate(t, w, parent.Meta.ID, "Rasen & Beete")
	again, err := NewWiki(w.dataDir, w.repo)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := again.ByID(child.Meta.ID)
	if !ok || got.Dir != "garten/rasen-beete" || got.Space != "haus" || got.Meta.Title != "Rasen & Beete" {
		t.Errorf("got %+v %v", got, ok)
	}
	tree := again.Tree("haus")
	if len(tree) != 1 || len(tree[0].Children) != 1 || tree[0].Children[0].Meta.ID != child.Meta.ID {
		t.Errorf("tree %+v", tree)
	}
}

func TestSaveKeepsFolder(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Alt")
	if err := w.Save(anna, p.Meta.ID, "Neu", "Text\n"); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	if got.Dir != "alt" || got.Meta.Title != "Neu" || got.Body != "Text\n" {
		t.Errorf("got %+v", got)
	}
}

func TestMoveKeepsHistoryAndChildren(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	b := mustCreate(t, w, "", "B")
	c := mustCreate(t, w, a.Meta.ID, "C")
	if err := w.Save(anna, a.Meta.ID, "A", "eins\n"); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(anna, a.Meta.ID, b.Meta.ID, 0); err != nil {
		t.Fatal(err)
	}
	moved, _ := w.ByID(a.Meta.ID)
	child, _ := w.ByID(c.Meta.ID)
	if moved.Dir != "b/a" || child.Dir != "b/a/c" {
		t.Errorf("dirs %q %q", moved.Dir, child.Dir)
	}
	revs, _ := w.repo.Log(filepath.Join(pagePath("haus", moved.Dir)), 10)
	if len(revs) < 3 {
		t.Errorf("history after move: %d revisions", len(revs))
	}
}

func TestMoveIntoOwnSubtreeFails(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	c := mustCreate(t, w, a.Meta.ID, "C")
	if err := w.Move(anna, a.Meta.ID, c.Meta.ID, 0); err == nil {
		t.Error("moved into own child")
	}
	if err := w.Move(anna, a.Meta.ID, a.Meta.ID, 0); err == nil {
		t.Error("moved into itself")
	}
}

func TestDeleteOnlyLeaves(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	c := mustCreate(t, w, a.Meta.ID, "C")
	if err := w.Delete(anna, a.Meta.ID); err == nil {
		t.Error("deleted page with children")
	}
	if err := w.Delete(anna, c.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, ok := w.ByID(c.Meta.ID); ok {
		t.Error("deleted page still indexed")
	}
	if err := w.Delete(anna, a.Meta.ID); err != nil {
		t.Fatal(err)
	}
}

func TestCreateUnderUnknownParentFails(t *testing.T) {
	w := newWiki(t)
	if _, err := w.Create(anna, "haus", "nope", "X"); err == nil {
		t.Error("unknown parent accepted")
	}
	if _, err := w.Create(anna, "haus", "", "   "); err == nil {
		t.Error("empty title accepted")
	}
}

func TestOrderSortsTree(t *testing.T) {
	w := newWiki(t)
	mustCreate(t, w, "", "Zebra")
	b := mustCreate(t, w, "", "Apfel")
	if err := w.Move(anna, b.Meta.ID, "", 5); err != nil {
		t.Fatal(err)
	}
	tree := w.Tree("haus")
	if tree[0].Meta.Title != "Zebra" {
		t.Errorf("order: first is %q", tree[0].Meta.Title)
	}
}
