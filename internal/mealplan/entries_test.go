// SPDX-License-Identifier: Apache-2.0

package mealplan

import "testing"

func TestEntriesSince(t *testing.T) {
	s := newStore(t)
	p, _ := s.Create(admin, "Küche", "familie")
	for _, d := range []struct{ week, day string }{{"2025-W02", "2025-01-07"}, {"2026-W46", "2026-11-10"}, {"2026-W46", "2026-11-12"}} {
		_ = s.Update(admin, p.ID, d.week, func(w *Week) error { w.Add(d.day, Entry{Recipe: "r1", Title: "Curry", Servings: 2}); return nil })
	}
	got := s.Entries([]Plan{p}, "2026-01-01")
	if len(got) != 2 || got[0].Recipe != "r1" || got[0].Title != "Curry" {
		t.Errorf("entries = %+v", got)
	}
}
