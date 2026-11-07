// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"context"
	_ "embed"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/ai"
	"github.com/bjblazko/kanso/internal/recipes"
)

// ErrNoRecipe means the AI found no recipe on the page.
var ErrNoRecipe = errors.New("no recipe found on the page")

// maxText is how much of a page's text goes to the AI.
const maxText = 30000

//go:embed prompt.md
var prompt string

var fence = regexp.MustCompile("(?s)^```[a-zA-Z]*\\n(.*?)\\n?```$")

// FromAI asks the AI to write the page's recipe in Kanso's format.
func FromAI(ctx context.Context, c ai.Client, cfg ai.Config, page Page) (Draft, error) {
	text := Text(page.HTML)
	if r := []rune(text); len(r) > maxText {
		text = string(r[:maxText])
	}
	answer, err := c.Complete(ctx, cfg, []ai.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "Seite: " + page.URL + "\n\n" + text},
	})
	if err != nil {
		return Draft{}, err
	}
	d := parseAnswer(answer)
	d.Source = page.URL
	if d.Title == "" || len(d.Ingredients) == 0 {
		return Draft{}, ErrNoRecipe
	}
	return d, nil
}

// parseAnswer reads "# Titel", "Portionen:", "Zeit:", a description and the
// recipe sections; anything else in front of the sections becomes the description.
func parseAnswer(answer string) Draft {
	answer = strings.TrimSpace(answer)
	if m := fence.FindStringSubmatch(answer); m != nil {
		answer = m[1]
	}
	var d Draft
	var body, desc []string
	for _, line := range strings.Split(answer, "\n") {
		t := strings.TrimSpace(line)
		switch {
		case d.Title == "" && strings.HasPrefix(t, "# "):
			d.Title = strings.TrimSpace(t[2:])
		case strings.HasPrefix(t, "Portionen:") || strings.HasPrefix(t, "Servings:"):
			d.Servings = number(t)
		case strings.HasPrefix(t, "Zeit:") || strings.HasPrefix(t, "Time:"):
			d.Time = number(t)
		case len(body) == 0 && !strings.HasPrefix(t, "## "):
			if t != "" {
				desc = append(desc, t)
			}
		default:
			body = append(body, line)
		}
	}
	d.Description = strings.Join(desc, " ")
	r := recipes.Parse(strings.Join(body, "\n"))
	for _, ing := range r.Ingredients() {
		d.Ingredients = append(d.Ingredients, ing.Line)
	}
	d.Steps = r.Steps
	return d
}

func number(line string) int {
	n, _ := strconv.Atoi(firstInt.FindString(line))
	return n
}
