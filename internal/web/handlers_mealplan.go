// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/mealplan"
	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/shopping"
	"github.com/bjblazko/kanso/internal/todos"
)

const dayLayout = "2006-01-02"

// planEntry is a meal with whether the person may open its recipe.
type planEntry struct {
	mealplan.Entry
	Readable bool
}

type planDay struct {
	Weekday int // 0 = Monday
	Date    time.Time
	Key     string
	Entries []planEntry
	Today   bool
}

type weekPage struct {
	Plan       mealplan.Plan
	Plans      []mealplan.Plan
	Week       string
	Prev, Next string
	Start, End time.Time
	Days       []planDay
}

// planFor loads {plan} for a member of its group; others get 404.
func (s *Server) planFor(w http.ResponseWriter, r *http.Request, id string) (mealplan.Plan, bool) {
	u, _ := auth.UserFrom(r.Context())
	p, err := s.MealPlans.Get(s.groups(), u.Sub, id)
	if err != nil {
		s.notFound(w, r)
		return mealplan.Plan{}, false
	}
	return p, true
}

// mealplanHome opens the person's first plan, or says there is none.
func (s *Server) mealplanHome(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if plans := s.MealPlans.For(s.groups(), u.Sub); len(plans) > 0 {
		http.Redirect(w, r, "/mealplan/"+plans[0].ID, http.StatusSeeOther)
		return
	}
	v := s.newView(r, "", weekPage{})
	v.Title = v.T("plan.title")
	s.render(w, r, http.StatusOK, "mealplan", v)
}

func (s *Server) mealplanWeek(w http.ResponseWriter, r *http.Request) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if !ok {
		return
	}
	start, key := mealplan.ParseWeek(r.URL.Query().Get("week"), time.Now())
	week, err := s.MealPlans.Week(p.ID, key)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	u, who := auth.UserFrom(r.Context())
	wp := weekPage{Plan: p, Plans: s.MealPlans.For(s.groups(), u.Sub), Week: key,
		Prev: mealplan.Shift(key, -1), Next: mealplan.Shift(key, 1), Start: start, End: start.AddDate(0, 0, 6)}
	today := time.Now().Format(dayLayout)
	for _, d := range mealplan.Days(start) {
		day := planDay{Weekday: (int(d.Weekday()) + 6) % 7, Date: d, Key: d.Format(dayLayout), Today: d.Format(dayLayout) == today}
		for _, e := range week.Days[day.Key] {
			pe := planEntry{Entry: e, Readable: s.recipeReadable(who, e.Recipe)}
			if page, ok := s.Wiki.ByID(e.Recipe); pe.Readable && ok {
				pe.Title = page.Meta.Title // renamed since it was planned
			}
			day.Entries = append(day.Entries, pe)
		}
		wp.Days = append(wp.Days, day)
	}
	v := s.newView(r, p.Name, wp)
	s.render(w, r, http.StatusOK, "mealplan", v)
}

// recipeReadable tells whether who may read the recipe page id.
func (s *Server) recipeReadable(who access.Principal, id string) bool {
	p, ok := s.Wiki.ByID(id)
	return ok && p.Meta.IsRecipe() && access.Can(s.pageRole(who, p.Space, id, s.groups()), access.Read)
}

type addMealPage struct {
	Plan     mealplan.Plan
	Plans    []mealplan.Plan
	Day      string
	Recipes  []recipeRow
	Recipe   *recipeRow // set when planning from a recipe page
	Servings int        // the recipe's servings, as the default
}

func (s *Server) addMealForm(w http.ResponseWriter, r *http.Request) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if !ok {
		return
	}
	_, who := auth.UserFrom(r.Context())
	day := r.URL.Query().Get("day")
	v := s.newView(r, "", addMealPage{Plan: p, Day: day, Recipes: s.visibleRecipes(who)})
	v.Title = v.T("plan.add")
	s.render(w, r, http.StatusOK, "mealplan_add", v)
}

func (s *Server) addMeal(w http.ResponseWriter, r *http.Request) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if ok {
		s.planMeal(w, r, p, r.PostFormValue("recipe"))
	}
}

// planMeal puts a readable recipe on a day of plan p and opens that week.
func (s *Server) planMeal(w http.ResponseWriter, r *http.Request, p mealplan.Plan, recipeID string) {
	u, who := auth.UserFrom(r.Context())
	page, ok := s.Wiki.ByID(recipeID)
	day, err := time.Parse(dayLayout, r.PostFormValue("day"))
	if !ok || !s.recipeReadable(who, recipeID) || err != nil {
		s.fail(w, r, http.StatusBadRequest, "plan.invalid")
		return
	}
	servings := max(page.Meta.Servings, 1)
	if raw := strings.TrimSpace(r.PostFormValue("servings")); raw != "" {
		servings = recipes.ClampServings(raw, servings)
	}
	key := mealplan.WeekKey(day)
	e := mealplan.Entry{Recipe: recipeID, Title: page.Meta.Title, Servings: servings}
	err = s.MealPlans.Update(u.Author(), p.ID, key, func(wk *mealplan.Week) error {
		if !wk.Add(day.Format(dayLayout), e) {
			return mealplan.ErrInvalid
		}
		return nil
	})
	if err != nil {
		s.fail(w, r, http.StatusBadRequest, "plan.invalid")
		return
	}
	http.Redirect(w, r, "/mealplan/"+p.ID+"?week="+key+"#day-"+day.Format(dayLayout), http.StatusSeeOther)
}

// planRecipeForm offers to put a recipe on a day of one of the person's plans.
func (s *Server) planRecipeForm(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok || !pc.Page.Meta.IsRecipe() {
		if ok {
			s.notFound(w, r)
		}
		return
	}
	u, _ := auth.UserFrom(r.Context())
	row := recipeRow{ID: pc.Page.Meta.ID, Title: pc.Page.Meta.Title}
	page := addMealPage{Plans: s.MealPlans.For(s.groups(), u.Sub), Recipe: &row, Servings: pc.Page.Meta.Servings,
		Day: time.Now().AddDate(0, 0, 1).Format(dayLayout)}
	v := s.newView(r, "", page)
	v.Title = v.T("plan.plan_recipe", pc.Page.Meta.Title)
	s.render(w, r, http.StatusOK, "mealplan_add", v)
}

func (s *Server) planRecipe(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.pageFor(w, r, access.Read); !ok {
		return
	}
	p, ok := s.planFor(w, r, r.PostFormValue("plan"))
	if ok {
		s.planMeal(w, r, p, r.PathValue("id"))
	}
}

type removeMealPage struct {
	Plan       mealplan.Plan
	Week       string
	Entry      mealplan.Entry
	ListName   string // the shopping list its ingredients went to; "" = none or gone
	ListHidden bool   // they went to a list the person cannot open
}

// removeMealForm asks before a meal leaves the plan.
func (s *Server) removeMealForm(w http.ResponseWriter, r *http.Request) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if !ok {
		return
	}
	week := r.URL.Query().Get("week")
	wk, err := s.MealPlans.Week(p.ID, week)
	e, _, found := wk.Find(r.URL.Query().Get("entry"))
	if err != nil || !found {
		s.notFound(w, r)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	page := removeMealPage{Plan: p, Week: week, Entry: e}
	if e.Fill != "" {
		l, err := s.Todos.Get(u.Sub, e.List, s.groups())
		page.ListName, page.ListHidden = l.Name, errors.Is(err, todos.ErrNotFound) && s.listExists(e.List)
	}
	v := s.newView(r, "", page)
	v.Title = v.T("plan.remove_title", e.Title)
	s.render(w, r, http.StatusOK, "mealplan_remove", v)
}

func (s *Server) removeMeal(w http.ResponseWriter, r *http.Request) {
	p, ok := s.planFor(w, r, r.PathValue("plan"))
	if !ok {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	_, week := mealplan.ParseWeek(r.PostFormValue("week"), time.Now())
	back := "/mealplan/" + p.ID + "?week=" + week
	wk, err := s.MealPlans.Week(p.ID, week)
	e, _, found := wk.Find(r.PostFormValue("entry"))
	if err != nil || !found { // gone already, e.g. a second tap
		http.Redirect(w, r, back, http.StatusSeeOther)
		return
	}
	if r.PostFormValue("subtract") != "" {
		if err := s.subtractMeal(u.Sub, s.newView(r, "", nil).Lang, e); err != nil {
			s.fail(w, r, http.StatusConflict, "plan.subtract_failed")
			return
		}
	}
	err = s.MealPlans.Update(u.Author(), p.ID, week, func(wk *mealplan.Week) error {
		wk.Remove(e.ID)
		return nil
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// listExists tells whether a list exists at all, whoever may see it.
func (s *Server) listExists(id string) bool { return s.Todos.Exists(id) }

// Dates of a week page, for the template.
func (wp weekPage) Range() string {
	return wp.Start.Format("2.1.") + "–" + wp.End.Format("2.1.2006")
}

// WeekNumber is the ISO week number of the page.
func (wp weekPage) WeekNumber() string {
	_, n := wp.Start.ISOWeek()
	return strconv.Itoa(n)
}

// subtractMeal takes what a removed meal put on its shopping list off again;
// a list that is gone has nothing left to take off.
func (s *Server) subtractMeal(sub, lang string, e mealplan.Entry) error {
	if e.Fill == "" || e.List == "" || !s.Todos.Exists(e.List) {
		return nil
	}
	return s.Todos.Change(sub, s.groups(), e.List, func(l *todos.List) error {
		shopping.Remove(l, e.Fill, lang, time.Now())
		return nil
	})
}
