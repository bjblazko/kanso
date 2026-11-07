// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"encoding/json"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// Draft is a recipe not yet saved.
type Draft struct {
	Title, Description string
	Servings, Time     int // Time in minutes
	Source             string
	Ingredients, Steps []string
}

var (
	ldScript = regexp.MustCompile(`(?is)<script[^>]*type\s*=\s*["']?application/ld\+json["']?[^>]*>(.*?)</script>`)
	tags     = regexp.MustCompile(`(?s)<[^>]*>`)
	firstInt = regexp.MustCompile(`\d+`)
	duration = regexp.MustCompile(`^P(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:[\d.]+S)?)?$`)
)

// FromJSONLD reads the first schema.org Recipe of a page.
func FromJSONLD(page, source string) (Draft, bool) {
	for _, m := range ldScript.FindAllStringSubmatch(page, -1) {
		var v any
		if json.Unmarshal([]byte(m[1]), &v) != nil {
			continue
		}
		if r := findRecipe(v); r != nil {
			d := draftOf(r, source)
			return d, d.Title != "" && len(d.Ingredients) > 0
		}
	}
	return Draft{}, false
}

func findRecipe(v any) map[string]any {
	switch x := v.(type) {
	case map[string]any:
		if isType(x["@type"], "Recipe") {
			return x
		}
		return findRecipe(x["@graph"])
	case []any:
		for _, e := range x {
			if r := findRecipe(e); r != nil {
				return r
			}
		}
	}
	return nil
}

func isType(v any, want string) bool {
	switch x := v.(type) {
	case string:
		return x == want
	case []any:
		for _, e := range x {
			if isType(e, want) {
				return true
			}
		}
	}
	return false
}

func draftOf(r map[string]any, source string) Draft {
	d := Draft{Title: clean(r["name"]), Description: clean(r["description"]), Source: source,
		Servings: servings(r["recipeYield"]), Time: minutes(r["totalTime"])}
	if d.Time == 0 {
		d.Time = minutes(r["prepTime"]) + minutes(r["cookTime"])
	}
	for _, s := range asList(r["recipeIngredient"]) {
		if c := clean(s); c != "" {
			d.Ingredients = append(d.Ingredients, c)
		}
	}
	d.Steps = steps(r["recipeInstructions"])
	return d
}

// clean turns a JSON value into plain text without tags or entities.
func clean(v any) string {
	s, _ := v.(string)
	s = html.UnescapeString(tags.ReplaceAllString(html.UnescapeString(s), " "))
	return strings.Join(strings.Fields(s), " ")
}

func asList(v any) []any {
	if l, ok := v.([]any); ok {
		return l
	}
	if v == nil {
		return nil
	}
	return []any{v}
}

func servings(v any) int {
	for _, e := range asList(v) {
		switch x := e.(type) {
		case float64:
			return int(x)
		case string:
			if n, err := strconv.Atoi(firstInt.FindString(x)); err == nil && n > 0 {
				return n
			}
		}
	}
	return 0
}

func minutes(v any) int {
	s, _ := v.(string)
	m := duration.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return 0
	}
	n := func(i int) int { k, _ := strconv.Atoi(m[i]); return k }
	return n(1)*24*60 + n(2)*60 + n(3)
}

// steps reads plain text, a list of texts, HowToSteps and HowToSections.
func steps(v any) []string {
	if s, ok := v.(string); ok {
		var out []string
		for _, line := range strings.Split(html.UnescapeString(s), "\n") {
			if c := clean(line); c != "" {
				out = append(out, c)
			}
		}
		return out
	}
	var out []string
	for _, e := range asList(v) {
		if m, ok := e.(map[string]any); ok && m["itemListElement"] != nil {
			out = append(out, steps(m["itemListElement"])...)
		} else if ok {
			out = append(out, stepText(m))
		} else if c := clean(e); c != "" {
			out = append(out, c)
		}
	}
	return out
}

func stepText(m map[string]any) string {
	if t := clean(m["text"]); t != "" {
		return t
	}
	return clean(m["name"])
}
