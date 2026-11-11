// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http/httptest"
	"testing"
)

func TestFiltersRoundTrip(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?q=+curry+&tag=Scharf&tag=schnell&type=recipe&stars=4", nil)
	f := filtersFrom(r)
	if f.Query != "curry" || len(f.Tags) != 2 || f.Tags[0] != "scharf" || !f.OnlyRecipes || f.Stars != 4 {
		t.Fatalf("%+v", f)
	}
	if got := f.searchURL(); got != "/search?q=curry&stars=4&tag=scharf&tag=schnell&type=recipe" {
		t.Errorf("search %s", got)
	}
	if got := f.kitchenURL(); got != "/recipes?q=curry&stars=4&tag=scharf&tag=schnell" {
		t.Errorf("kitchen %s", got)
	}
	if got := (filters{}).kitchenURL(); got != "/recipes" {
		t.Errorf("empty kitchen %s", got)
	}
}

func TestFiltersStars(t *testing.T) {
	for _, raw := range []string{"0", "6", "-1", "x", ""} {
		if f := filtersFrom(httptest.NewRequest("GET", "/recipes?stars="+raw, nil)); f.Stars != 0 {
			t.Errorf("%q gave %d", raw, f.Stars)
		}
	}
	f := filters{Tags: []string{"scharf"}, Stars: 4}
	if !f.accepts([]string{"scharf", "rot"}, 5) || f.accepts([]string{"scharf"}, 3) || f.accepts(nil, 5) {
		t.Error("accepts")
	}
	links := starLinks(f, filters.kitchenURL)
	if len(links) != 3 || links[0].N != 3 || links[0].Href != "/recipes?stars=3&tag=scharf" {
		t.Fatalf("%+v", links)
	}
	if !links[1].Pressed || links[1].Href != "/recipes?tag=scharf" { // pressing the chosen one clears it
		t.Errorf("%+v", links[1])
	}
}
