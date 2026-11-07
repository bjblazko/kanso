// SPDX-License-Identifier: Apache-2.0

// Package shopping fills a shopping list from a recipe (ADR-011): it plans
// which items to add or increase, adding up amounts of the same ingredient
// in compatible units, and applies the lines a person chose.
package shopping

import (
	"slices"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/todos"
)

// Line is one planned change: a new item or more of an existing one.
type Line struct {
	Name   string // the ingredient as the list calls it
	Text   string // the item's text afterwards
	Old    string // the item's text before, when an item is added to; "" = new item
	Item   string // id of the item added to; "" = new item
	Note   string // the ingredient's note, for a new item
	Amount string // how much this recipe asks for, e.g. "500 g"; "" without an amount

	have recipes.Ingredient // the item's amount afterwards
	mine recipes.Ingredient // this recipe's amount
	rest string             // what follows the amount in the text, kept as written
	done bool               // a ticked item: never added to
}

// Plan lists what adding ings to a list with items would change, one line
// per ingredient; several lines for the same ingredient run together.
func Plan(items []todos.Item, ings []recipes.Ingredient, lang string) []Line {
	var existing, lines []Line
	for _, it := range items {
		have := recipes.ParseIngredient(it.Text)
		existing = append(existing, Line{Item: it.ID, Text: it.Text, Old: it.Text, have: have, rest: have.Rest, done: it.Done})
	}
	for _, ing := range ings {
		lines = plan1(existing, lines, ing, lang)
	}
	for i := range lines {
		lines[i].Amount = amount(lines[i].mine)
	}
	return lines
}

// plan1 adds one ingredient: to a planned line, to an existing item, or as a new line.
func plan1(existing, lines []Line, ing recipes.Ingredient, lang string) []Line {
	if i := match(lines, ing); i >= 0 {
		lines[i].add(ing, lang)
		return lines
	}
	if i := match(existing, ing); i >= 0 {
		l := existing[i]
		l.Name, l.mine = l.have.Name, recipes.Ingredient{}
		l.add(ing, lang)
		return append(lines, l)
	}
	return append(lines, Line{Name: ing.Name, Text: text(ing, ing.Name), Note: ing.Note, have: ing, mine: ing, rest: ing.Name})
}

// match finds a line for the same ingredient that may take ing.
func match(lines []Line, ing recipes.Ingredient) int {
	return slices.IndexFunc(lines, func(l Line) bool {
		if l.done || l.have.Key != ing.Key {
			return false
		}
		_, ok := sum(l.have, ing, "de")
		return ok || l.have.Qty == nil || ing.Qty == nil
	})
}

// add puts ing into the line: amounts add up where both have one; where
// only ing has one, the line takes it.
func (l *Line) add(ing recipes.Ingredient, lang string) {
	l.have = combine(l.have, ing, lang)
	l.Text = text(l.have, l.rest)
	if l.mine.Name == "" {
		l.mine = ing
	} else {
		l.mine = combine(l.mine, ing, lang)
	}
}

func combine(a, b recipes.Ingredient, lang string) recipes.Ingredient {
	if s, ok := sum(a, b, lang); ok {
		return s
	}
	if a.Qty == nil && b.Qty != nil {
		return b
	}
	return a
}

// text writes an item: "amount [unit] rest", rest as the person wrote it.
func text(ing recipes.Ingredient, rest string) string {
	if ing.Qty == nil {
		return rest
	}
	return strings.Join(strings.Fields(ing.Qty.Text+" "+ing.UnitText+" "+rest), " ")
}

func amount(ing recipes.Ingredient) string {
	if ing.Qty == nil {
		return ""
	}
	return strings.TrimSpace(ing.Qty.Text + " " + ing.UnitText)
}

// Apply makes the lines take chooses part of the list, each with src as
// where it comes from, and reports how many it applied.
// A fill already on the list (src.Fill) adds nothing again.
func Apply(list *todos.List, lines []Line, take func(i int) bool, src todos.Source, now time.Time) int {
	if src.Fill != "" && Filled([]todos.List{*list}, src.Fill) {
		return 0
	}
	n := 0
	for i, line := range lines {
		if !take(i) {
			continue
		}
		s := src
		s.Amount = line.Amount
		if j := slices.IndexFunc(list.Items, func(it todos.Item) bool { return it.ID == line.Item }); line.Item != "" && j >= 0 {
			it := &list.Items[j]
			if it.Text != line.Text {
				it.Text, it.At.Text = line.Text, max(now.UnixMilli(), it.At.Text+1)
			}
			it.For = append(it.For, s)
		} else {
			it := todos.NewItem(line.Text, line.Note, now)
			s.New = true
			it.For = []todos.Source{s}
			list.Items = append(list.Items, it)
		}
		n++
	}
	return n
}

// Filled tells whether one of lists already holds the fill named fill.
func Filled(lists []todos.List, fill string) bool {
	for _, l := range lists {
		for _, it := range l.Items {
			if slices.ContainsFunc(it.For, func(s todos.Source) bool { return s.Fill == fill }) {
				return true
			}
		}
	}
	return false
}
