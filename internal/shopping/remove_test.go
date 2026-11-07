// SPDX-License-Identifier: Apache-2.0

package shopping

import (
	"testing"

	"github.com/bjblazko/kanso/internal/todos"
)

func fillList(t *testing.T, list *todos.List, fill string, lines ...string) {
	t.Helper()
	src := todos.Source{Recipe: "r-" + fill, Title: fill, Fill: fill}
	Apply(list, Plan(list.Items, ings(lines...), "de"), func(int) bool { return true }, src, now)
}

func TestRemoveOnlySource(t *testing.T) {
	list := &todos.List{}
	fillList(t, list, "a", "500 g Mehl", "Salz")
	if n := Remove(list, "a", "de", now); n != 2 || len(list.Items) != 0 || len(list.Deleted) != 2 {
		t.Fatalf("%d %+v %v", n, list.Items, list.Deleted)
	}
}

func TestRemoveOneOfTwo(t *testing.T) {
	list := &todos.List{}
	fillList(t, list, "a", "200 g Mehl")
	fillList(t, list, "b", "500 g Mehl")
	Remove(list, "b", "de", now)
	if len(list.Items) != 1 || list.Items[0].Text != "200 g Mehl" || len(list.Items[0].For) != 1 || list.Items[0].For[0].Fill != "a" {
		t.Fatalf("%+v", list.Items)
	}
	if list.Items[0].At.Text <= now.UnixMilli() {
		t.Error("text stamp not moved on")
	}
}

func TestRemoveKeepsChangedAndBought(t *testing.T) {
	list := &todos.List{}
	fillList(t, list, "a", "200 g Mehl", "1 Ei", "2 Zwiebeln")
	fillList(t, list, "b", "500 g Mehl", "1 Ei", "1 Zwiebeln")
	list.Items[0].Text = "2 Pck. Mehl" // changed by hand: amount no longer comparable
	list.Items[1].Done = true          // the eggs are bought
	Remove(list, "b", "de", now)
	if list.Items[0].Text != "2 Pck. Mehl" || len(list.Items[0].For) != 1 {
		t.Errorf("changed item: %+v", list.Items[0])
	}
	if list.Items[1].Text != "2 Ei" || len(list.Items[1].For) != 2 {
		t.Errorf("bought item touched: %+v", list.Items[1])
	}
	if list.Items[2].Text != "2 Zwiebeln" {
		t.Errorf("zwiebeln: %+v", list.Items[2])
	}
}
