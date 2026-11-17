// SPDX-License-Identifier: Apache-2.0

package todos

import "testing"

func TestAllListsOfEveryone(t *testing.T) {
	s := newStore(t)
	if _, err := s.Create("anna", "A", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create("ben", "B", ""); err != nil {
		t.Fatal(err)
	}
	if n := len(s.All()); n != 2 {
		t.Errorf("All = %d lists", n)
	}
}
