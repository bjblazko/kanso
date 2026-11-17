// SPDX-License-Identifier: Apache-2.0

package reports

import "testing"

func TestTopSortsAndCuts(t *testing.T) {
	got := Top(map[string]int{"a": 1, "b": 3, "c": 3, "d": 2}, 3)
	if len(got) != 3 || got[0].Key != "b" || got[1].Key != "c" || got[2].Key != "d" || got[0].N != 3 {
		t.Errorf("top = %+v", got)
	}
}

func TestNormaliseShopping(t *testing.T) {
	for in, want := range map[string]string{"  Milch ": "milch", "Rote   Zwiebeln": "rote zwiebeln", "EIER": "eier"} {
		if got := Normalise(in); got != want {
			t.Errorf("Normalise(%q) = %q", in, got)
		}
	}
}

func TestTimesBins(t *testing.T) {
	bins := Times([]int{0, 10, 15, 25, 45, 90, 200})
	want := map[string]int{"t15": 2, "t30": 1, "t60": 1, "t120": 1, "more": 1}
	if len(bins) != 5 {
		t.Fatalf("bins %+v", bins)
	}
	for _, b := range bins {
		if b.N != want[b.Key] {
			t.Errorf("%s = %d, want %d", b.Key, b.N, want[b.Key])
		}
	}
}
