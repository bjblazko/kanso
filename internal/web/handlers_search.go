// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
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
		d.Kind = search.KindRecipe
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

// searchPage is the one search: words, tags and "only recipes" as filters;
// hits grouped into recipes (with photos), pages and own notes.
type searchPage struct {
	Query        string
	Tags         []string // chosen tag filters
	OnlyRecipes  bool
	Searched     bool // words or a filter were given
	Recipes      []recipeHit
	Hits         []search.Hit // other pages
	Notes        []search.Hit // own notes
	Count        int
	Filters      []tagLink // the tags of the hits, to switch on and off
	RecipeFilter tagLink
	KitchenURL   string
	Spaces       map[string]string // slug → name
}

type recipeHit struct {
	search.Hit
	Photo string // address of its photo; "" = none
}

func (s *Server) search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	data := searchPage{Query: q, Spaces: map[string]string{}, OnlyRecipes: r.URL.Query().Get("type") == search.KindRecipe,
		Tags: recipes.NormalizeTags(strings.Join(r.URL.Query()["tag"], ","))}
	data.Searched = q != "" || len(data.Tags) > 0 || data.OnlyRecipes
	if data.Searched {
		s.fillSearch(&data, s.searchAllowed(r, data))
	}
	for _, sp := range s.Spaces.List() {
		data.Spaces[sp.Slug] = sp.Name
	}
	v := s.newView(r, q, data)
	v.Title = v.T("search.title")
	s.render(w, r, http.StatusOK, "search", v)
}

// searchAllowed lets through what the person may read and the filters accept.
func (s *Server) searchAllowed(r *http.Request, data searchPage) func(search.Doc) bool {
	u, p := auth.UserFrom(r.Context())
	g := s.groups()
	return func(d search.Doc) bool {
		if data.OnlyRecipes && d.Kind != search.KindRecipe || !hasAll(d.Tags, data.Tags) {
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
	data.Filters = filterLinks(append(tags, data.Tags...), data.Tags, func(t []string) string { return searchURL(data.Query, t, data.OnlyRecipes) })
	data.RecipeFilter = tagLink{Pressed: data.OnlyRecipes, Href: searchURL(data.Query, data.Tags, !data.OnlyRecipes)}
	data.KitchenURL = kitchenURL(data.Query, data.Tags)
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

func searchURL(q string, tags []string, onlyRecipes bool) string {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	for _, t := range tags {
		v.Add("tag", t)
	}
	if onlyRecipes {
		v.Set("type", search.KindRecipe)
	}
	return "/search?" + v.Encode()
}
