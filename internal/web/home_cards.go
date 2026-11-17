// SPDX-License-Identifier: Apache-2.0

package web

import (
	"fmt"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
)

// bandData is the "stacked-area" shape with hours or days instead of months.
type bandData struct {
	Keys []string  `json:"keys"`
	Unit string    `json:"unit"` // "hour" or "day"
	Rows []bandRow `json:"rows"`
}

type bandRow struct {
	Label  string `json:"label"`
	Values []int  `json:"values"`
}

// homeTiles are one number per kind, each with its spark line in data.
func homeTiles(v view, sum reports.Summary, data map[string]any) []homeTile {
	var out []homeTile
	for _, k := range sum.Kinds {
		id := "spark-" + string(k.Kind)
		data[id] = map[string][]int{"values": k.Spark}
		out = append(out, homeTile{Kind: string(k.Kind), Label: kindLabel(v, k.Kind), Value: fmt.Sprint(sparkSum(k.Spark)), Spark: id})
	}
	return out
}

// bandCard is the changes over the window, stacked by kind.
func bandCard(v view, sum reports.Summary, data map[string]any) reportCard {
	unit, layout, show := "day", "2006-01-02", "02.01."
	if sum.Hourly {
		unit, layout, show = "hour", "2006-01-02T15:04", "02.01. 15:04"
	}
	card := reportCard{ID: "band", Chart: "stacked-area", Title: v.T("home.band"), Why: v.T("home.band_why"),
		Head: []string{v.T("home.when")}}
	d := bandData{Unit: unit}
	for _, k := range sum.Kinds {
		d.Keys = append(d.Keys, kindLabel(v, k.Kind))
		card.Head = append(card.Head, kindLabel(v, k.Kind))
	}
	for i, b := range sum.Buckets {
		row := bandRow{Label: b.Format(layout)}
		cells := []reportCell{text(b.Format(show))}
		for _, k := range sum.Kinds {
			row.Values = append(row.Values, k.Spark[i])
			cells = append(cells, num(k.Spark[i]))
		}
		d.Rows = append(d.Rows, row)
		card.Rows = append(card.Rows, cells)
	}
	data[card.ID] = d
	return card
}

// peopleCard is who changed how much; changes nobody records count as "ohne Namen".
func (s *Server) peopleCard(v view, sum reports.Summary, data map[string]any) reportCard {
	card := reportCard{ID: "people", Chart: "ranking", Title: v.T("home.people"), Why: v.T("home.people_why"),
		Head: []string{v.T("home.person"), v.T("reports.changes")}}
	d := rankingData{Unit: v.T("reports.changes")}
	for _, p := range sum.People {
		name := s.homePerson(v, p.Person)
		d.Rows = append(d.Rows, rankingItem{name, p.N})
		card.Rows = append(card.Rows, []reportCell{text(name), num(p.N)})
	}
	data[card.ID] = d
	return card
}

// homePerson names who made a change.
func (s *Server) homePerson(v view, sub string) string {
	if sub == "" {
		return v.T("home.nobody")
	}
	return s.userName(sub)
}

// homeGroups are the changed documents by space, newest first, at most
// homeMaxItems; the second value is how many more there are.
func (s *Server) homeGroups(v view, docs []reports.Recent, now time.Time) ([]homeGroup, int) {
	var groups []homeGroup
	at := map[string]int{}
	for i, r := range docs {
		if i == homeMaxItems {
			return groups, len(docs) - homeMaxItems
		}
		name := s.groupName(v, r.Doc)
		g, ok := at[name]
		if !ok {
			g = len(groups)
			at[name] = g
			groups = append(groups, homeGroup{Name: name})
		}
		item := homeItem{Title: r.Doc.Title, Href: r.Doc.Href, Person: s.homePerson(v, r.Person), When: homeWhen(v, r.When, now)}
		groups[g].Items = append(groups[g].Items, item)
	}
	return groups, 0
}

// groupName is the space's name, or the kind for lists, notes and tasks.
func (s *Server) groupName(v view, d reports.Doc) string {
	switch d.Space {
	case "lists", "notes", "boards":
		return kindLabel(v, d.Kind)
	}
	if sp, ok := s.Spaces.Get(d.Space); ok {
		return sp.Name
	}
	return d.Space
}
