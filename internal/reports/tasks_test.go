// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"testing"
	"time"
)

func TestFlowPerWeek(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local) // Wednesday
	done := now.AddDate(0, 0, -1)
	tasks := []TaskInfo{
		{Created: now.AddDate(0, 0, -1), Done: &done},
		{Created: now.AddDate(0, 0, -8), Open: true},
		{Created: now.AddDate(-1, 0, 0), Open: true}, // too old
	}
	weeks := Flow(tasks, 4, now)
	if len(weeks) != 4 {
		t.Fatalf("weeks %+v", weeks)
	}
	last, prev := weeks[3], weeks[2]
	if last.New != 1 || last.Done != 1 || prev.New != 1 || prev.Done != 0 {
		t.Errorf("last %+v prev %+v", last, prev)
	}
	if last.Start.Weekday() != time.Monday {
		t.Errorf("week starts %v", last.Start.Weekday())
	}
}

func TestQuadrantsMedianAndOverdue(t *testing.T) {
	now := time.Date(2026, 11, 18, 12, 0, 0, 0, time.Local)
	at := func(d int) *time.Time { x := now.AddDate(0, 0, -d); return &x }
	tasks := []TaskInfo{
		{Urgent: true, Important: true, Open: true, Due: "2026-11-01"},
		{Urgent: true, Important: true, Open: true, Due: "2026-12-01"},
		{Urgent: true, Important: true, Created: now.AddDate(0, 0, -10), Done: at(8)},  // 2 days
		{Urgent: true, Important: true, Created: now.AddDate(0, 0, -10), Done: at(4)},  // 6 days
		{Urgent: true, Important: true, Created: now.AddDate(0, 0, -20), Done: at(16)}, // 4 days
		{Important: true, Open: true},
		{Open: true},
	}
	q := Quadrants(tasks, now)
	if q[0].Open != 2 || q[0].Overdue != 1 || q[0].MedianDays != 4 {
		t.Errorf("urgent+important %+v", q[0])
	}
	if q[1].Open != 1 || q[3].Open != 1 || q[2].Open != 0 {
		t.Errorf("quadrants %+v", q)
	}
}
