// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/shopping"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/todos"
)

// newList is the form value for "a new shopping list".
const newList = "new"

// shopView is the page that adds a recipe's ingredients to a shopping list (ADR-011).
type shopView struct {
	pageCtx
	Servings int
	Lists    []todos.List // shopping lists the person sees
	List     string       // chosen list id or newList
	Name     string       // name of a new list
	Lines    []shopping.Line
	Fill     string   // names this "add to list", so sending it twice adds nothing
	Taken    []string // line numbers ticked when the preview was refreshed; nil = all

	ings []recipes.Ingredient // scaled to Servings
	lang string
}

// Takes tells the template whether line i is ticked.
func (sv shopView) Takes(i int) bool {
	return sv.Taken == nil || slices.Contains(sv.Taken, strconv.Itoa(i))
}

// shopForm reads the choice from a request and plans the lines; ok is false
// after an answer was written (recipe or list not found).
func (s *Server) shopForm(w http.ResponseWriter, r *http.Request) (shopView, bool) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok {
		return shopView{}, false
	}
	if !pc.Page.Meta.IsRecipe() {
		s.notFound(w, r)
		return shopView{}, false
	}
	u, _ := auth.UserFrom(r.Context())
	sv := shopView{pageCtx: pc, Lists: s.shoppingLists(u.Sub), List: r.FormValue("list"), Name: strings.TrimSpace(r.FormValue("name"))}
	base := max(pc.Page.Meta.Servings, 1)
	sv.Servings = recipes.ClampServings(r.FormValue("servings"), base)
	items, ok := sv.chosenItems()
	if !ok {
		s.notFound(w, r)
		return shopView{}, false
	}
	v := s.newView(r, "", nil)
	sv.lang = v.Lang
	if sv.Name == "" {
		sv.Name = v.T("shop.list_name", pc.Page.Meta.Title) // derived, not asked for
	}
	for _, ing := range recipes.Parse(pc.Page.Body).Ingredients() {
		sv.ings = append(sv.ings, ing.Scale(float64(sv.Servings)/float64(base), sv.lang))
	}
	sv.Lines = shopping.Plan(items, sv.ings, sv.lang)
	if sv.Fill = r.PostFormValue("fill"); sv.Fill == "" {
		sv.Fill = store.NewID()
	}
	return sv, true
}

// chosenItems picks the list (the newest one when none is chosen) and returns its items.
func (sv *shopView) chosenItems() ([]todos.Item, bool) {
	var items []todos.Item
	var ok bool
	sv.List, items, ok = pickList(sv.Lists, sv.List)
	return items, ok
}

// pickList resolves the chosen list among lists: "" means the newest, newList a new one.
func pickList(lists []todos.List, id string) (string, []todos.Item, bool) {
	if id == "" && len(lists) > 0 {
		id = lists[0].ID
	}
	if id == "" || id == newList {
		return newList, nil, true
	}
	i := slices.IndexFunc(lists, func(l todos.List) bool { return l.ID == id })
	if i < 0 {
		return id, nil, false
	}
	return id, slices.Clone(lists[i].Items), true
}

func (s *Server) shoppingLists(sub string) []todos.List {
	var out []todos.List
	for _, l := range s.Todos.Visible(sub, s.groups()) {
		if l.Kind == todos.KindShopping {
			out = append(out, l)
		}
	}
	return out
}

func (s *Server) shopRecipe(w http.ResponseWriter, r *http.Request) {
	if sv, ok := s.shopForm(w, r); ok {
		s.renderShop(w, r, sv)
	}
}

func (s *Server) renderShop(w http.ResponseWriter, r *http.Request, sv shopView) {
	v := s.newView(r, "", nil)
	v.Title, v.Data = v.T("shop.title", sv.Page.Meta.Title), sv
	s.render(w, r, http.StatusOK, "recipe_shop", v)
}

// addToList refreshes the preview, or applies the chosen lines to the chosen
// or a new list and opens it. The lines are planned again under the list's
// lock, against the list as it is then.
func (s *Server) addToList(w http.ResponseWriter, r *http.Request) {
	sv, ok := s.shopForm(w, r)
	if !ok {
		return
	}
	if r.PostFormValue("refresh") != "" {
		sv.Taken = append([]string{}, r.PostForm["take"]...)
		s.renderShop(w, r, sv)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	id, err := s.targetList(sv.Lists, sv.List, sv.Name, sv.Fill, u.Sub)
	if err == nil {
		take := r.PostForm["take"]
		src := todos.Source{Recipe: sv.Page.Meta.ID, Title: sv.Page.Meta.Title, Fill: sv.Fill}
		err = s.Todos.Change(u.Sub, s.groups(), id, func(l *todos.List) error {
			lines := shopping.Plan(l.Items, sv.ings, sv.lang)
			shopping.Apply(l, lines, func(i int) bool { return slices.Contains(take, strconv.Itoa(i)) }, src, time.Now())
			return nil
		})
	}
	switch {
	case errors.Is(err, todos.ErrNotFound):
		s.notFound(w, r)
	case err != nil:
		s.fail(w, r, http.StatusBadRequest, "shop.invalid")
	default:
		http.Redirect(w, r, "/todos/"+id, http.StatusSeeOther)
	}
}

// targetList is the list this fill already went to, the chosen list, or a new one.
func (s *Server) targetList(lists []todos.List, list, name, fill, sub string) (string, error) {
	for _, l := range lists {
		if shopping.Filled([]todos.List{l}, fill) {
			return l.ID, nil
		}
	}
	if list != newList {
		return list, nil
	}
	l, err := s.Todos.Create(sub, name, todos.KindShopping)
	return l.ID, err
}
