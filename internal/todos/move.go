// SPDX-License-Identifier: Apache-2.0

package todos

import (
	"slices"

	"github.com/bjblazko/kanso/internal/access"
)

// move takes an item (with its sub-items) from the shopping list from to the
// shopping list op.To, unticked. Sent again, it finds the item gone and does nothing.
func (s *Store) move(sub string, g access.Groups, from *List, op Op, lists map[string]*List, changed map[string]bool) error {
	if op.To == from.ID {
		return errRejected
	}
	to, err := s.listFor(sub, g, Op{List: op.To}, lists)
	if err != nil {
		return err
	}
	if from.Kind != KindShopping || to.Kind != KindShopping {
		return errRejected
	}
	i := from.index(op.Item)
	if i < 0 {
		return nil // moved or deleted before
	}
	moved := slices.Clone(from.Items[i:end(from, i)])
	if len(to.Items)+len(moved) > maxItems {
		return errRejected
	}
	now := s.now()
	deleteItem(from, op, now)
	for _, it := range moved {
		s.put(to, it, op.Lang, now.UnixMilli())
	}
	changed[from.ID], changed[to.ID] = true, true
	return nil
}

// put adds a moved item to l: into a like item through Join, else at the end.
func (s *Store) put(l *List, it Item, lang string, now int64) {
	it.Done = false
	it.At = Stamps{Text: now, Done: now, Sub: now, Note: now}
	if !it.Sub && s.Join != nil && s.Join(l, it, lang) {
		return
	}
	it.Sub = it.Sub && len(l.Items) > 0
	l.Items = append(l.Items, it)
}

// end is the index after the item at i and its sub-items.
func end(l *List, i int) int {
	e := i + 1
	for !l.Items[i].Sub && e < len(l.Items) && l.Items[e].Sub {
		e++
	}
	return e
}
