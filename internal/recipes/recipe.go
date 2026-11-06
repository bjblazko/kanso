// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"regexp"
	"strings"
)

// Group is a run of ingredients under an optional "###" heading.
type Group struct {
	Name        string // "" = without a heading
	Ingredients []Ingredient
}

// Recipe is what Kanso understands of a recipe page's text.
type Recipe struct {
	Groups []Group
	Steps  []string
}

type section int

const (
	sectionNone section = iota
	sectionIngredients
	sectionSteps
)

var (
	ingredientHeadings = map[string]bool{"zutaten": true, "ingredients": true}
	stepHeadings       = map[string]bool{"zubereitung": true, "schritte": true, "steps": true, "method": true}
	numberedRE         = regexp.MustCompile(`^\d+[.)]\s+`)
	headingRE          = regexp.MustCompile(`^(#{1,6})(?:\s+(.*))?$`)
)

// parser walks a page line by line.
type parser struct {
	r       Recipe
	sec     section
	fence   bool
	lastNew bool // the previous line belongs to a step, so a following plain line continues it
}

// Parse reads the ingredients and steps of a recipe page's Markdown text.
func Parse(body string) Recipe {
	var p parser
	for _, line := range strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n") {
		p.line(line)
	}
	return p.r
}

func (p *parser) line(line string) {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
		p.fence = !p.fence
		return
	}
	switch {
	case p.fence:
	case p.heading(trimmed):
	case trimmed == "":
		p.lastNew = false
	case strings.HasPrefix(trimmed, "#"): // a hashtag line, not a heading
	case trimmed == "---" || trimmed == "***":
		p.sec = sectionNone
	case p.sec == sectionIngredients:
		p.ingredient(trimmed)
	case p.sec == sectionSteps:
		p.step(trimmed)
	}
}

// heading handles a Markdown heading ("#" to "######" and a space) and
// reports whether the line was one.
func (p *parser) heading(s string) bool {
	m := headingRE.FindStringSubmatch(s)
	if m == nil {
		return false
	}
	level, text := len(m[1]), strings.TrimSpace(m[2])
	word := strings.ToLower(strings.Trim(strings.SplitN(text+" ", " ", 2)[0], ":()."))
	switch {
	case level == 3 && p.sec == sectionIngredients:
		p.r.Groups = append(p.r.Groups, Group{Name: text})
	case level <= 2 && ingredientHeadings[word]:
		p.sec = sectionIngredients
	case level <= 2 && stepHeadings[word]:
		p.sec = sectionSteps
	case level <= 2:
		p.sec = sectionNone
	}
	p.lastNew = false
	return true
}

// ingredient takes a list item, or a plain line, as one ingredient.
func (p *parser) ingredient(s string) {
	if len(p.r.Groups) == 0 {
		p.r.Groups = append(p.r.Groups, Group{})
	}
	g := &p.r.Groups[len(p.r.Groups)-1]
	g.Ingredients = append(g.Ingredients, ParseIngredient(s))
}

// step starts a step with a list item or a new paragraph; other lines continue the step.
func (p *parser) step(s string) {
	switch {
	case isListItem(s):
		p.r.Steps = append(p.r.Steps, strings.TrimSpace(numberedRE.ReplaceAllString(stripMarker(s), "")))
	case p.lastNew && len(p.r.Steps) > 0:
		p.r.Steps[len(p.r.Steps)-1] += " " + s
	default:
		p.r.Steps = append(p.r.Steps, s)
	}
	p.lastNew = true
}

func isListItem(s string) bool {
	return strings.HasPrefix(s, "- ") || strings.HasPrefix(s, "* ") || strings.HasPrefix(s, "+ ") || numberedRE.MatchString(s)
}

// Ingredients lists every ingredient of all groups in order.
func (r Recipe) Ingredients() []Ingredient {
	var all []Ingredient
	for _, g := range r.Groups {
		all = append(all, g.Ingredients...)
	}
	return all
}
