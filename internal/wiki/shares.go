// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"cmp"
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// Chain returns the page and the pages above it, the page first; nil for an unknown id.
func (w *Wiki) Chain(id string) []PageMeta {
	w.mu.RLock()
	loc, ok := w.index[id]
	w.mu.RUnlock()
	if !ok {
		return nil
	}
	out := []PageMeta{loc.meta}
	anc := w.Ancestors(id)
	for i := len(anc) - 1; i >= 0; i-- {
		out = append(out, anc[i])
	}
	return out
}

// SetMembers replaces who the page is shared with. Sharing is not editing:
// Updated and the author of the text stay as they are.
func (w *Wiki) SetMembers(a store.Author, id string, members []access.Member) error {
	return w.repo.Locked(func() error {
		p, ok := w.ByID(id)
		if !ok {
			return ErrNotFound
		}
		p.Meta.Members = members
		w.mu.Lock()
		defer w.mu.Unlock()
		return w.write(a, p, "Freigabe von „"+p.Meta.Title+"“ geändert")
	})
}

// Descendants counts the pages below id.
func (w *Wiki) Descendants(id string) int {
	w.mu.RLock()
	defer w.mu.RUnlock()
	loc, ok := w.index[id]
	if !ok {
		return 0
	}
	n := 0
	for _, other := range w.index {
		if other.space == loc.space && strings.HasPrefix(other.dir, loc.dir+"/") {
			n++
		}
	}
	return n
}

// Shared returns every page that has members of its own, without bodies,
// sorted by space and title.
func (w *Wiki) Shared() []Page {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var out []Page
	for _, loc := range w.index {
		if len(loc.meta.Members) > 0 {
			out = append(out, Page{Meta: loc.meta, Space: loc.space, Dir: loc.dir})
		}
	}
	slices.SortFunc(out, func(a, b Page) int {
		return cmp.Or(cmp.Compare(a.Space, b.Space), cmp.Compare(strings.ToLower(a.Meta.Title), strings.ToLower(b.Meta.Title)))
	})
	return out
}
