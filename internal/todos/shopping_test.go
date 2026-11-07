// SPDX-License-Identifier: Apache-2.0

package todos

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestCreateShoppingByOp(t *testing.T) {
	s := newStore(t)
	op := create(garden, "Einkauf")
	op.ListKind = KindShopping
	apply(t, s, "anna", op)
	if l := mustGet(t, s, "anna", garden); l.Kind != KindShopping {
		t.Fatalf("kind %q", l.Kind)
	}
	bad := Op{ID: opID(2), Kind: "create", List: "bbbbbbbbbbbb", Text: "X", ListKind: "wish", At: 1}
	if rejected := s.Apply("anna", noGroups, []Op{bad}); len(rejected) != 1 {
		t.Fatal("unknown kind accepted")
	}
}

func TestNoteOp(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	note := func(text string, at int64) Op {
		return Op{ID: opID(3), Kind: "note", List: garden, Item: itemA, Text: text, At: at}
	}
	apply(t, s, "anna", note("laktosefrei", 5))
	if l := mustGet(t, s, "anna", garden); l.Items[0].Note != "laktosefrei" {
		t.Fatalf("note %q", l.Items[0].Note)
	}
	apply(t, s, "anna", note("älter", 4))
	apply(t, s, "anna", note("", 6))
	if l := mustGet(t, s, "anna", garden); l.Items[0].Note != "" {
		t.Fatalf("not cleared: %q", l.Items[0].Note)
	}
	if rejected := s.Apply("anna", noGroups, []Op{note(strings.Repeat("x", MaxNote+1), 7)}); len(rejected) != 1 {
		t.Fatal("long note accepted")
	}
}

func TestOldListRoundTrip(t *testing.T) {
	s := newStore(t)
	gardenList(t, s)
	apply(t, s, "anna", Op{ID: opID(4), Kind: "done", List: garden, Item: itemA, Done: true, At: 2})
	b, err := os.ReadFile(s.path(garden))
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"kind"`, `"note"`, `"for"`} {
		if strings.Contains(string(b), field) {
			t.Errorf("old list got %s:\n%s", field, b)
		}
	}
}

func TestStoreCreateAndChange(t *testing.T) {
	s := newStore(t)
	l, err := s.Create("anna", "Wocheneinkauf", KindShopping)
	if err != nil || l.Kind != KindShopping || !idPattern.MatchString(l.ID) {
		t.Fatalf("%+v %v", l, err)
	}
	if _, err := s.Create("anna", " ", KindShopping); err == nil {
		t.Error("empty name accepted")
	}
	now := time.UnixMilli(1000)
	err = s.Change("anna", noGroups, l.ID, func(l *List) error {
		it := NewItem("500 g Mehl", "Typ 405", now)
		it.For = []Source{{Recipe: "r1", Title: "Brot", Amount: "500 g"}}
		l.Items = append(l.Items, it)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	got := mustGet(t, s, "anna", l.ID)
	if len(got.Items) != 1 || got.Items[0].Note != "Typ 405" || got.Items[0].For[0].Title != "Brot" || got.Items[0].At.Text != 1000 {
		t.Fatalf("%+v", got.Items)
	}
	if err := s.Change("ben", noGroups, l.ID, func(*List) error { return nil }); !errors.Is(err, ErrNotFound) {
		t.Fatalf("stranger: %v", err)
	}
}

// An offline tick older than a later change of the amount must not tick the larger amount.
func TestOldTickDoesNotTickRaisedAmount(t *testing.T) {
	s := newStore(t)
	l, _ := s.Create("anna", "Einkauf", KindShopping)
	it := NewItem("200 g Mehl", "", time.UnixMilli(1000))
	_ = s.Change("anna", noGroups, l.ID, func(l *List) error { l.Items = append(l.Items, it); return nil })
	_ = s.Change("anna", noGroups, l.ID, func(l *List) error {
		l.Items[0].Text, l.Items[0].At.Text = "700 g Mehl", 3000
		return nil
	})
	apply(t, s, "anna", Op{ID: opID(5), Kind: "done", List: l.ID, Item: it.ID, Done: true, At: 2000})
	if got := mustGet(t, s, "anna", l.ID); got.Items[0].Done {
		t.Fatal("old tick ticked the raised amount")
	}
	apply(t, s, "anna", Op{ID: opID(6), Kind: "done", List: l.ID, Item: it.ID, Done: true, At: 4000})
	if got := mustGet(t, s, "anna", l.ID); !got.Items[0].Done {
		t.Fatal("a newer tick is lost")
	}
}

func TestNewItemLeavesDeviceFieldsUnstamped(t *testing.T) {
	it := NewItem("x", "", time.UnixMilli(5000))
	if it.At.Text != 5000 || it.At.Done != 0 || it.At.Note != 0 {
		t.Fatalf("%+v", it.At)
	}
}
