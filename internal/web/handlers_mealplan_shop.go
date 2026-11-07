// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/mealplan"
	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/shopping"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/todos"
)

// mealSection is one planned meal in the shopping preview.
type mealSection struct {
	Entry   mealplan.Entry
	Weekday int
	Lines   []shopping.Line
	ings    []recipes.Ingredient
}

type shoppedMeal struct {
	Entry    mealplan.Entry
	ListName string
}

// planShopView adds the meals of chosen days to a shopping list (ADR-011, ADR-012).
type planShopView struct {
	Plan    mealplan.Plan
	Week    string
	Days    []string
	Lists   []todos.List
	List    string
	Name    string
	Fill    string
	Taken   []string
	Changed bool // the week changed since the preview was shown
	Meals   []mealSection
	Shopped []shoppedMeal
	Skipped []mealplan.Entry
}

// Takes tells the template whether line i of a meal is ticked.
func (v planShopView) Takes(meal string, i int) bool {
	return v.Taken == nil || slices.Contains(v.Taken, takeName(meal, i))
}

// takeName names line i of a meal in the form; by entry id, so a meal
// added or removed by someone else meanwhile cannot shift the ticks.
func takeName(meal string, i int) string { return fmt.Sprintf("%s-%d", meal, i) }

// mealIDs are the entries of the meals to shop, in order.
func (v planShopView) mealIDs() []string {
	var ids []string
	for _, m := range v.Meals {
		ids = append(ids, m.Entry.ID)
	}
	return ids
}

func fillOf(base string, e mealplan.Entry) string { return base + "-" + e.ID }

// planShopForm reads the chosen days and list and plans every meal in turn.
func (s *Server) planShopForm(w http.ResponseWriter, r *http.Request) (planShopView, bool) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if !ok {
		return planShopView{}, false
	}
	_, key := mealplan.ParseWeek(r.FormValue("week"), time.Now())
	wk, err := s.MealPlans.Week(p.ID, key)
	if err != nil {
		s.serverError(w, r, err)
		return planShopView{}, false
	}
	u, who := auth.UserFrom(r.Context())
	v := planShopView{Plan: p, Week: key, Days: r.Form["day"], Lists: s.shoppingLists(u.Sub),
		Name: strings.TrimSpace(r.FormValue("name")), Fill: r.PostFormValue("fill")}
	if v.Fill == "" {
		v.Fill = store.NewID()
	}
	slices.Sort(v.Days)
	v.Days = slices.Compact(v.Days)
	var items []todos.Item
	if v.List, items, ok = pickList(v.Lists, r.FormValue("list")); !ok {
		s.notFound(w, r)
		return planShopView{}, false
	}
	lang := s.newView(r, "", nil).Lang
	s.collectMeals(&v, wk, who, lang)
	v.preview(items, lang)
	return v, true
}

// collectMeals sorts the meals of the chosen days into to shop, shopped and unreadable.
func (s *Server) collectMeals(v *planShopView, wk mealplan.Week, who access.Principal, lang string) {
	for _, day := range v.Days {
		d, err := time.Parse(dayLayout, day)
		if err != nil || mealplan.WeekKey(d) != v.Week {
			continue
		}
		for _, e := range wk.Days[day] {
			switch {
			case e.Fill != "" && shopping.Filled(v.Lists, e.Fill):
				v.Shopped = append(v.Shopped, shoppedMeal{Entry: e, ListName: listName(v.Lists, e.List)})
			case !s.recipeReadable(who, e.Recipe):
				v.Skipped = append(v.Skipped, e)
			default:
				v.Meals = append(v.Meals, mealSection{Entry: e, Weekday: (int(d.Weekday()) + 6) % 7, ings: s.mealIngredients(e, lang)})
			}
		}
	}
}

func (s *Server) mealIngredients(e mealplan.Entry, lang string) []recipes.Ingredient {
	page, _ := s.Wiki.ByID(e.Recipe)
	factor := float64(e.Servings) / float64(max(page.Meta.Servings, 1))
	var out []recipes.Ingredient
	for _, ing := range recipes.Parse(page.Body).Ingredients() {
		out = append(out, ing.Scale(factor, lang))
	}
	return out
}

func listName(lists []todos.List, id string) string {
	if i := slices.IndexFunc(lists, func(l todos.List) bool { return l.ID == id }); i >= 0 {
		return lists[i].Name
	}
	return ""
}

// preview plans each meal against the list as the meals before it left it.
func (v *planShopView) preview(items []todos.Item, lang string) {
	work := &todos.List{Items: items}
	for k := range v.Meals {
		v.Meals[k].Lines = shopping.Plan(work.Items, v.Meals[k].ings, lang)
		shopping.Apply(work, v.Meals[k].Lines, func(int) bool { return true }, todos.Source{}, time.Now())
	}
}

func (s *Server) planShop(w http.ResponseWriter, r *http.Request) {
	if v, ok := s.planShopForm(w, r); ok {
		s.renderPlanShop(w, r, v)
	}
}

func (s *Server) renderPlanShop(w http.ResponseWriter, r *http.Request, v planShopView) {
	vw := s.newView(r, "", nil)
	if v.Name == "" {
		v.Name = vw.T("shop.default_name")
	}
	vw.Title, vw.Data = vw.T("plan.shop_title"), v
	s.render(w, r, http.StatusOK, "mealplan_shop", vw)
}

// addPlanToList refreshes the preview, or adds the chosen lines of every meal
// to the list, each meal as its own fill, and marks the meals as shopped.
func (s *Server) addPlanToList(w http.ResponseWriter, r *http.Request) {
	v, ok := s.planShopForm(w, r)
	if !ok {
		return
	}
	v.Changed = !slices.Equal(r.PostForm["meal"], v.mealIDs())
	if r.PostFormValue("refresh") != "" || len(v.Meals) == 0 || v.Changed {
		v.Taken = append([]string{}, r.PostForm["take"]...)
		s.renderPlanShop(w, r, v)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	id, err := s.targetList(v.Lists, v.List, v.Name, fillOf(v.Fill, v.Meals[0].Entry), u.Sub)
	var landed []string
	if err == nil {
		err = s.Todos.Change(u.Sub, s.groups(), id, func(l *todos.List) error {
			landed = v.apply(l, r.PostForm["take"], s.newView(r, "", nil).Lang)
			return nil
		})
	}
	if err == nil {
		err = s.markShopped(u.Author(), v, id, landed)
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

// apply adds each meal's ticked lines and returns the meals now on the list.
func (v planShopView) apply(l *todos.List, take []string, lang string) []string {
	var landed []string
	for _, m := range v.Meals {
		lines := shopping.Plan(l.Items, m.ings, lang)
		fill := fillOf(v.Fill, m.Entry)
		src := todos.Source{Recipe: m.Entry.Recipe, Title: m.Entry.Title, Fill: fill}
		shopping.Apply(l, lines, func(i int) bool { return slices.Contains(take, takeName(m.Entry.ID, i)) }, src, time.Now())
		if shopping.Filled([]todos.List{*l}, fill) {
			landed = append(landed, m.Entry.ID)
		}
	}
	return landed
}

// markShopped records in the plan which fill and list the meals that
// landed on the list went to; older marks of a list now gone are replaced.
func (s *Server) markShopped(a store.Author, v planShopView, listID string, landed []string) error {
	return s.MealPlans.Update(a, v.Plan.ID, v.Week, func(wk *mealplan.Week) error {
		for _, m := range v.Meals {
			if e, _, ok := wk.Find(m.Entry.ID); ok && slices.Contains(landed, e.ID) {
				e.Fill, e.List = fillOf(v.Fill, m.Entry), listID
				wk.Set(e)
			}
		}
		return nil
	})
}
