// SPDX-License-Identifier: Apache-2.0

// Package recipegen asks the AI for one recipe from ingredients and wishes.
// It guards the prompt (ADR-017): choices come from fixed lists, free text is
// cleaned, capped and sent as marked data, and only a recipe gets through.
package recipegen

import (
	"errors"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/bjblazko/kanso/internal/recipes"
)

// ErrInvalid means a wish is not one Kanso offers.
var ErrInvalid = errors.New("invalid wish")

// The choices a form offers; the keys are also the i18n suffixes gen.<field>_<key>.
var (
	Temps    = []string{"", "warm", "cold"}
	Heats    = []string{"", "mild", "hot"}
	Styles   = []string{"", "homestyle", "italian", "mediterranean", "asian", "indian", "oriental", "mexican", "french"}
	Times    = []string{"", "20", "45", "90"}
	Withouts = []string{"meat", "fish", "vegetables", "gluten", "lactose"}
)

// Limits of the free text.
const (
	maxIngredients  = 15
	maxIngredient   = 40
	maxExtra        = 200
	maxRejected     = 10
	maxTitle        = 80
	defaultServings = 4
)

// Wishes are what the person asks the dish to be.
type Wishes struct {
	Ingredients             []string
	Extend                  bool // may add more than basic pantry items
	Temp, Heat, Style, Time string
	Without                 []string
	Servings                int
	Extra                   string
	Rejected                []string // titles thrown away, newest last
}

// Parse reads and checks the wishes of a form; an unknown choice is ErrInvalid.
func Parse(form url.Values) (Wishes, error) {
	w := Wishes{Ingredients: ingredients(form.Get("ingredients")), Extend: form.Get("extend") == "1",
		Temp: form.Get("temp"), Heat: form.Get("heat"), Style: form.Get("style"), Time: form.Get("time"),
		Without: form["without"], Extra: clean(form.Get("extra"), maxExtra)}
	if len(w.Ingredients) == 0 || !chosen(w) {
		return Wishes{}, ErrInvalid
	}
	n, err := servings(form.Get("servings"))
	if err != nil {
		return Wishes{}, err
	}
	w.Servings = n
	for _, t := range form["rejected"] {
		w = w.Reject(t)
	}
	return w, nil
}

// chosen says whether every choice is one of the offered ones.
func chosen(w Wishes) bool {
	ok := slices.Contains(Temps, w.Temp) && slices.Contains(Heats, w.Heat) &&
		slices.Contains(Styles, w.Style) && slices.Contains(Times, w.Time)
	for _, x := range w.Without {
		ok = ok && slices.Contains(Withouts, x)
	}
	return ok
}

func servings(s string) (int, error) {
	if strings.TrimSpace(s) == "" {
		return defaultServings, nil
	}
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > recipes.MaxServings {
		return 0, ErrInvalid
	}
	return n, nil
}

// ingredients splits at commas and line breaks, cleans each and keeps the first ones.
func ingredients(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' || r == ';' }) {
		if c := clean(part, maxIngredient); c != "" && len(out) < maxIngredients {
			out = append(out, c)
		}
	}
	return out
}

// Values are the wishes as form fields; Parse reads them back.
func (w Wishes) Values() url.Values {
	v := url.Values{"ingredients": {strings.Join(w.Ingredients, ", ")}, "servings": {strconv.Itoa(w.Servings)}}
	if w.Extend {
		v.Set("extend", "1")
	}
	for k, x := range map[string]string{"temp": w.Temp, "heat": w.Heat, "style": w.Style, "time": w.Time, "extra": w.Extra} {
		if x != "" {
			v.Set(k, x)
		}
	}
	v["without"], v["rejected"] = w.Without, w.Rejected
	for _, k := range []string{"without", "rejected"} {
		if len(v[k]) == 0 {
			delete(v, k)
		}
	}
	return v
}

// Reject adds a thrown-away title and keeps the newest ones.
func (w Wishes) Reject(title string) Wishes {
	t := clean(title, maxTitle)
	if t == "" {
		return w
	}
	w.Rejected = append(slices.Clone(w.Rejected), t)
	if n := len(w.Rejected); n > maxRejected {
		w.Rejected = w.Rejected[n-maxRejected:]
	}
	return w
}

// clean drops control characters and the characters that could open or close
// a block or markup, folds white space and cuts to at most limit runes.
func clean(s string, limit int) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.IsControl(r):
			return ' '
		case strings.ContainsRune("<>`{}", r):
			return -1
		}
		return r
	}, s)
	r := []rune(strings.Join(strings.Fields(s), " "))
	if len(r) > limit {
		r = r[:limit]
	}
	return strings.TrimSpace(string(r))
}
