// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"testing"
	"time"
)

func TestAgeBins(t *testing.T) {
	now := day("2026-11-17")
	ago := func(d int) time.Time { return now.AddDate(0, 0, -d) }
	docs := []Doc{
		{Kind: Page, Updated: ago(3)}, {Kind: Recipe, Updated: ago(40)}, {Kind: Page, Updated: ago(100)},
		{Kind: Page, Updated: ago(200)}, {Kind: Page, Updated: ago(500)}, {Kind: Page, Updated: ago(1000)},
		{Kind: Note, Updated: ago(3)}, // not a wiki page
	}
	bins := Age(docs, now)
	want := []string{"m1", "m3", "m6", "m12", "y2", "older"}
	if len(bins) != len(want) {
		t.Fatalf("bins %+v", bins)
	}
	for i, b := range bins {
		if b.Key != want[i] || b.N != 1 {
			t.Errorf("bin %d = %+v", i, b)
		}
	}
}

func TestOldestPages(t *testing.T) {
	docs := []Doc{{Kind: Page, Title: "neu", Updated: day("2026-11-01")}, {Kind: Page, Title: "alt", Updated: day("2020-01-01")},
		{Kind: Attachment, Title: "x", Updated: day("2010-01-01")}}
	got := Oldest(docs, 1)
	if len(got) != 1 || got[0].Title != "alt" {
		t.Errorf("oldest = %+v", got)
	}
}
