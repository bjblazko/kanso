// SPDX-License-Identifier: Apache-2.0

package shopping

import (
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/todos"
)

var now = time.UnixMilli(5000)

func items(texts ...string) []todos.Item {
	var out []todos.Item
	for _, t := range texts {
		out = append(out, todos.NewItem(t, "", now))
	}
	return out
}

func ings(lines ...string) []recipes.Ingredient {
	var out []recipes.Ingredient
	for _, l := range lines {
		out = append(out, recipes.ParseIngredient(l))
	}
	return out
}

func TestPlanMerges(t *testing.T) {
	cases := []struct {
		have, add string
		text, old string
	}{
		{"200 g Mehl", "0,5 kg Mehl", "700 g Mehl", "200 g Mehl"},
		{"800 ml Milch", "0,5 l Milch", "1,3 l Milch", "800 ml Milch"},
		{"2 Zwiebeln", "3 Zwiebeln", "5 Zwiebeln", "2 Zwiebeln"},
		{"1 Dose Tomaten", "2 Dosen Tomaten", "3 Dose Tomaten", "1 Dose Tomaten"},
		{"Salz", "1 TL Salz", "Salz", "Salz"},
		{"1 TL Salz", "Salz", "1 TL Salz", "1 TL Salz"},
		{"600 g Mehl", "0,5 kg Mehl", "1,1 kg Mehl", "600 g Mehl"},
	}
	for _, c := range cases {
		have := items(c.have)
		lines := Plan(have, ings(c.add), "de")
		if len(lines) != 1 || lines[0].Text != c.text || lines[0].Old != c.old || lines[0].Item != have[0].ID {
			t.Errorf("%s + %s: %+v", c.have, c.add, lines)
		}
	}
}

func TestPlanKeepsApart(t *testing.T) {
	have := items("2 EL Mehl")
	if l := Plan(have, ings("200 g Mehl"), "de"); l[0].Item != "" || l[0].Text != "200 g Mehl" {
		t.Errorf("incompatible units merged: %+v", l)
	}
	have = items("200 g Mehl")
	have[0].Done = true
	if l := Plan(have, ings("500 g Mehl"), "de"); l[0].Item != "" || l[0].Text != "500 g Mehl" {
		t.Errorf("merged into a bought item: %+v", l)
	}
	if l := Plan(items("Kaffee"), ings("2 Zwiebeln"), "de"); l[0].Item != "" || l[0].Text != "2 Zwiebeln" {
		t.Errorf("unrelated: %+v", l)
	}
}

func TestPlanNewItemsAndDoubles(t *testing.T) {
	l := Plan(nil, ings("- 800 g Kartoffeln, festkochend", "- 100 g Zucker", "- 100 g Zucker", "- Salz", "- 1 ½ Tassen Milch"), "de")
	if len(l) != 4 {
		t.Fatalf("%+v", l)
	}
	if l[0].Text != "800 g Kartoffeln" || l[0].Note != "festkochend" || l[0].Amount != "800 g" || l[0].Name != "Kartoffeln" {
		t.Errorf("kartoffeln %+v", l[0])
	}
	if l[1].Text != "200 g Zucker" || l[1].Amount != "200 g" {
		t.Errorf("zucker %+v", l[1])
	}
	if l[2].Text != "Salz" || l[2].Amount != "" {
		t.Errorf("salz %+v", l[2])
	}
	if l[3].Text != "1 ½ Tassen Milch" {
		t.Errorf("milch %+v", l[3])
	}
}

func TestApply(t *testing.T) {
	list := &todos.List{Items: items("200 g Mehl", "Kaffee")}
	list.Items[1].Done = true
	lines := Plan(list.Items, ings("500 g Mehl", "2 Zwiebeln", "1 Zitrone"), "de")
	src := todos.Source{Recipe: "r1", Title: "Brot"}
	n := Apply(list, lines, func(i int) bool { return i != 2 }, src, now)
	if n != 2 || len(list.Items) != 3 {
		t.Fatalf("%d %+v", n, list.Items)
	}
	mehl, zwiebeln := list.Items[0], list.Items[2]
	if mehl.Text != "700 g Mehl" || mehl.For[0] != (todos.Source{Recipe: "r1", Title: "Brot", Amount: "500 g"}) || mehl.At.Text != 5000 {
		t.Errorf("mehl %+v", mehl)
	}
	if zwiebeln.Text != "2 Zwiebeln" || zwiebeln.For[0].Amount != "2" {
		t.Errorf("zwiebeln %+v", zwiebeln)
	}
	if !list.Items[1].Done || list.Items[1].Text != "Kaffee" {
		t.Error("bought item touched")
	}
}
