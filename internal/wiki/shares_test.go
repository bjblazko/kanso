// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

var christina = []access.Member{{User: "christina", Role: access.Viewer}}

func TestSetMembersCommitsAndIndexes(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Wunschliste")
	if err := w.SetMembers(anna, p.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	if len(got.Meta.Members) != 1 || got.Meta.Members[0].User != "christina" {
		t.Fatalf("members %+v", got.Meta.Members)
	}
	if c := w.Chain(p.Meta.ID); len(c) != 1 || len(c[0].Members) != 1 {
		t.Fatalf("index not updated: %+v", c)
	}
	out, _ := exec.Command("git", "-C", w.dataDir, "log", "-1", "--format=%s").Output()
	if !strings.Contains(string(out), "Freigabe von „Wunschliste“ geändert") {
		t.Fatalf("commit %q", out)
	}
	if err := w.SetMembers(anna, "nothing", christina); err != ErrNotFound {
		t.Fatalf("unknown page = %v", err)
	}
}

func TestSaveKeepsMembers(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Wunschliste")
	if err := w.SetMembers(anna, p.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	if err := w.Save(anna, p.Meta.ID, "Wunschliste 2026", "Bücher"); err != nil {
		t.Fatal(err)
	}
	if got, _ := w.ByID(p.Meta.ID); len(got.Meta.Members) != 1 {
		t.Fatalf("members lost: %+v", got.Meta)
	}
}

func TestChainIsPageThenAncestors(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	b := mustCreate(t, w, a.Meta.ID, "B")
	c := mustCreate(t, w, b.Meta.ID, "C")
	chain := w.Chain(c.Meta.ID)
	if len(chain) != 3 || chain[0].ID != c.Meta.ID || chain[1].ID != b.Meta.ID || chain[2].ID != a.Meta.ID {
		t.Fatalf("chain %+v", chain)
	}
	if w.Chain("nothing") != nil {
		t.Fatal("unknown page has a chain")
	}
}

func TestDescendants(t *testing.T) {
	w := newWiki(t)
	a := mustCreate(t, w, "", "A")
	b := mustCreate(t, w, a.Meta.ID, "B")
	mustCreate(t, w, b.Meta.ID, "C")
	mustCreate(t, w, a.Meta.ID, "D")
	mustCreate(t, w, "", "E")
	if n := w.Descendants(a.Meta.ID); n != 3 {
		t.Fatalf("descendants = %d", n)
	}
}

func TestMembersMoveWithPage(t *testing.T) {
	w := newWiki(t)
	shared := mustCreate(t, w, "", "Wunschliste")
	child := mustCreate(t, w, shared.Meta.ID, "Bücher")
	other := mustCreate(t, w, "", "Anderes")
	if err := w.SetMembers(anna, shared.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	if err := w.Move(anna, shared.Meta.ID, other.Meta.ID, 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := w.ByID(shared.Meta.ID); len(got.Meta.Members) != 1 {
		t.Fatalf("members did not move: %+v", got.Meta)
	}
	if err := w.Move(anna, child.Meta.ID, "", 0); err != nil {
		t.Fatal(err)
	}
	for _, m := range w.Chain(child.Meta.ID) {
		if len(m.Members) > 0 {
			t.Fatalf("moved child still inherits: %+v", w.Chain(child.Meta.ID))
		}
	}
}

func TestShared(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Wunschliste")
	mustCreate(t, w, "", "Anderes")
	if err := w.SetMembers(anna, p.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	got := w.Shared()
	if len(got) != 1 || got[0].Meta.ID != p.Meta.ID || got[0].Space != "haus" {
		t.Fatalf("shared %+v", got)
	}
}
