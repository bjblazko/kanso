// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"strings"
	"testing"
	"time"
)

func day(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestInventoryByKind(t *testing.T) {
	docs := []Doc{{Kind: Page}, {Kind: Page}, {Kind: Note}, {Kind: Attachment}}
	bars := Inventory(docs, false, strings.ToUpper, "Andere")
	if len(bars) != 1 || bars[0].Values[Page] != 2 || bars[0].Values[Note] != 1 || bars[0].Values[Attachment] != 1 {
		t.Errorf("bars = %+v", bars)
	}
}

func TestInventoryByPersonFoldsOthers(t *testing.T) {
	var docs []Doc
	for i, p := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		for range 7 - i { // a has most
			docs = append(docs, Doc{Kind: Note, Person: p})
		}
	}
	bars := Inventory(docs, true, strings.ToUpper, "Andere")
	if len(bars) != 6 || bars[0].Label != "A" || bars[4].Label != "E" || bars[5].Label != "Andere" {
		t.Fatalf("labels = %+v", bars)
	}
	if bars[5].Values[Note] != 2+1 {
		t.Errorf("others = %d", bars[5].Values[Note])
	}
}

func TestGrowthIsCumulative(t *testing.T) {
	docs := []Doc{
		{Kind: Page, Created: day("2025-12-20")}, // before the range: counted from the start
		{Kind: Page, Created: day("2026-01-05")},
		{Kind: Note, Created: day("2026-03-01")},
	}
	pts := Growth(docs, day("2026-01-15"), day("2026-03-10"))
	if len(pts) != 3 {
		t.Fatalf("points = %+v", pts)
	}
	if pts[0].Values[Page] != 2 || pts[1].Values[Page] != 2 || pts[2].Values[Note] != 1 || pts[1].Values[Note] != 0 {
		t.Errorf("points = %+v", pts)
	}
	if !pts[0].Month.Equal(day("2026-01-01")) {
		t.Errorf("first month %v", pts[0].Month)
	}
}

func TestTotals(t *testing.T) {
	now := day("2026-11-17")
	docs := []Doc{
		{Kind: Page, Created: day("2026-11-01"), Bytes: 10},
		{Kind: Attachment, Created: day("2025-01-01"), Bytes: 100},
	}
	tt := Totals(docs, now)
	if tt.Docs != 2 || tt.Docs30 != 1 || tt.Bytes != 110 || tt.AttachmentBytes != 100 {
		t.Errorf("totals = %+v", tt)
	}
}
