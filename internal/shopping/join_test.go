// SPDX-License-Identifier: Apache-2.0

package shopping

import (
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/todos"
)

func TestJoinAddsUpALikeItem(t *testing.T) {
	list := todos.List{Kind: todos.KindShopping, Items: items("2 Zwiebeln", "200 g Mehl")}
	moved := todos.NewItem("0,5 kg Mehl", "Type 405", now)
	moved.For = []todos.Source{{Recipe: "r1", Title: "Brot", Amount: "0,5 kg"}}
	if !Join(&list, moved, "de", time.UnixMilli(9000)) {
		t.Fatal("not joined")
	}
	it := list.Items[1]
	if len(list.Items) != 2 || it.Text != "700 g Mehl" || it.Note != "Type 405" || len(it.For) != 1 || it.At.Text != 9000 {
		t.Fatalf("%+v", list.Items)
	}
}

func TestJoinLeavesOthersApart(t *testing.T) {
	list := todos.List{Kind: todos.KindShopping, Items: items("200 g Mehl", "Salz")}
	list.Items[0].Done = true // bought already: a new item instead
	for _, text := range []string{"500 g Mehl", "1 Dose Tomaten"} {
		if Join(&list, todos.NewItem(text, "", now), "de", now) {
			t.Errorf("%s joined", text)
		}
	}
}
