// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/search"
)

// filters are what the kitchen and the search narrow their lists by.
type filters struct {
	Query       string
	Tags        []string // all of them
	OnlyRecipes bool
	Stars       int // at least this many; 0 = any
}

// starChoices are the minimum ratings offered as filters.
var starChoices = []int{3, 4, maxStars}

// starLink switches a minimum rating on or off.
type starLink struct {
	N       int
	Href    string
	Pressed bool
}

func filtersFrom(r *http.Request) filters {
	q := r.URL.Query()
	f := filters{Query: strings.TrimSpace(q.Get("q")), OnlyRecipes: q.Get("type") == search.KindRecipe,
		Tags: recipes.NormalizeTags(strings.Join(q["tag"], ","))}
	if n, err := strconv.Atoi(q.Get("stars")); err == nil && n >= 1 && n <= maxStars {
		f.Stars = n
	}
	return f
}

// accepts tells whether something with these tags and stars passes the filters.
func (f filters) accepts(tags []string, rating int) bool {
	return hasAll(tags, f.Tags) && rating >= f.Stars
}

func (f filters) values() url.Values {
	v := url.Values{}
	if f.Query != "" {
		v.Set("q", f.Query)
	}
	for _, t := range f.Tags {
		v.Add("tag", t)
	}
	if f.Stars > 0 {
		v.Set("stars", strconv.Itoa(f.Stars))
	}
	return v
}

func (f filters) kitchenURL() string {
	if v := f.values(); len(v) > 0 {
		return "/recipes?" + v.Encode()
	}
	return "/recipes"
}

func (f filters) searchURL() string {
	v := f.values()
	if f.OnlyRecipes {
		v.Set("type", search.KindRecipe)
	}
	return "/search?" + v.Encode()
}

// withTags and withStars are the same filters with other tags or stars.
func (f filters) withTags(t []string) filters { f.Tags = t; return f }
func (f filters) withStars(n int) filters     { f.Stars = n; return f }

// starLinks offers each minimum rating; the chosen one switches off again.
func starLinks(f filters, href func(filters) string) []starLink {
	links := make([]starLink, 0, len(starChoices))
	for _, n := range starChoices {
		on := f.Stars == n
		next := n
		if on {
			next = 0
		}
		links = append(links, starLink{N: n, Pressed: on, Href: href(f.withStars(next))})
	}
	return links
}
