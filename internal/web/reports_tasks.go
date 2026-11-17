// SPDX-License-Identifier: Apache-2.0

package web

import (
	"fmt"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/tasks"
)

// flowWeeks is how many weeks "Neu gegen erledigt" shows.
const flowWeeks = 16

// reportQuadrants name the Eisenhower fields in reports.Quadrants' order.
var reportQuadrants = []string{"both", "important", "urgent", "neither"}

// tasksTab: how many tasks come and go, and where the open ones sit.
func (s *Server) tasksTab(rb *reportBuild) {
	infos := s.taskInfos(rb)
	flowCard(rb, infos)
	quadrantsCard(rb, infos)
}

// taskInfos are the scope's tasks with what the task reports need.
func (s *Server) taskInfos(rb *reportBuild) []reports.TaskInfo {
	in := map[string]bool{}
	for _, d := range rb.docs {
		if d.Kind == reports.Task {
			in[d.ID] = true
		}
	}
	var out []reports.TaskInfo
	for _, t := range s.Tasks.AllTasks() {
		if !in[t.Meta.ID] {
			continue
		}
		out = append(out, reports.TaskInfo{Created: t.Meta.Created, Done: t.Meta.DoneAt, Open: t.Meta.Status != tasks.Done,
			Urgent: t.Meta.Urgent, Important: t.Meta.Important, Due: t.Meta.Due})
	}
	return out
}

type weekData struct {
	Label string `json:"label"`
	New   int    `json:"new"`
	Done  int    `json:"done"`
}

func flowCard(rb *reportBuild, infos []reports.TaskInfo) {
	v := rb.rc.v
	weeks := reports.Flow(infos, flowWeeks, time.Now())
	card := reportCard{ID: "flow", Chart: "flow", Title: v.T("reports.flow"), Why: v.T("reports.flow_why"), Empty: len(infos) == 0,
		Head: []string{v.T("reports.week"), v.T("reports.new"), v.T("reports.done")}}
	data := struct {
		Labels map[string]string `json:"labels"`
		Weeks  []weekData        `json:"weeks"`
	}{Labels: map[string]string{"new": v.T("reports.new"), "done": v.T("reports.done")}}
	for _, w := range weeks {
		_, kw := w.Start.ISOWeek()
		label := v.T("reports.week_n", kw)
		data.Weeks = append(data.Weeks, weekData{label, w.New, w.Done})
		card.Rows = append(card.Rows, []reportCell{text(label + " · " + w.Start.Format("02.01.")), num(w.New), num(w.Done)})
	}
	rb.card(card, data)
}

type quadrantData struct {
	Name    string  `json:"name"`
	Open    int     `json:"open"`
	Overdue int     `json:"overdue"`
	Median  float64 `json:"median"`
}

func quadrantsCard(rb *reportBuild, infos []reports.TaskInfo) {
	v := rb.rc.v
	q := reports.Quadrants(infos, time.Now())
	card := reportCard{ID: "quadrants", Chart: "quadrants", Title: v.T("reports.quadrants"), Why: v.T("reports.quadrants_why"),
		Empty: len(infos) == 0, Head: []string{v.T("reports.field"), v.T("reports.open"), v.T("reports.overdue"), v.T("reports.median")}}
	data := struct {
		Fields []quadrantData    `json:"fields"`
		Labels map[string]string `json:"labels"`
	}{Labels: map[string]string{"open": v.T("reports.open"), "overdue": v.T("reports.overdue_lc"), "median": v.T("reports.median_days")}}
	for i, f := range q {
		name := v.T("reports.q_" + reportQuadrants[i])
		data.Fields = append(data.Fields, quadrantData{name, f.Open, f.Overdue, f.MedianDays})
		card.Rows = append(card.Rows, []reportCell{text(name), num(f.Open), num(f.Overdue), {Text: fmt.Sprintf("%.1f", f.MedianDays), Num: true}})
	}
	rb.card(card, data)
}
