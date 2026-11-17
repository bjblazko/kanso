// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"cmp"
	"slices"
	"time"
)

// maxPeople is how many people a chart names before folding the rest into one.
const maxPeople = 5

// Bar is one bar of a stacked chart: a count per kind.
type Bar struct {
	Label  string
	Values map[Kind]int
}

// Inventory counts docs per kind: one bar, or with byPerson one bar per person
// (name gives the label), the five with most and the rest as others.
func Inventory(docs []Doc, byPerson bool, name func(person string) string, others string) []Bar {
	if !byPerson {
		b := Bar{Values: map[Kind]int{}}
		for _, d := range docs {
			b.Values[d.Kind]++
		}
		return []Bar{b}
	}
	per := map[string]map[Kind]int{}
	total := map[string]int{}
	for _, d := range docs {
		if per[d.Person] == nil {
			per[d.Person] = map[Kind]int{}
		}
		per[d.Person][d.Kind]++
		total[d.Person]++
	}
	people := sortedKeys(total)
	var bars []Bar
	for i, p := range people {
		if i < maxPeople {
			bars = append(bars, Bar{Label: name(p), Values: per[p]})
			continue
		}
		if i == maxPeople {
			bars = append(bars, Bar{Label: others, Values: map[Kind]int{}})
		}
		for k, n := range per[p] {
			bars[maxPeople].Values[k] += n
		}
	}
	return bars
}

// sortedKeys are the keys by count, most first, then by key.
func sortedKeys(total map[string]int) []string {
	keys := make([]string, 0, len(total))
	for k := range total {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, func(a, b string) int { return cmp.Or(total[b]-total[a], cmp.Compare(a, b)) })
	return keys
}

// Point is the number of docs per kind that existed at the end of Month.
type Point struct {
	Month  time.Time
	Values map[Kind]int
}

// Growth counts, for each month from from to to, the docs created until its end.
func Growth(docs []Doc, from, to time.Time) []Point {
	var pts []Point
	for m := monthOf(from); !m.After(to); m = m.AddDate(0, 1, 0) {
		end := m.AddDate(0, 1, 0)
		p := Point{Month: m, Values: map[Kind]int{}}
		for _, d := range docs {
			if d.Created.Before(end) {
				p.Values[d.Kind]++
			}
		}
		pts = append(pts, p)
	}
	return pts
}

func monthOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// Total is the headline of the overview.
type Total struct {
	Docs, Docs30           int // all, and created in the last 30 days
	Bytes, AttachmentBytes int64
}

// Totals sums docs up at now.
func Totals(docs []Doc, now time.Time) Total {
	var t Total
	for _, d := range docs {
		t.Docs++
		if d.Created.After(now.AddDate(0, 0, -30)) {
			t.Docs30++
		}
		t.Bytes += d.Bytes
		if d.Kind == Attachment {
			t.AttachmentBytes += d.Bytes
		}
	}
	return t
}
