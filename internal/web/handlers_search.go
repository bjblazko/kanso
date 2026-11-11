// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/recipes"
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
	d := docText(p)
	d.Tags = recipes.NormalizeTags(strings.Join(p.Meta.Tags, ","))
	if p.Meta.IsRecipe() {
		d.Kind, d.Rating = search.KindRecipe, p.Meta.Rating
	}
	return d
}

func docText(p wiki.Page) search.Doc {
	body := p.Body
	if len(p.Meta.Tags) > 0 {
		body += "\n\n" + strings.Join(p.Meta.Tags, " ")
	}
	return search.Doc{ID: p.Meta.ID, Space: p.Space, Title: p.Meta.Title, Body: body}
}

func docOfNote(n notes.Note) search.Doc {
	return search.Doc{ID: n.Meta.ID, Owner: n.Owner, Title: notes.Title(n), Body: n.Body}
}

// searchPage is the one search: words, tags, stars and "only recipes" as
// filters; hits grouped into recipes (with photos), pages and own notes.
type searchPage struct {
	filters
	Searched     bool // words or a filter were given
	Recipes      []recipeHit
	Hits         []search.Hit // other pages
	Notes        []search.Hit // own notes
	Count        int
	Filters      []tagLink // the tags of the hits, to switch on and off
	StarFilters  []starLink
	RecipeFilter tagLink
	KitchenURL   string
	Spaces       map[string]string // slug → name
}

type recipeHit struct {
	search.Hit
	Photo string // address of its photo; "" = none
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	data := searchPage{filters: filtersFrom(r), Spaces: map[string]string{}}
	data.Searched = data.Query != "" || len(data.Tags) > 0 || data.OnlyRecipes || data.Stars > 0
	if data.Searched {
		s.fillSearch(&data, s.searchAllowed(r, data))
	}
	for _, sp := range s.Spaces.List() {
		data.Spaces[sp.Slug] = sp.Name
	}
	v := s.newView(r, data.Query, data)
	v.Title = v.T("search.title")
	s.render(w, r, http.StatusOK, "search", v)
}

// searchAllowed lets through what the person may read and the filters accept.
func (s *Server) searchAllowed(r *http.Request, data searchPage) func(search.Doc) bool {
	u, p := auth.UserFrom(r.Context())
	g := s.groups()
	return func(d search.Doc) bool {
		if data.OnlyRecipes && d.Kind != search.KindRecipe || !data.accepts(d.Tags, d.Rating) {
			return false
		}
		if d.Owner != "" {
			return d.Owner == u.Sub
		}
		return access.Can(s.pageRole(p, d.Space, d.ID, g), access.Read)
	}
}

func hasAll(have, want []string) bool {
	for _, t := range want {
		if !slices.Contains(have, t) {
			return false
		}
	}
	return true
}

// fillSearch finds the hits, groups them and offers their tags as filters.
func (s *Server) fillSearch(data *searchPage, allowed func(search.Doc) bool) {
	var hits []search.Hit
	if data.Query != "" {
		hits = s.Search.Search(data.Query, allowed, searchLimit)
	} else {
		hits = s.Search.Browse(allowed, searchLimit)
	}
	var tags []string
	for _, h := range hits {
		tags = append(tags, h.Tags...)
		switch {
		case h.Owner != "":
			data.Notes = append(data.Notes, h)
		case h.Kind == search.KindRecipe:
			data.Recipes = append(data.Recipes, recipeHit{Hit: h, Photo: s.photoOf(h.ID)})
		default:
			data.Hits = append(data.Hits, h)
		}
	}
	data.Count = len(hits)
	f := data.filters
	data.Filters = filterLinks(append(tags, f.Tags...), f.Tags, func(t []string) string { return f.withTags(t).searchURL() })
	if len(data.Recipes) > 0 || f.Stars > 0 {
		data.StarFilters = starLinks(f, filters.searchURL)
	}
	only := f
	only.OnlyRecipes = !f.OnlyRecipes
	data.RecipeFilter = tagLink{Pressed: f.OnlyRecipes, Href: only.searchURL()}
	data.KitchenURL = f.kitchenURL()
}

// photoOf is the address of a recipe page's photo, or "".
func (s *Server) photoOf(id string) string {
	if p, ok := s.Wiki.ByID(id); ok {
		if img := recipePhoto(p); img != "" {
			return "/p/" + id + "/attachments/" + img
		}
	}
	return ""
}
