// SPDX-License-Identifier: Apache-2.0

package todos

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

const (
	garden = "gartenaaaaaa"
	itemA  = "itemaaaaaaaa"
	itemB  = "itembbbbbbbb"
	itemC  = "itemcccccccc"
	itemD  = "itemdddddddd"
)

var noGroups = access.Groups{}

func newStore(t *testing.T) *Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// apply sends ops as sub and fails on a rejection.
func apply(t *testing.T, s *Store, sub string, ops ...Op) {
	t.Helper()
	if rejected := s.Apply(sub, noGroups, ops); len(rejected) > 0 {
		t.Fatalf("rejected %v", rejected)
	}
}

func opID(i int) string { return "opaaaaaaaaa" + string(rune('a'+i%26)) }

func create(id, name string) Op { return Op{ID: opID(0), Kind: "create", List: id, Text: name, At: 1} }

func add(item, after, text string, sub bool) Op {
	return Op{ID: opID(1), Kind: "add", List: garden, Item: item, After: after, Text: text, Sub: sub, At: 1}
}

func mustGet(t *testing.T, s *Store, sub, id string) List {
	t.Helper()
	l, err := s.Get(sub, id, noGroups)
	if err != nil {
		t.Fatal(err)
	}
	return l
}

func texts(l List) []string {
	var out []string
	for _, it := range l.Items {
		out = append(out, it.Text)
	}
	return out
}

// gardenList makes anna's list with Laub (main), Haustür (sub), Garten (sub).
func gardenList(t *testing.T, s *Store) {
	t.Helper()
	apply(t, s, "anna", create(garden, "Gartenarbeit"),
		add(itemA, "", "Laub wegfegen", false),
		add(itemB, itemA, "vor der Haustür", true),
		add(itemC, itemB, "Garten", true))
}

func TestCreateIsIdempotent(t *testing.T) {
	s := newStore(t)
	apply(t, s, "anna", create(garden, "Gartenarbeit"))
	apply(t, s, "anna", create(garden, "Anders"))
	l := mustGet(t, s, "anna", garden)
	if l.Name != "Gartenarbeit" || l.Owner != "anna" || len(s.Visible("anna", noGroups)) != 1 {
		t.Fatalf("got %+v", l)
	}
}

func TestAddAfter(t *testing.T) {
	s := newStore(t)
	apply(t, s, "anna", create(garden, "G"), add(itemA, "", "a", false), add(itemC, "", "c", false),
		add(itemB, itemA, "b", false), add(itemC, "", "c again", false))
	if got := texts(mustGet(t, s, "anna", garden)); !slices.Equal(got, []string{"a", "b", "c"}) {
		t.Fatalf("got %v", got)
	}
}

func TestAddSubAfterMain(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	l := mustGet(t, s, "anna", garden)
	if l.Items[0].Sub || !l.Items[1].Sub || !l.Items[2].Sub {
		t.Fatalf("got %+v", l.Items)
	}
}

func TestDoneStaysInPlace(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "done", List: garden, Item: itemA, Done: true, At: 2})
	l := mustGet(t, s, "anna", garden)
	if !l.Items[0].Done || !slices.Equal(texts(l), []string{"Laub wegfegen", "vor der Haustür", "Garten"}) {
		t.Fatalf("got %+v", l.Items)
	}
}

func TestLaterChangeWins(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "text", List: garden, Item: itemA, Text: "neu", At: 5},
		Op{ID: opID(3), Kind: "text", List: garden, Item: itemA, Text: "alt", At: 3},
		Op{ID: opID(4), Kind: "done", List: garden, Item: itemA, Done: true, At: 4})
	l := mustGet(t, s, "anna", garden)
	if l.Items[0].Text != "neu" || !l.Items[0].Done {
		t.Fatalf("got %+v", l.Items[0])
	}
}

func TestSubNeedsMainBefore(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "sub", List: garden, Item: itemA, Sub: true, At: 2})
	if mustGet(t, s, "anna", garden).Items[0].Sub {
		t.Fatal("first item became a sub-item")
	}
}

func TestDeleteMainDeletesSubs(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", add(itemD, itemC, "Rasen", false),
		Op{ID: opID(2), Kind: "delete", List: garden, Item: itemA, At: 2})
	if got := texts(mustGet(t, s, "anna", garden)); !slices.Equal(got, []string{"Rasen"}) {
		t.Fatalf("got %v", got)
	}
}

func TestDeletedItemNotRevived(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "delete", List: garden, Item: itemC, At: 2},
		Op{ID: opID(3), Kind: "text", List: garden, Item: itemC, Text: "zurück", At: 9},
		add(itemC, itemB, "wieder", true))
	if got := texts(mustGet(t, s, "anna", garden)); !slices.Equal(got, []string{"Laub wegfegen", "vor der Haustür"}) {
		t.Fatalf("got %v", got)
	}
}

func TestTombstonesExpire(t *testing.T) {
	s := newStore(t)
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start }
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(2), Kind: "delete", List: garden, Item: itemC, At: 2})
	s.now = func() time.Time { return start.Add(31 * 24 * time.Hour) }
	apply(t, s, "anna", Op{ID: opID(3), Kind: "done", List: garden, Item: itemA, Done: true, At: 3})
	if l := mustGet(t, s, "anna", garden); len(l.Deleted) != 0 {
		t.Fatalf("tombstones kept: %v", l.Deleted)
	}
}

func TestLimits(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	long := Op{ID: opID(2), Kind: "text", List: garden, Item: itemA, Text: strings.Repeat("x", 501), At: 2}
	name := Op{ID: opID(3), Kind: "rename", List: garden, Text: strings.Repeat("x", 81), At: 2}
	if got := s.Apply("anna", noGroups, []Op{long, name}); len(got) != 2 {
		t.Fatalf("rejected %v", got)
	}
	l := mustGet(t, s, "anna", garden)
	for len(l.Items) < maxItems {
		l.Items = append(l.Items, Item{ID: store.NewID()})
	}
	if err := s.write(l); err != nil {
		t.Fatal(err)
	}
	if got := s.Apply("anna", noGroups, []Op{add(itemD, "", "zu viel", false)}); len(got) != 1 {
		t.Fatalf("1001st item accepted")
	}
}

func TestRenameOwnerOnly(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	if err := s.SetShares("anna", garden, []Share{{User: "ben"}}); err != nil {
		t.Fatal(err)
	}
	rename := Op{ID: opID(2), Kind: "rename", List: garden, Text: "Bens Liste", At: 2}
	if got := s.Apply("ben", noGroups, []Op{rename}); len(got) != 1 {
		t.Fatal("ben renamed anna's list")
	}
	apply(t, s, "ben", Op{ID: opID(3), Kind: "done", List: garden, Item: itemB, Done: true, At: 2})
	apply(t, s, "anna", rename)
	if l := mustGet(t, s, "anna", garden); l.Name != "Bens Liste" || !l.Items[1].Done {
		t.Fatalf("got %+v", l)
	}
}

func TestUpdatedOnChange(t *testing.T) {
	s := newStore(t)
	start := time.Date(2026, 10, 27, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return start }
	gardenList(t, s)
	s.now = func() time.Time { return start.Add(time.Hour) }
	apply(t, s, "anna", Op{ID: opID(2), Kind: "done", List: garden, Item: itemA, Done: true, At: 2})
	if l := mustGet(t, s, "anna", garden); !l.Updated.Equal(start.Add(time.Hour)) {
		t.Fatalf("updated %v", l.Updated)
	}
}
