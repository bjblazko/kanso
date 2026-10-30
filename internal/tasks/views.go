// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"cmp"
	"slices"
	"strings"
	"time"
)

// Quadrant is a field of the Eisenhower matrix.
type Quadrant int

// The quadrants: urgent and important, important, urgent, neither.
const (
	DoNow Quadrant = iota
	Plan
	Quick
	Later
)

// Quadrants lists the matrix fields in reading order.
var Quadrants = []Quadrant{DoNow, Plan, Quick, Later}

// QuadrantOf places a task in the matrix.
func QuadrantOf(t Task) Quadrant {
	switch {
	case t.Meta.Urgent && t.Meta.Important:
		return DoNow
	case t.Meta.Important:
		return Plan
	case t.Meta.Urgent:
		return Quick
	}
	return Later
}

// Sorted orders tasks by due date (none last), then newest first.
func Sorted(ts []Task) []Task {
	out := slices.Clone(ts)
	slices.SortStableFunc(out, func(x, y Task) int {
		dx, dy := x.Meta.Due, y.Meta.Due
		switch {
		case dx != dy && dx == "":
			return 1
		case dx != dy && dy == "":
			return -1
		case dx != dy:
			return cmp.Compare(dx, dy)
		}
		return y.Meta.Created.Compare(x.Meta.Created)
	})
	return out
}

// doneShown is how long a finished task stays in the Done column.
const doneShown = 14 * 24 * time.Hour

// Kanban groups tasks by column; Done holds only the last 14 days unless allDone.
func Kanban(ts []Task, now time.Time, allDone bool) map[Status][]Task {
	out := map[Status][]Task{}
	for _, t := range Sorted(ts) {
		st := t.Meta.Status
		if !st.valid() {
			st = Open
		}
		if st == Done && !allDone && (t.Meta.DoneAt == nil || now.Sub(*t.Meta.DoneAt) > doneShown) {
			continue
		}
		out[st] = append(out[st], t)
	}
	return out
}

// Matrix groups the tasks that are not done by quadrant.
func Matrix(ts []Task) map[Quadrant][]Task {
	out := map[Quadrant][]Task{}
	for _, t := range Sorted(ts) {
		if t.Meta.Status != Done {
			out[QuadrantOf(t)] = append(out[QuadrantOf(t)], t)
		}
	}
	return out
}

// DueState says how close a due date is.
type DueState int

// From no date to past due.
const (
	DueNone DueState = iota
	DueLater
	DueTomorrow
	DueToday
	DueOverdue
)

// DueOf compares a due date with the day of now (in now's location).
func DueOf(due string, now time.Time) DueState {
	d, err := time.ParseInLocation(dateLayout, due, now.Location())
	if err != nil {
		return DueNone
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	switch {
	case d.Before(today):
		return DueOverdue
	case d.Equal(today):
		return DueToday
	case d.Equal(today.AddDate(0, 0, 1)):
		return DueTomorrow
	}
	return DueLater
}

// FromNote splits a note into a task subject and text. The subject is the
// first line of text as notes name it (no heading or list mark, at most 200
// characters); the text is what follows. The note's photo links are left out:
// the photos stay with the note, which the task links to.
func FromNote(body string) (title, rest string) {
	var kept []string
	for line := range strings.SplitSeq(body, "\n") {
		switch {
		case isPhoto(line):
		case title == "":
			title = subjectOf(line)
		default:
			kept = append(kept, line)
		}
	}
	if rest = strings.Trim(strings.Join(kept, "\n"), "\n"); rest != "" {
		rest += "\n"
	}
	return title, rest
}

func isPhoto(line string) bool {
	line = strings.TrimSpace(line)
	return strings.HasPrefix(line, "![") && strings.Contains(line, "](attachments/")
}

// subjectOf is a line without heading and list marks, cut to 200 characters.
func subjectOf(line string) string {
	line = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(line), "#"))
	line = strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
	if r := []rune(line); len(r) > maxTitle {
		line = string(r[:maxTitle])
	}
	return line
}
