// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

var ben = store.Author{Name: "Ben", Email: "ben@example.org"}

func TestHistoryDiffRestore(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Heizung")
	w.Save(anna, p.Meta.ID, "Heizung", "eins\n")
	w.Save(ben, p.Meta.ID, "Heizung", "zwei\n")
	revs, err := w.History(p.Meta.ID, 50)
	if err != nil || len(revs) != 3 {
		t.Fatalf("history = %+v, %v", revs, err)
	}
	if revs[0].Author.Name != "Ben" || revs[1].Author.Name != "Anna" {
		t.Errorf("authors %s %s", revs[0].Author.Name, revs[1].Author.Name)
	}
	lines, err := w.Diff(p.Meta.ID, revs[1].Hash, revs[0].Hash)
	if err != nil {
		t.Fatal(err)
	}
	if !hasLine(lines, '-', "eins") || !hasLine(lines, '+', "zwei") {
		t.Errorf("diff = %+v", lines)
	}
	old, err := w.Revision(p.Meta.ID, revs[1].Hash)
	if err != nil || old.Body != "eins\n" {
		t.Errorf("revision = %+v, %v", old, err)
	}
	if err := w.Restore(anna, p.Meta.ID, revs[1].Hash); err != nil {
		t.Fatal(err)
	}
	cur, _ := w.ByID(p.Meta.ID)
	after, _ := w.History(p.Meta.ID, 50)
	if cur.Body != "eins\n" || len(after) != 4 {
		t.Errorf("after restore: %q, %d revisions", cur.Body, len(after))
	}
}

func TestHistorySurvivesMove(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	b := mustCreate(t, w, "", "B")
	w.Save(anna, a.Meta.ID, "A", "vorher\n")
	w.Move(anna, a.Meta.ID, b.Meta.ID, 1)
	revs, _ := w.History(a.Meta.ID, 50)
	oldest := revs[len(revs)-1]
	if _, err := w.Revision(a.Meta.ID, oldest.Hash); err != nil {
		t.Errorf("revision before move: %v", err)
	}
	if _, err := w.Diff(a.Meta.ID, oldest.Hash, revs[0].Hash); err != nil {
		t.Errorf("diff across move: %v", err)
	}
}

func TestRevisionRejectsForeignHashes(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	b := mustCreate(t, w, "", "B")
	revsB, _ := w.History(b.Meta.ID, 50)
	if _, err := w.Revision(a.Meta.ID, revsB[0].Hash); err == nil {
		t.Error("revision of another page accepted")
	}
	if _, err := w.Revision(a.Meta.ID, "HEAD;rm"); err == nil {
		t.Error("bad hash accepted")
	}
}

func hasLine(lines []DiffLine, kind byte, text string) bool {
	for _, l := range lines {
		if l.Kind == kind && l.Text == text {
			return true
		}
	}
	return false
}

func TestDiffIgnoresFrontmatter(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "A")
	w.Save(anna, p.Meta.ID, "A", "text\n")
	w.Save(ben, p.Meta.ID, "A", "text\n")
	revs, _ := w.History(p.Meta.ID, 10)
	lines, err := w.Diff(p.Meta.ID, revs[1].Hash, revs[0].Hash)
	if err != nil || len(lines) != 0 {
		t.Errorf("diff of equal bodies = %+v, %v", lines, err)
	}
}
