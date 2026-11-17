// SPDX-License-Identifier: Apache-2.0

package web

import (
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/mealplan"
	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/todos"
)

// Lengths of the kitchen rankings.
const (
	topCooked   = 10
	topShopping = 15
	topTags     = 15
)

// kitchenTab: what gets cooked and bought, and what the recipes are like.
func (s *Server) kitchenTab(rb *reportBuild) {
	s.cookedCard(rb)
	s.shoppingCard(rb)
	s.tagsCard(rb)
	s.timesCard(rb)
}

// rankingData is a ranking for the "ranking" chart.
type rankingData struct {
	Unit string        `json:"unit"`
	Rows []rankingItem `json:"rows"`
}

type rankingItem struct {
	Label string `json:"label"`
	N     int    `json:"n"`
}

// rankingCard writes a ranking as chart and table; label and href name a key.
func rankingCard(rb *reportBuild, card reportCard, counts []reports.Count, label func(string) (string, string)) {
	v := rb.rc.v
	data := rankingData{Unit: card.Head[1]}
	for _, c := range counts {
		name, href := label(c.Key)
		data.Rows = append(data.Rows, rankingItem{name, c.N})
		card.Rows = append(card.Rows, []reportCell{docCell(v, name, href), num(c.N)})
	}
	card.Chart, card.Empty = "ranking", len(counts) == 0
	rb.card(card, data)
}

func (s *Server) cookedCard(rb *reportBuild) {
	v := rb.rc.v
	counts, titles := s.cookedCounts(rb)
	card := reportCard{ID: "cooked", Title: v.T("reports.cooked"), Why: v.T("reports.cooked_why"),
		Head: []string{v.T("reports.one_recipe"), v.T("reports.times_cooked")}}
	rankingCard(rb, card, reports.Top(counts, topCooked), func(key string) (string, string) {
		if title, free := titles[key]; free {
			return title, ""
		}
		pg, ok := s.Wiki.ByID(key)
		if !ok || !access.Can(s.pageRole(rb.rc.p, pg.Space, key, rb.rc.g), access.Read) {
			return "", ""
		}
		return pg.Meta.Title, "/p/" + key
	})
}

// cookedCounts counts the last year's meals per recipe id, or per title for
// meals without a recipe ("t:<title>", titles in the second map). Admins in
// everyone's scope count plans they are not in without their meals' titles.
func (s *Server) cookedCounts(rb *reportBuild) (map[string]int, map[string]string) {
	mine := map[string]bool{}
	plans := s.MealPlans.For(rb.rc.g, rb.rc.p.Sub)
	for _, p := range plans {
		mine[p.ID] = true
	}
	if rb.rc.scope == scopeAll {
		plans = s.MealPlans.All()
	}
	since := time.Now().AddDate(-1, 0, 0).Format("2006-01-02")
	counts, titles := map[string]int{}, map[string]string{}
	for _, p := range plans {
		for _, e := range s.MealPlans.Entries([]mealplan.Plan{p}, since) {
			key := e.Recipe
			if key == "" {
				key = "t:" + e.Title
				if !mine[p.ID] {
					key = "t:"
				}
				titles[key] = key[2:]
			}
			counts[key]++
		}
	}
	return counts, titles
}

func (s *Server) shoppingCard(rb *reportBuild) {
	v := rb.rc.v
	in := map[string]bool{}
	for _, d := range rb.docs {
		if d.Kind == reports.List {
			in[d.ID] = d.Title != "" // only lists the viewer may read name their items
		}
	}
	counts := map[string]int{}
	for _, l := range s.Todos.All() {
		readable, ok := in[l.ID]
		if !ok || l.Kind != todos.KindShopping {
			continue
		}
		for _, it := range l.Items {
			if it.Done {
				key := reports.Normalise(it.Text)
				if !readable {
					key = ""
				}
				counts[key]++
			}
		}
	}
	card := reportCard{ID: "shopping", Title: v.T("reports.shopping"), Why: v.T("reports.shopping_why"),
		Head: []string{v.T("reports.item"), v.T("reports.times_ticked")}}
	rankingCard(rb, card, reports.Top(counts, topShopping), func(key string) (string, string) { return key, "" })
}

func (s *Server) tagsCard(rb *reportBuild) {
	v := rb.rc.v
	counts := map[string]int{}
	for _, d := range rb.docs {
		if d.Kind != reports.Recipe || d.Title == "" {
			continue
		}
		if pg, ok := s.Wiki.ByID(d.ID); ok {
			for _, t := range pg.Meta.Tags {
				counts[t]++
			}
		}
	}
	card := reportCard{ID: "tags", Title: v.T("reports.tags"), Why: v.T("reports.tags_why"),
		Head: []string{v.T("reports.tag"), v.T("reports.kind_recipe")}}
	rankingCard(rb, card, reports.Top(counts, topTags), func(key string) (string, string) { return key, "" })
}

func (s *Server) timesCard(rb *reportBuild) {
	v := rb.rc.v
	var minutes []int
	for _, d := range rb.docs {
		if d.Kind != reports.Recipe {
			continue
		}
		if pg, ok := s.Wiki.ByID(d.ID); ok {
			minutes = append(minutes, pg.Meta.Time)
		}
	}
	bins := reports.Times(minutes)
	card := reportCard{ID: "times", Chart: "age", Title: v.T("reports.times"), Why: v.T("reports.times_why"), Empty: true,
		Head: []string{v.T("reports.prep_time"), v.T("reports.kind_recipe")}}
	data := struct {
		Rows  []chartRow `json:"rows"`
		Stale int        `json:"stale"`
		Unit  string     `json:"unit"`
	}{Stale: len(bins), Unit: v.T("reports.kind_recipe")}
	for _, b := range bins {
		label := v.T("reports.time_" + b.Key)
		data.Rows = append(data.Rows, chartRow{Label: label, Values: []int{b.N}})
		card.Rows = append(card.Rows, []reportCell{text(label), num(b.N)})
		card.Empty = card.Empty && b.N == 0
	}
	rb.card(card, data)
}
