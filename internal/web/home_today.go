// SPDX-License-Identifier: Apache-2.0

package web

import (
	"log/slog"
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/mealplan"
	"github.com/bjblazko/kanso/internal/tasks"
)

// homeToday is what is on today: meals planned and tasks due.
type homeToday struct {
	Meals, Due []homeItem
}

// today collects today's meals of the plans sub may enter and the open tasks
// due today or overdue on the boards sub sees, overdue first.
func (s *Server) today(v view, sub string, g access.Groups, now time.Time) homeToday {
	return homeToday{Meals: s.todaysMeals(sub, g, now), Due: s.dueTasks(v, sub, g, now)}
}

func (s *Server) todaysMeals(sub string, g access.Groups, now time.Time) []homeItem {
	var out []homeItem
	for _, p := range s.MealPlans.For(g, sub) {
		w, err := s.MealPlans.Week(p.ID, mealplan.WeekKey(now))
		if err != nil {
			slog.Warn("home: reading the meal plan", "plan", p.ID, "err", err)
			continue
		}
		for _, e := range w.Days[now.Format("2006-01-02")] {
			out = append(out, homeItem{Title: e.Title, Href: "/mealplan/" + p.ID, Person: p.Name})
		}
	}
	return out
}

func (s *Server) dueTasks(v view, sub string, g access.Groups, now time.Time) []homeItem {
	viewer := tasks.Actor{Sub: sub, Groups: g}
	boards := map[string]string{}
	for _, b := range s.Tasks.AllBoards() {
		if viewer.Sees(b) {
			boards[b.ID] = boardName(v, b)
		}
	}
	var overdue, due []homeItem
	for _, t := range tasks.Sorted(s.Tasks.AllTasks()) {
		name, ok := boards[t.Board]
		if !ok || t.Meta.Status == tasks.Done {
			continue
		}
		item := homeItem{Title: t.Meta.Title, Href: "/tasks/b/" + t.Board + "/" + t.Meta.ID, Person: name}
		switch tasks.DueOf(t.Meta.Due, now) {
		case tasks.DueOverdue:
			item.When = v.T("home.overdue")
			overdue = append(overdue, item)
		case tasks.DueToday:
			item.When = v.T("home.due_today")
			due = append(due, item)
		}
	}
	return slices.Concat(overdue, due)
}
