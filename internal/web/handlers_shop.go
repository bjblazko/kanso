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
	var ings []recipes.Ingredient
	lang := s.newView(r, "", nil).Lang
	for _, ing := range recipes.Parse(pc.Page.Body).Ingredients() {
		ings = append(ings, ing.Scale(float64(sv.Servings)/float64(base), lang))
	}
	sv.Lines = shopping.Plan(items, ings, lang)
	return sv, true
}

// chosenItems picks the list (the newest one when none is chosen) and returns its items.
func (sv *shopView) chosenItems() ([]todos.Item, bool) {
	if sv.List == "" && len(sv.Lists) > 0 {
		sv.List = sv.Lists[0].ID
	}
	if sv.List == "" || sv.List == newList {
		sv.List = newList
		return nil, true
	}
	i := slices.IndexFunc(sv.Lists, func(l todos.List) bool { return l.ID == sv.List })
	if i < 0 {
		return nil, false
	}
	return sv.Lists[i].Items, true
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
	sv, ok := s.shopForm(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", nil)
	if sv.Name == "" {
		sv.Name = v.T("shop.default_name")
	}
	v.Title, v.Data = v.T("shop.title", sv.Page.Meta.Title), sv
	s.render(w, r, http.StatusOK, "recipe_shop", v)
}

// addToList applies the chosen lines to the chosen or a new list and opens it.
func (s *Server) addToList(w http.ResponseWriter, r *http.Request) {
	sv, ok := s.shopForm(w, r)
	if !ok {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	id, err := s.targetList(sv, u.Sub)
	if err == nil {
		take := r.PostForm["take"]
		src := todos.Source{Recipe: sv.Page.Meta.ID, Title: sv.Page.Meta.Title}
		err = s.Todos.Change(u.Sub, s.groups(), id, func(l *todos.List) error {
			shopping.Apply(l, sv.Lines, func(i int) bool { return slices.Contains(take, strconv.Itoa(i)) }, src, time.Now())
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

func (s *Server) targetList(sv shopView, sub string) (string, error) {
	if sv.List != newList {
		return sv.List, nil
	}
	l, err := s.Todos.Create(sub, sv.Name, todos.KindShopping)
	return l.ID, err
}
