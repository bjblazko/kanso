// SPDX-License-Identifier: Apache-2.0

package todos

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/access"
)

func TestStrangerSeesNothing(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	if _, err := s.Get("ben", garden, noGroups); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err %v", err)
	}
	if len(s.Visible("ben", noGroups)) != 0 {
		t.Fatal("ben sees anna's list")
	}
}

// Admins have no say over todo lists: the API takes no admin flag at all, and
// an admin group is just a group.
func TestAdminSeesNothing(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	g := access.Groups{"kanso-admin": {"root"}}
	if len(s.Visible("root", g)) != 0 {
		t.Fatal("admin sees anna's list")
	}
}

func TestSharedByGroup(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	if err := s.SetShares("anna", garden, []Share{{Group: "Familie"}}); err != nil {
		t.Fatal(err)
	}
	g := access.Groups{"Familie": {"anna", "ben"}}
	if _, err := s.Get("ben", garden, g); err != nil {
		t.Fatal(err)
	}
	if rejected := s.Apply("ben", g, []Op{{ID: opID(5), Kind: "done", List: garden, Item: itemA, Done: true, At: 2}}); len(rejected) > 0 {
		t.Fatalf("rejected %v", rejected)
	}
}

func TestVisibleNewestFirst(t *testing.T) {
	s := newStore(t)
	start := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start }
	gardenList(t, s)
	s.now = func() time.Time { return start.Add(time.Minute) }
	apply(t, s, "anna", create("weihnachtenb", "Weihnachten"))
	s.now = func() time.Time { return start.Add(2 * time.Minute) }
	apply(t, s, "anna", Op{ID: opID(2), Kind: "done", List: garden, Item: itemA, Done: true, At: 2})
	got := s.Visible("anna", noGroups)
	if len(got) != 2 || got[0].ID != garden {
		t.Fatalf("got %v", got)
	}
}

func TestOpsForInvisibleListRejected(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	op := Op{ID: opID(5), Kind: "done", List: garden, Item: itemA, Done: true, At: 2}
	if got := s.Apply("ben", noGroups, []Op{op}); len(got) != 1 || got[0] != op.ID {
		t.Fatalf("rejected %v", got)
	}
	if mustGet(t, s, "anna", garden).Items[0].Done {
		t.Fatal("ben ticked anna's item")
	}
}

func TestBadOpsRejected(t *testing.T) {
	s := newStore(t)
	ops := []Op{
		{ID: opID(1), Kind: "create", List: "../../etc/x", Text: "x"},
		{ID: opID(2), Kind: "explode", List: garden},
		{ID: opID(3), Kind: "add", List: "nonexistenta", Item: itemA, Text: "x"},
		{ID: opID(4), Kind: "create", List: garden, Text: "  "},
	}
	if got := s.Apply("anna", noGroups, ops); len(got) != 4 {
		t.Fatalf("rejected %v", got)
	}
}

func TestDeleteOwnerOnly(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	if err := s.SetShares("anna", garden, []Share{{User: "ben"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("ben", garden); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ben deleted: %v", err)
	}
	if err := s.SetShares("ben", garden, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("ben changed shares: %v", err)
	}
	if err := s.Delete("anna", garden); err != nil {
		t.Fatal(err)
	}
	if len(s.Visible("anna", noGroups)) != 0 {
		t.Fatal("list still there")
	}
}

func TestGitignore(t *testing.T) {
	s := newStore(t)
	if _, err := NewStore(s.dataDir, s.repo); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(s.dataDir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "/todos/\n") != 1 {
		t.Fatalf(".gitignore: %q", b)
	}
}

func TestDoneCount(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "done", List: garden, Item: itemB, Done: true, At: 2})
	if d, n := Done(mustGet(t, s, "anna", garden)); d != 1 || n != 3 {
		t.Fatalf("%d of %d", d, n)
	}
}
