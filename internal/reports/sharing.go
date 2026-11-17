// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"cmp"
	"slices"
)

// Link is how many things two people both see; A sorts before B.
type Link struct {
	A, B string
	N    int
}

// Network turns things (each the people who see it) into people and the
// links between them; a thing only one person sees links nobody.
func Network(things [][]string) ([]string, []Link) {
	weights := map[[2]string]int{}
	seen := map[string]bool{}
	for _, people := range things {
		ps := slices.Clone(people)
		slices.Sort(ps)
		ps = slices.Compact(ps)
		for i := range ps {
			for _, b := range ps[i+1:] {
				weights[[2]string{ps[i], b}]++
				seen[ps[i]], seen[b] = true, true
			}
		}
	}
	people := make([]string, 0, len(seen))
	for p := range seen {
		people = append(people, p)
	}
	slices.Sort(people)
	links := make([]Link, 0, len(weights))
	for k, n := range weights {
		links = append(links, Link{k[0], k[1], n})
	}
	slices.SortFunc(links, func(x, y Link) int { return cmp.Or(y.N-x.N, cmp.Compare(x.A, y.A), cmp.Compare(x.B, y.B)) })
	return people, links
}
