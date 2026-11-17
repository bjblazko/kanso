// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"slices"
	"time"
)

// TaskInfo is what the task reports need of a task.
type TaskInfo struct {
	Created           time.Time
	Done              *time.Time // when it was done; nil while open
	Open              bool       // not done
	Urgent, Important bool
	Due               string // YYYY-MM-DD or ""
}

// Week is how many tasks were created and done in the week from Start (a Monday).
type Week struct {
	Start     time.Time
	New, Done int
}

// Flow counts new and done tasks for the last n weeks up to now's week.
func Flow(tasks []TaskInfo, n int, now time.Time) []Week {
	this := dayOf(now).AddDate(0, 0, -((int(now.Weekday()) + 6) % 7))
	weeks := make([]Week, n)
	for i := range weeks {
		weeks[i].Start = this.AddDate(0, 0, -7*(n-1-i))
	}
	index := func(t time.Time) int {
		d := daysBetween(weeks[0].Start, t.In(now.Location()))
		if d < 0 || d >= 7*n {
			return -1
		}
		return d / 7
	}
	for _, t := range tasks {
		if i := index(t.Created); i >= 0 {
			weeks[i].New++
		}
		if t.Done != nil {
			if i := index(*t.Done); i >= 0 {
				weeks[i].Done++
			}
		}
	}
	return weeks
}

// Quadrant is the Eisenhower field of tasks: how many are open, how many of
// those are overdue, and the median days from created to done over the last year.
type Quadrant struct {
	Open, Overdue int
	MedianDays    float64
}

// quadrantOf: 0 urgent and important, 1 important, 2 urgent, 3 neither.
func quadrantOf(t TaskInfo) int {
	switch {
	case t.Urgent && t.Important:
		return 0
	case t.Important:
		return 1
	case t.Urgent:
		return 2
	}
	return 3
}

// Quadrants sums the tasks up per Eisenhower field.
func Quadrants(tasks []TaskInfo, now time.Time) [4]Quadrant {
	var q [4]Quadrant
	var days [4][]float64
	today := now.Format("2006-01-02")
	for _, t := range tasks {
		i := quadrantOf(t)
		switch {
		case t.Open:
			q[i].Open++
			if t.Due != "" && t.Due < today {
				q[i].Overdue++
			}
		case t.Done != nil && t.Done.After(now.AddDate(-1, 0, 0)):
			days[i] = append(days[i], t.Done.Sub(t.Created).Hours()/24)
		}
	}
	for i := range q {
		q[i].MedianDays = median(days[i])
	}
	return q
}

func median(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	slices.Sort(xs)
	n := len(xs)
	if n%2 == 1 {
		return xs[n/2]
	}
	return (xs[n/2-1] + xs[n/2]) / 2
}
