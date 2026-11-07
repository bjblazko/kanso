// SPDX-License-Identifier: Apache-2.0

package shopping

import (
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/todos"
)

// Remove takes what the fill named fill added off the list again: an item
// only it asked for goes, otherwise its amount shrinks by the fill's share
// where the units still compare. Ticked (bought) items stay as they are.
// It reports how many items it changed.
func Remove(list *todos.List, fill, lang string, now time.Time) int {
	n := 0
	kept := list.Items[:0]
	for _, it := range list.Items {
		i := slices.IndexFunc(it.For, func(s todos.Source) bool { return s.Fill == fill })
		switch {
		case i < 0 || it.Done:
			kept = append(kept, it)
			continue
		case len(it.For) == 1 || !shrink(&it, it.For[i].Amount, lang, now):
			tombstone(list, it.ID, now)
		default:
			it.For = slices.Delete(it.For, i, i+1)
			kept = append(kept, it)
		}
		n++
	}
	list.Items = kept
	return n
}

// shrink takes amount off the item's amount; false means nothing is left.
// An amount that no longer compares leaves the text as it is.
func shrink(it *todos.Item, amount, lang string, now time.Time) bool {
	if amount == "" {
		return true
	}
	have := recipes.ParseIngredient(it.Text)
	part := recipes.ParseIngredient(amount + " " + have.Name)
	if part.Qty == nil {
		return true
	}
	neg := *part.Qty
	neg.Low, neg.High = -neg.Low, -neg.High
	part.Qty = &neg
	left, ok := sum(have, part, lang)
	if !ok {
		return true
	}
	if left.Qty.High <= 0 {
		return false
	}
	it.Text, it.At.Text = text(left, have.Rest), max(now.UnixMilli(), it.At.Text+1)
	return true
}

func tombstone(list *todos.List, id string, now time.Time) {
	if list.Deleted == nil {
		list.Deleted = map[string]int64{}
	}
	list.Deleted[id] = now.UnixMilli()
}
