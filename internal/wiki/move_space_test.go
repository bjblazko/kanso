// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMoveToOtherSpace(t *testing.T) {
	w := newWiki(t)
	page := mustCreate(t, w, "", "Heizung")
	child := mustCreate(t, w, page.Meta.ID, "Wartung")
	if _, err := w.AddAttachment(anna, page.Meta.ID, "plan.txt", strings.NewReader("x"), 1<<20); err != nil {
		t.Fatal(err)
	}
	var notified []string
	w.OnChange = func(p Page, deleted bool) { notified = append(notified, p.Space+":"+p.Meta.ID) }
	if err := w.MoveTo(anna, page.Meta.ID, "keller", "", 0); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{page.Meta.ID, child.Meta.ID} {
		if p, ok := w.ByID(id); !ok || p.Space != "keller" {
			t.Fatalf("%s: %+v %v", id, p, ok)
		}
	}
	if names := w.ListAttachments(page.Meta.ID); len(names) != 1 {
		t.Fatalf("attachments %v", names)
	}
	if _, err := os.Stat(filepath.Join(w.dataDir, "spaces", "haus", "pages", page.Dir)); !os.IsNotExist(err) {
		t.Fatalf("old folder still there: %v", err)
	}
	if revs, err := w.History(page.Meta.ID, 10); err != nil || len(revs) < 2 {
		t.Fatalf("history after move: %d %v", len(revs), err)
	}
	if len(notified) != 2 || !strings.HasPrefix(notified[0], "keller:") {
		t.Fatalf("search not told: %v", notified)
	}
	if tree := w.Tree("haus"); len(tree) != 0 {
		t.Fatalf("haus still has %d pages", len(tree))
	}
}

func TestMoveToOtherSpaceBelowAPage(t *testing.T) {
	w := newWiki(t)
	page := mustCreate(t, w, "", "Heizung")
	target, err := w.Create(anna, "keller", "", "Technik")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.MoveTo(anna, page.Meta.ID, "keller", target.Meta.ID, 0); err != nil {
		t.Fatal(err)
	}
	if c := w.Chain(page.Meta.ID); len(c) != 2 || c[1].ID != target.Meta.ID {
		t.Fatalf("chain %+v", c)
	}
	if err := w.MoveTo(anna, page.Meta.ID, "haus", target.Meta.ID, 0); err != ErrNotFound {
		t.Fatalf("parent from another space = %v", err)
	}
}

func TestMoveToOtherSpaceKeepsMembers(t *testing.T) {
	w := newWiki(t)
	page := mustCreate(t, w, "", "Wunschliste")
	if err := w.SetMembers(anna, page.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	if err := w.MoveTo(anna, page.Meta.ID, "keller", "", 0); err != nil {
		t.Fatal(err)
	}
	if p, _ := w.ByID(page.Meta.ID); len(p.Meta.Members) != 1 {
		t.Fatalf("members %+v", p.Meta)
	}
}
