// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"testing"
	"time"
)

var (
	cet  = time.FixedZone("CET", 3600)
	to   = time.Date(2026, 11, 17, 12, 0, 0, 0, cet)
	from = to.Add(-24 * time.Hour)
)

func ago(h float64) time.Time { return to.Add(-time.Duration(h * float64(time.Hour))) }

func kindSum(s Summary, k Kind) KindSum {
	for _, ks := range s.Kinds {
		if ks.Kind == k {
			return ks
		}
	}
	return KindSum{}
}

func TestSummarizeCountsNewChangedTicked(t *testing.T) {
	docs := []Doc{
		{Kind: Page, ID: "p", Created: ago(5)},
		{Kind: Recipe, ID: "r", Created: ago(100)},
		{Kind: List, ID: "l", Created: ago(100)},
	}
	changes := []Change{
		{Kind: Page, DocID: "p", When: ago(5), Person: "bo"},
		{Kind: Page, DocID: "p", When: ago(4), Person: "bo"},
		{Kind: Recipe, DocID: "r", When: ago(3), Person: "bo"},
		{Kind: List, DocID: "l", When: ago(2), Ticked: true},
		{Kind: List, DocID: "l", When: ago(1), Ticked: true},
	}
	s := Summarize(docs, changes, from, to, "anna", false)
	if p := kindSum(s, Page); p.New != 1 || p.Changed != 0 {
		t.Errorf("page: %+v", p)
	}
	if r := kindSum(s, Recipe); r.New != 0 || r.Changed != 1 {
		t.Errorf("recipe: %+v", r)
	}
	if l := kindSum(s, List); l.Ticked != 2 {
		t.Errorf("list: %+v", l)
	}
	if s.Total != 5 || len(s.Kinds) != 3 || s.Kinds[0].Kind != Page {
		t.Errorf("total %d, kinds %+v", s.Total, s.Kinds)
	}
}

func TestSummarizeHidesOwnUnlessAsked(t *testing.T) {
	docs := []Doc{{Kind: Page, ID: "p", Created: ago(100)}}
	changes := []Change{{Kind: Page, DocID: "p", When: ago(1), Person: "anna"}}
	if s := Summarize(docs, changes, from, to, "anna", false); s.Total != 0 || len(s.Docs) != 0 {
		t.Errorf("own shown: %+v", s)
	}
	if s := Summarize(docs, changes, from, to, "anna", true); s.Total != 1 || len(s.Docs) != 1 {
		t.Errorf("own hidden when asked: %+v", s)
	}
}

func TestSummarizeUnknownPersonStaysWithOwnHidden(t *testing.T) {
	docs := []Doc{{Kind: Page, ID: "p", Created: ago(100)}}
	changes := []Change{
		{Kind: Page, DocID: "p", When: ago(2), Person: "bo"},
		{Kind: Page, DocID: "p", When: ago(1), Person: ""},
	}
	s := Summarize(docs, changes, from, to, "anna", false)
	if s.Total != 2 || len(s.People) != 2 || s.People[1] != (PersonSum{"", 1}) {
		t.Errorf("people: %+v", s.People)
	}
}

func TestSummarizeSkipsOutsideWindowAndUnknownDocs(t *testing.T) {
	docs := []Doc{{Kind: Page, ID: "p", Created: ago(100)}, {Kind: Attachment, ID: "a"}}
	changes := []Change{
		{Kind: Page, DocID: "p", When: from.Add(-time.Second), Person: "bo"},
		{Kind: Page, DocID: "gone", When: ago(1), Person: "bo"},
		{Kind: Attachment, DocID: "a", When: ago(1), Person: "bo"},
	}
	if s := Summarize(docs, changes, from, to, "anna", false); s.Total != 0 || len(s.Kinds) != 0 || len(s.Docs) != 0 {
		t.Errorf("kept: %+v", s)
	}
}

func TestSummarizeRecentNewestFirst(t *testing.T) {
	docs := []Doc{{Kind: Page, ID: "a", Created: ago(100)}, {Kind: Note, ID: "b", Created: ago(100)}}
	changes := []Change{
		{Kind: Page, DocID: "a", When: ago(6), Person: "bo"},
		{Kind: Note, DocID: "b", When: ago(5), Person: "bo"},
		{Kind: Page, DocID: "a", When: ago(2), Person: "cy"},
	}
	s := Summarize(docs, changes, from, to, "anna", false)
	if len(s.Docs) != 2 || s.Docs[0].Doc.ID != "a" || s.Docs[0].Person != "cy" || !s.Docs[0].When.Equal(ago(2)) {
		t.Errorf("docs: %+v", s.Docs)
	}
}

func TestBucketsHoursUpToTwoDays(t *testing.T) {
	docs := []Doc{{Kind: Page, ID: "p", Created: ago(100)}}
	two := Summarize(docs, nil, to.Add(-48*time.Hour), to, "anna", false)
	if !two.Hourly || len(two.Buckets) != 49 {
		t.Errorf("48 h: hourly %v, %d buckets", two.Hourly, len(two.Buckets))
	}
	late := time.Date(2026, 11, 16, 23, 30, 0, 0, cet)
	more := Summarize(docs, []Change{{Kind: Page, DocID: "p", When: late, Person: "bo"}}, to.Add(-49*time.Hour), to, "anna", false)
	if more.Hourly || len(more.Buckets) != 3 {
		t.Fatalf("49 h: hourly %v, %d buckets", more.Hourly, len(more.Buckets))
	}
	if b := more.Buckets[1]; b.Hour() != 0 || b.Day() != 16 {
		t.Errorf("day bucket starts %v", b)
	}
	if band := more.Band[Page]; band[1] != 1 || kindSum(more, Page).Spark[1] != 1 {
		t.Errorf("23:30 not in its day: %v", band)
	}
}
