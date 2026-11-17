// SPDX-License-Identifier: Apache-2.0

package web

import (
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
)

// activityTab: when things change, over the year and over the week.
func (s *Server) activityTab(rb *reportBuild) {
	events := s.activityEvents(rb)
	calendarCard(rb, events)
	weekHoursCard(rb, events)
}

// activityEvents are the changes to the scope's documents: commits touching
// their files (every commit for everyone's scope), notes written, list items
// written and ticked.
func (s *Server) activityEvents(rb *reportBuild) []reports.Event {
	paths := map[string]string{}
	lists := map[string]bool{}
	var out []reports.Event
	for _, d := range rb.docs {
		paths[filepath.ToSlash(d.Path)] = d.Person
		switch d.Kind {
		case reports.Note:
			out = append(out, reports.Event{When: d.Created, Person: d.Person})
			if d.Updated.After(d.Created) {
				out = append(out, reports.Event{When: d.Updated, Person: d.Person})
			}
		case reports.List:
			lists[d.ID] = true
		}
	}
	for _, c := range rb.rc.scan.Changes {
		if person, ok := paths[c.Path]; ok || rb.rc.scope == scopeAll {
			out = append(out, reports.Event{When: c.When, Person: person})
		}
	}
	return append(out, s.listEvents(lists)...)
}

// listEvents are the items written and ticked on the lists named.
func (s *Server) listEvents(lists map[string]bool) []reports.Event {
	var out []reports.Event
	for _, l := range s.Todos.All() {
		if !lists[l.ID] {
			continue
		}
		for _, it := range l.Items {
			out = append(out, reports.Event{When: time.UnixMilli(it.At.Text), Person: l.Owner})
			if it.Done && it.At.Done > it.At.Text {
				out = append(out, reports.Event{When: time.UnixMilli(it.At.Done), Person: l.Owner})
			}
		}
	}
	return out
}

type dayData struct {
	Date string `json:"d"`
	N    int    `json:"n"`
}

func calendarCard(rb *reportBuild, events []reports.Event) {
	v := rb.rc.v
	days := reports.Calendar(events, time.Now())
	card := reportCard{ID: "calendar", Chart: "calendar", Title: v.T("reports.calendar"), Why: v.T("reports.calendar_why"),
		Head: []string{v.T("reports.date"), v.T("reports.changes")}, Empty: true}
	data := struct {
		Label string    `json:"label"`
		Days  []dayData `json:"days"`
	}{Label: v.T("reports.changes")}
	for _, d := range days {
		data.Days = append(data.Days, dayData{d.Date.Format("2006-01-02"), d.N})
		if d.N > 0 {
			card.Empty = false
			card.Rows = append(card.Rows, []reportCell{text(d.Date.Format("02.01.2006")), num(d.N)})
		}
	}
	slices.Reverse(card.Rows) // newest first
	rb.card(card, data)
}

func weekHoursCard(rb *reportBuild, events []reports.Event) {
	v := rb.rc.v
	wh := reports.WeekHours(events)
	card := reportCard{ID: "weekhours", Chart: "weekhours", Title: v.T("reports.weekhours"), Why: v.T("reports.weekhours_why"),
		Empty: len(events) == 0, Head: []string{v.T("reports.weekday")}}
	data := struct {
		Days   []string   `json:"days"`
		Counts [7][24]int `json:"counts"`
		Hour   string     `json:"hour"`
	}{Counts: wh, Hour: v.T("reports.hour")}
	for h := range 24 {
		card.Head = append(card.Head, fmt.Sprint(h))
	}
	for i, hours := range wh {
		day := v.T(fmt.Sprintf("reports.wd_%d", i))
		data.Days = append(data.Days, day)
		row := []reportCell{text(day)}
		for _, n := range hours {
			row = append(row, num(n))
		}
		card.Rows = append(card.Rows, row)
	}
	rb.card(card, data)
}
