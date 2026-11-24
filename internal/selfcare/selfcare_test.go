// SPDX-License-Identifier: Apache-2.0

package selfcare

import (
	"math/rand/v2"
	"testing"
)

func TestPickNeverRepeats(t *testing.T) {
	before := ""
	for range 500 {
		got := Pick(rand.IntN, before)
		if got.ID == before {
			t.Fatalf("picked %q twice in a row", before)
		}
		before = got.ID
	}
}

func TestPickReachesEveryTip(t *testing.T) {
	seen := map[string]bool{}
	for i := range len(tips) - 1 {
		seen[Pick(func(int) int { return i }, tips[len(tips)-1].ID).ID] = true
	}
	if len(seen) != len(tips)-1 {
		t.Fatalf("seen %d of %d", len(seen), len(tips)-1)
	}
}

func TestContentComplete(t *testing.T) {
	ids := map[string]bool{}
	for _, tp := range tips {
		if ids[tp.ID] || tp.Title.De == "" || tp.Title.En == "" || tp.Body.De == "" || tp.Body.En == "" {
			t.Errorf("tip %q incomplete or twice", tp.ID)
		}
		ids[tp.ID] = true
	}
	for _, e := range exercises {
		if len(e.Steps) == 0 || e.Lead.En == "" || (e.Kind != "steps" && e.Kind != "breath" && e.Kind != "note") {
			t.Errorf("exercise %q incomplete", e.ID)
		}
		for _, s := range e.Steps {
			if s.De == "" || s.En == "" {
				t.Errorf("exercise %q: step without both languages", e.ID)
			}
		}
	}
	for _, l := range Levels {
		if got := l.Suggestions(); len(got) != len(l.tips) {
			t.Errorf("level %s names an unknown tip", l.Key)
		}
	}
}

func TestLevels(t *testing.T) {
	l, ok := LevelByKey("10")
	if !ok || !l.Empty() {
		t.Fatal("empty battery must point to help first")
	}
	for _, tp := range l.Suggestions() {
		if !tp.Gentle {
			t.Errorf("empty battery suggests %q, which is not gentle", tp.ID)
		}
	}
	if _, ok := LevelByKey("x"); ok {
		t.Fatal("unknown level found")
	}
	if (Text{"Hallo", "Hello"}).In("en") != "Hello" || (Text{"Hallo", ""}).In("en") != "Hallo" {
		t.Fatal("In")
	}
}
