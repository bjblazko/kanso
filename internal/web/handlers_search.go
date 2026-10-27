// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/search"
	"github.com/bjblazko/kanso/internal/wiki"
)

const searchLimit = 50

// buildIndex fills a search index with every page and note and keeps it current.
func buildIndex(w *wiki.Wiki, n *notes.Store) *search.Index {
	ix := search.NewIndex()
	for _, id := range w.IDs() {
		if p, ok := w.ByID(id); ok {
			ix.Put(docOf(p))
		}
	}
	w.OnChange = func(p wiki.Page, deleted bool) {
		if deleted {
			ix.Remove(p.Meta.ID)
			return
		}
		ix.Put(docOf(p))
	}
	for _, note := range n.All() {
		ix.Put(docOfNote(note))
	}
	n.OnChange = func(note notes.Note, deleted bool) {
		if deleted {
			ix.Remove(note.Meta.ID)
			return
		}
		ix.Put(docOfNote(note))
	}
	return ix
}

// BuildIndex is buildIndex for main.
func BuildIndex(w *wiki.Wiki, n *notes.Store) *search.Index { return buildIndex(w, n) }

func docOf(p wiki.Page) search.Doc {
	return search.Doc{ID: p.Meta.ID, Space: p.Space, Title: p.Meta.Title, Body: p.Body}
}

func docOfNote(n notes.Note) search.Doc {
	return search.Doc{ID: n.Meta.ID, Owner: n.Owner, Title: notes.Title(n), Body: n.Body}
}

type searchPage struct {
	Query  string
	Hits   []search.Hit
	Notes  []search.Hit // own notes
	Count  int
	Spaces map[string]string // slug → name
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	u, p := auth.UserFrom(r.Context())
	g := s.groups()
	allowed := func(d search.Doc) bool {
		if d.Owner != "" {
			return d.Owner == u.Sub
		}
		return access.Can(s.Spaces.RoleOf(p, d.Space, g), access.Read)
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data := searchPage{Query: q, Spaces: map[string]string{}}
	if q != "" {
		data.Hits, data.Notes = splitNotes(s.Search.Search(q, allowed, searchLimit))
		data.Count = len(data.Hits) + len(data.Notes)
	}
	for _, sp := range s.Spaces.List() {
		data.Spaces[sp.Slug] = sp.Name
	}
	v := s.newView(r, q, data)
	v.Title = v.T("search.title")
	s.render(w, r, http.StatusOK, "search", v)
}

// splitNotes separates note hits from page hits, keeping the order.
func splitNotes(hits []search.Hit) (pages, own []search.Hit) {
	for _, h := range hits {
		if h.Owner != "" {
			own = append(own, h)
		} else {
			pages = append(pages, h)
		}
	}
	return pages, own
}
