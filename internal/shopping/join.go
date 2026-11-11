// SPDX-License-Identifier: Apache-2.0

package shopping

import (
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/todos"
)

// Join adds an item moved from another list to a like, unticked item on
// list, adding up the amounts as filling from a recipe does; it keeps the
// item's note if list's has none and takes along whom it is for. It
// reports false when list has no such item.
func Join(list *todos.List, it todos.Item, lang string, now time.Time) bool {
	line := Plan(list.Items, []recipes.Ingredient{recipes.ParseIngredient(it.Text)}, lang)[0]
	j := slices.IndexFunc(list.Items, func(x todos.Item) bool { return x.ID == line.Item })
	if line.Item == "" || j < 0 {
		return false
	}
	have := &list.Items[j]
	stamp := max(now.UnixMilli(), have.At.Text+1)
	if have.Text != line.Text {
		have.Text, have.At.Text = line.Text, stamp
	}
	if have.Note == "" && it.Note != "" {
		have.Note, have.At.Note = it.Note, stamp
	}
	have.For = append(have.For, it.For...)
	return true
}
