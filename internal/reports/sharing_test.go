// SPDX-License-Identifier: Apache-2.0

package reports

import "testing"

func TestNetworkWeights(t *testing.T) {
	people, links := Network([][]string{{"a", "b"}, {"a", "b", "c"}, {"d"}})
	if len(people) != 3 || people[0] != "a" || people[2] != "c" {
		t.Errorf("people %v", people)
	}
	want := map[[2]string]int{{"a", "b"}: 2, {"a", "c"}: 1, {"b", "c"}: 1}
	if len(links) != len(want) {
		t.Fatalf("links %+v", links)
	}
	for _, l := range links {
		if want[[2]string{l.A, l.B}] != l.N {
			t.Errorf("link %+v", l)
		}
	}
}
