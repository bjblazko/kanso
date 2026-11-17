// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"cmp"
	"slices"
	"sort"
	"time"
)

// hourlyUpTo is the longest window the summary shows in hours; longer ones in days.
const hourlyUpTo = 48 * time.Hour

// Change is one change to a document, for the start page summary.
type Change struct {
	Kind   Kind
	DocID  string
	When   time.Time
	Person string // a sub; "" when unknown (e.g. list items: nobody records who ticked)
	Ticked bool   // a list item ticked off
}

// KindSum is what happened to one kind of document in the window.
type KindSum struct {
	Kind                 Kind
	New, Changed, Ticked int   // documents created, other documents changed, list items ticked
	Spark                []int // changes per bucket
}

// PersonSum is how many changes one person made.
type PersonSum struct {
	Person string
	N      int
}

// Recent is a changed document with its latest change.
type Recent struct {
	Doc    Doc
	When   time.Time
	Person string
}

// Summary is what happened between From and To.
type Summary struct {
	From, To time.Time
	Hourly   bool        // buckets are hours, else days
	Buckets  []time.Time // bucket starts in To's location
	Kinds    []KindSum   // in Kinds order, only kinds with changes
	Band     map[Kind][]int
	People   []PersonSum // most first; unknown last
	Docs     []Recent    // newest first
	Total    int         // changes counted
}

// Summarize counts the changes to docs between from and to. Changes to
// documents not in docs, to attachments and, unless own, by viewer are left
// out; changes by nobody known always stay.
func Summarize(docs []Doc, changes []Change, from, to time.Time, viewer string, own bool) Summary {
	byID := map[string]Doc{}
	for _, d := range docs {
		if d.Kind != Attachment {
			byID[d.ID] = d
		}
	}
	s := Summary{From: from, To: to, Hourly: to.Sub(from) <= hourlyUpTo, Band: map[Kind][]int{}}
	s.Buckets = buckets(from, to, s.Hourly)
	sums, people, latest := map[Kind]*KindSum{}, map[string]int{}, map[string]Recent{}
	for _, c := range changes {
		d, ok := byID[c.DocID]
		if !ok || !counts(c, from, to, viewer, own) {
			continue
		}
		c.Kind = d.Kind // the document decides
		s.Total++
		people[c.Person]++
		s.count(sums, c)
		if r, seen := latest[d.ID]; !seen || c.When.After(r.When) {
			latest[d.ID] = Recent{Doc: d, When: c.When, Person: c.Person}
		}
	}
	s.finish(sums, people, latest, from)
	return s
}

// counts tells whether c falls into the window and is not the viewer's own, unless asked.
func counts(c Change, from, to time.Time, viewer string, own bool) bool {
	inWindow := !c.When.Before(from) && !c.When.After(to)
	return inWindow && (own || c.Person == "" || c.Person != viewer)
}

// count adds c to its kind's band and spark line.
func (s *Summary) count(sums map[Kind]*KindSum, c Change) {
	ks := sums[c.Kind]
	if ks == nil {
		ks = &KindSum{Kind: c.Kind, Spark: make([]int, len(s.Buckets))}
		sums[c.Kind] = ks
		s.Band[c.Kind] = ks.Spark
	}
	if c.Ticked {
		ks.Ticked++
	}
	ks.Spark[bucketOf(s.Buckets, c.When)]++
}

// finish sorts what count gathered and counts the documents per kind.
func (s *Summary) finish(sums map[Kind]*KindSum, people map[string]int, latest map[string]Recent, from time.Time) {
	for _, r := range latest {
		s.Docs = append(s.Docs, r)
		if r.Doc.Kind == List {
			continue // lists count their items
		}
		if r.Doc.Created.Before(from) {
			sums[r.Doc.Kind].Changed++
		} else {
			sums[r.Doc.Kind].New++
		}
	}
	slices.SortFunc(s.Docs, func(a, b Recent) int { return b.When.Compare(a.When) })
	for _, k := range Kinds {
		if ks := sums[k]; ks != nil {
			s.Kinds = append(s.Kinds, *ks)
		}
	}
	for p, n := range people {
		s.People = append(s.People, PersonSum{p, n})
	}
	slices.SortFunc(s.People, func(a, b PersonSum) int {
		if (a.Person == "") != (b.Person == "") {
			return cmp.Compare(b.Person, a.Person) // the unknown one last
		}
		return cmp.Or(cmp.Compare(b.N, a.N), cmp.Compare(a.Person, b.Person))
	})
}

// buckets are the starts of the hours or days from from through to, in to's location.
func buckets(from, to time.Time, hourly bool) []time.Time {
	f := from.In(to.Location())
	start := time.Date(f.Year(), f.Month(), f.Day(), f.Hour(), 0, 0, 0, f.Location())
	if !hourly {
		start = time.Date(f.Year(), f.Month(), f.Day(), 0, 0, 0, 0, f.Location())
	}
	var out []time.Time
	for t := start; !t.After(to); {
		out = append(out, t)
		if hourly {
			t = t.Add(time.Hour)
		} else {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
		}
	}
	return out
}

// bucketOf is the index of the last bucket starting at or before when.
func bucketOf(starts []time.Time, when time.Time) int {
	i := sort.Search(len(starts), func(i int) bool { return starts[i].After(when) })
	return max(0, i-1)
}
