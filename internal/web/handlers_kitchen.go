// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/wiki"
)

// firstPhoto finds the first attached photo in a page's text.
var firstPhoto = regexp.MustCompile(`!\[[^\]]*\]\(attachments/([^)\s]+)\)`)

// recipePhoto is the attachment name of a recipe's photo: its image, else
// the first photo in its text (recipes typed by hand); "" = none.
func recipePhoto(p wiki.Page) string {
	if p.Meta.Image != "" {
		return p.Meta.Image
	}
	if m := firstPhoto.FindStringSubmatch(p.Body); m != nil {
		return m[1]
	}
	return ""
}

// newRecipeServings is what a new recipe starts with.
const newRecipeServings = 4

// recipeRow is one recipe in the kitchen, with the words it can be found by.
type recipeRow struct {
	ID, Title, Space string
	Image            string // attachment name of its photo; "" = none
	Tags             []string
	Time             int
	words            []string // title, tags and ingredient names, lower case
}

type tagLink struct {
	Name, Href string
	Pressed    bool
}

type kitchenPage struct {
	Rows     []recipeRow
	Tags     []tagLink
	Query    string
	Selected []string
	Any      bool // the person can read at least one recipe
}

// visibleRecipes lists every recipe page the person may read, by title.
func (s *Server) visibleRecipes(who access.Principal) []recipeRow {
	g := s.groups()
	var rows []recipeRow
	for _, id := range s.Wiki.IDs() {
		p, ok := s.Wiki.ByID(id)
		if !ok || !p.Meta.IsRecipe() || !access.Can(s.pageRole(who, p.Space, id, g), access.Read) {
			continue
		}
		rows = append(rows, s.rowOf(p))
	}
	sort.Slice(rows, func(i, j int) bool { return strings.ToLower(rows[i].Title) < strings.ToLower(rows[j].Title) })
	return rows
}

func (s *Server) rowOf(p wiki.Page) recipeRow {
	tags := recipes.NormalizeTags(strings.Join(p.Meta.Tags, ",")) // files edited outside Kanso may say "Deftig"
	row := recipeRow{ID: p.Meta.ID, Title: p.Meta.Title, Tags: tags, Time: p.Meta.Time, Space: p.Space, Image: recipePhoto(p)}
	if sp, ok := s.Spaces.Get(p.Space); ok {
		row.Space = sp.Name
	}
	row.words = append(strings.Fields(strings.ToLower(p.Meta.Title)), tags...)
	for _, ing := range recipes.Parse(p.Body).Ingredients() {
		row.words = append(row.words, strings.Fields(ing.Key)...)
	}
	return row
}

// matchRecipe tells whether row carries all tags and every word of q starts one of its words.
func matchRecipe(row recipeRow, q string, tags []string) bool {
	for _, t := range tags {
		if !slices.Contains(row.Tags, t) {
			return false
		}
	}
	for _, term := range strings.Fields(strings.ToLower(q)) {
		if !slices.ContainsFunc(row.words, func(w string) bool { return strings.HasPrefix(w, term) }) {
			return false
		}
	}
	return true
}

func (s *Server) kitchen(w http.ResponseWriter, r *http.Request) {
	_, who := auth.UserFrom(r.Context())
	all := s.visibleRecipes(who)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	selected := recipes.NormalizeTags(strings.Join(r.URL.Query()["tag"], ","))
	kp := kitchenPage{Query: q, Selected: selected, Any: len(all) > 0, Tags: tagLinks(all, q, selected)}
	for _, row := range all {
		if matchRecipe(row, q, selected) {
			kp.Rows = append(kp.Rows, row)
		}
	}
	v := s.newView(r, "", kp)
	v.Title = v.T("nav.kitchen")
	s.render(w, r, http.StatusOK, "recipes", v)
}

// tagLinks offers every tag of the visible recipes as a link that switches it on or off.
func tagLinks(rows []recipeRow, q string, selected []string) []tagLink {
	var names []string
	for _, row := range rows {
		for _, t := range row.Tags {
			if !slices.Contains(names, t) {
				names = append(names, t)
			}
		}
	}
	sort.Strings(names)
	links := make([]tagLink, 0, len(names))
	for _, n := range names {
		on := slices.Contains(selected, n)
		links = append(links, tagLink{Name: n, Pressed: on, Href: kitchenURL(q, toggle(selected, n, on))})
	}
	return links
}

func toggle(tags []string, t string, on bool) []string {
	if on {
		return slices.DeleteFunc(slices.Clone(tags), func(x string) bool { return x == t })
	}
	return append(slices.Clone(tags), t)
}

func kitchenURL(q string, tags []string) string {
	v := url.Values{}
	if q != "" {
		v.Set("q", q)
	}
	for _, t := range tags {
		v.Add("tag", t)
	}
	if len(v) == 0 {
		return "/recipes"
	}
	return "/recipes?" + v.Encode()
}

type newRecipeForm struct {
	Spaces []spaces.Space
	Space  string
	Title  string
	Error  string
}

// writableSpaces are the spaces the person may create pages in.
func (s *Server) writableSpaces(who access.Principal) []spaces.Space {
	var out []spaces.Space
	for _, sp := range s.Spaces.List() {
		if access.Can(s.Spaces.RoleOf(who, sp.Slug, s.groups()), access.Write) {
			out = append(out, sp)
		}
	}
	return out
}

func (s *Server) newRecipeForm(w http.ResponseWriter, r *http.Request) {
	_, who := auth.UserFrom(r.Context())
	v := s.newView(r, "", newRecipeForm{Spaces: s.writableSpaces(who)})
	v.Title = v.T("recipe.new")
	s.render(w, r, http.StatusOK, "recipe_new", v)
}

// createRecipe makes a recipe page from the skeleton and opens its editor.
func (s *Server) createRecipe(w http.ResponseWriter, r *http.Request) {
	u, who := auth.UserFrom(r.Context())
	space := r.PostFormValue("space")
	if _, ok := s.Spaces.Get(space); !ok || !access.Can(s.Spaces.RoleOf(who, space, s.groups()), access.Write) {
		s.forbidden(w, r)
		return
	}
	v := s.newView(r, "", nil)
	p, err := s.Wiki.Create(u.Author(), space, "", r.PostFormValue("title"))
	if err == nil {
		err = s.Wiki.SaveWith(u.Author(), p.Meta.ID, p.Meta.Title, v.T("recipe.skeleton")+"\n", func(m *wiki.PageMeta) {
			m.Type, m.Servings = wiki.TypeRecipe, newRecipeServings
		})
	}
	switch {
	case errors.Is(err, wiki.ErrNoTitle):
		v.Title, v.Data = v.T("recipe.new"), newRecipeForm{Spaces: s.writableSpaces(who), Space: space, Error: v.T(pageErrorKey(err))}
		s.render(w, r, http.StatusBadRequest, "recipe_new", v)
	case err != nil:
		s.serverError(w, r, err)
	default:
		http.Redirect(w, r, "/p/"+p.Meta.ID+"/edit", http.StatusSeeOther)
	}
}
