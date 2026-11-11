// SPDX-License-Identifier: Apache-2.0

package todos

import "testing"

const market = "marktaaaaaaa"

// twoShoppingLists makes garden and market shopping lists; garden holds itemA "500 g Mehl", ticked.
func twoShoppingLists(t *testing.T, s *Store) {
	t.Helper()
	g, m := create(garden, "Einkauf KW 46"), create(market, "Wochenmarkt")
	g.ListKind, m.ListKind, m.ID = KindShopping, KindShopping, opID(5)
	apply(t, s, "anna", g, m, add(itemA, "", "500 g Mehl", false),
		Op{ID: opID(6), Kind: "note", List: garden, Item: itemA, Text: "Type 405", At: 2},
		Op{ID: opID(7), Kind: "done", List: garden, Item: itemA, Done: true, At: 3})
}

func TestMoveTakesTheItemAcross(t *testing.T) {
	s := newStore(t)
	twoShoppingLists(t, s)
	move := Op{ID: opID(8), Kind: "move", List: garden, Item: itemA, To: market, At: 4}
	apply(t, s, "anna", move)
	if g := mustGet(t, s, "anna", garden); len(g.Items) != 0 || g.Deleted[itemA] == 0 {
		t.Fatalf("still on the old list: %+v", g)
	}
	m := mustGet(t, s, "anna", market)
	if len(m.Items) != 1 || m.Items[0].Text != "500 g Mehl" || m.Items[0].Note != "Type 405" || m.Items[0].Done {
		t.Fatalf("%+v", m.Items)
	}
	apply(t, s, "anna", move) // sent again: nothing more happens
	if m := mustGet(t, s, "anna", market); len(m.Items) != 1 {
		t.Fatalf("twice: %+v", m.Items)
	}
}

func TestMoveJoinsThroughTheHook(t *testing.T) {
	s := newStore(t)
	twoShoppingLists(t, s)
	var got Item
	s.Join = func(dst *List, it Item, lang string) bool { got = it; return lang == "de" }
	apply(t, s, "anna", Op{ID: opID(8), Kind: "move", List: garden, Item: itemA, To: market, Lang: "de", At: 4})
	if got.ID != itemA || len(mustGet(t, s, "anna", market).Items) != 0 {
		t.Fatalf("hook not used: %+v", got)
	}
}

func TestMoveOnlyBetweenShoppingLists(t *testing.T) {
	s := newStore(t)
	twoShoppingLists(t, s)
	plain := create("planaaaaaaaa", "Todos")
	plain.ID = opID(9)
	apply(t, s, "anna", plain)
	for _, to := range []string{"planaaaaaaaa", garden, "unknownaaaaa"} {
		op := Op{ID: opID(10), Kind: "move", List: garden, Item: itemA, To: to, At: 4}
		if r := s.Apply("anna", noGroups, []Op{op}); len(r) != 1 {
			t.Errorf("move to %s accepted", to)
		}
	}
	if r := s.Apply("ben", noGroups, []Op{{ID: opID(11), Kind: "move", List: garden, Item: itemA, To: market, At: 4}}); len(r) != 1 {
		t.Error("a stranger moved it")
	}
}
