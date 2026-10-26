// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/search"
	"github.com/bjblazko/kanso/internal/wiki"
)

const searchLimit = 50

// buildIndex fills a search index with every page and keeps it current.
func buildIndex(w *wiki.Wiki) *search.Index {
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
	return ix
}

// BuildIndex is buildIndex for main.
func BuildIndex(w *wiki.Wiki) *search.Index { return buildIndex(w) }

func docOf(p wiki.Page) search.Doc {
	return search.Doc{ID: p.Meta.ID, Space: p.Space, Title: p.Meta.Title, Body: p.Body}
}

type searchPage struct {
	Query  string
	Hits   []search.Hit
	Spaces map[string]string // slug → name
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	_, p := auth.UserFrom(r.Context())
	g := s.groups()
	readable := func(space string) bool { return access.Can(s.Spaces.RoleOf(p, space, g), access.Read) }
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data := searchPage{Query: q, Spaces: map[string]string{}}
	if q != "" {
		data.Hits = s.Search.Search(q, readable, searchLimit)
	}
	for _, sp := range s.Spaces.List() {
		data.Spaces[sp.Slug] = sp.Name
	}
	v := s.newView(r, q, data)
	v.Title = v.T("search.title")
	s.render(w, r, http.StatusOK, "search", v)
}
