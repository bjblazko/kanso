// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"slices"
	"time"
)

// Bin is a count under a key the caller names.
type Bin struct {
	Key string
	N   int
}

// ageBins are the upper bounds in months of the page-age bins; the last is open.
var ageBins = []struct {
	key    string
	months int
}{{"m1", 1}, {"m3", 3}, {"m6", 6}, {"m12", 12}, {"y2", 24}, {"older", 0}}

func isWikiPage(d Doc) bool { return d.Kind == Page || d.Kind == Recipe }

// Age counts wiki pages and recipes by their last change.
func Age(docs []Doc, now time.Time) []Bin {
	bins := make([]Bin, len(ageBins))
	for i, b := range ageBins {
		bins[i].Key = b.key
	}
	for _, d := range docs {
		if !isWikiPage(d) {
			continue
		}
		i := len(ageBins) - 1
		for j, b := range ageBins[:i] {
			if d.Updated.After(now.AddDate(0, -b.months, 0)) {
				i = j
				break
			}
		}
		bins[i].N++
	}
	return bins
}

// Oldest are the n wiki pages changed longest ago.
func Oldest(docs []Doc, n int) []Doc {
	var pages []Doc
	for _, d := range docs {
		if isWikiPage(d) {
			pages = append(pages, d)
		}
	}
	slices.SortStableFunc(pages, func(a, b Doc) int { return a.Updated.Compare(b.Updated) })
	return pages[:min(n, len(pages))]
}
