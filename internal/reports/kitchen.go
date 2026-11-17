// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"cmp"
	"slices"
	"strings"
)

// Count is how often something occurs.
type Count struct {
	Key string
	N   int
}

// Top are the n keys counted most often, ties by key.
func Top(counts map[string]int, n int) []Count {
	out := make([]Count, 0, len(counts))
	for k, c := range counts {
		out = append(out, Count{k, c})
	}
	slices.SortFunc(out, func(a, b Count) int { return cmp.Or(b.N-a.N, cmp.Compare(a.Key, b.Key)) })
	return out[:min(n, len(out))]
}

// Normalise makes shopping items that differ only in case and spaces one.
func Normalise(item string) string { return strings.ToLower(strings.Join(strings.Fields(item), " ")) }

// timeBins are the upper bounds in minutes of the preparation-time bins; the last is open.
var timeBins = []struct {
	key     string
	minutes int
}{{"t15", 15}, {"t30", 30}, {"t60", 60}, {"t120", 120}, {"more", 0}}

// Times counts recipes by preparation time; 0 means unknown and is left out.
func Times(minutes []int) []Bin {
	bins := make([]Bin, len(timeBins))
	for i, b := range timeBins {
		bins[i].Key = b.key
	}
	for _, m := range minutes {
		if m <= 0 {
			continue
		}
		i := len(timeBins) - 1
		for j, b := range timeBins[:i] {
			if m <= b.minutes {
				i = j
				break
			}
		}
		bins[i].N++
	}
	return bins
}
