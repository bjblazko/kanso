// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"context"
	_ "embed"
	"errors"
	"slices"
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

// languageNames name the person's language for the AI, in German like the prompt.
var languageNames = map[string]string{"de": "Deutsch", "en": "Englisch"}

// FromAI asks the AI to write the page's recipe in Kanso's format; lang is
// the person's language, used when the page's own is not clear.
func FromAI(ctx context.Context, c ai.Client, cfg ai.Config, page Page, lang string) (Draft, error) {
	text := Text(page.HTML)
	if r := []rune(text); len(r) > maxText {
		text = string(r[:maxText])
	}
	answer, err := c.Complete(ctx, cfg, []ai.Message{
		{Role: "system", Content: prompt},
		{Role: "user", Content: "Seite: " + page.URL + "\nSprache, wenn die Seite keine eindeutige hat: " + languageNames[langOr(lang)] + "\n\n" + text},
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
	var a answerParts
	for _, line := range strings.Split(cleanAnswer(answer), "\n") {
		a.line(line)
	}
	d := a.d
	d.Description = strings.Join(a.desc, " ")
	r := recipes.Parse(strings.Join(a.body, "\n"))
	for _, ing := range r.Ingredients() {
		d.Ingredients = append(d.Ingredients, ing.Line)
	}
	d.Steps = r.Steps
	return d
}

// answerParts collects the head lines, the description and the recipe body.
type answerParts struct {
	d          Draft
	body, desc []string
}

func (a *answerParts) line(line string) {
	t := strings.TrimSpace(line)
	switch {
	case a.d.Title == "" && strings.HasPrefix(t, "# "):
		a.d.Title = strings.TrimSpace(t[2:])
	case hasAnyPrefix(t, "Portionen:", "Servings:"):
		a.d.Servings = number(t)
	case hasAnyPrefix(t, "Zeit:", "Time:"):
		a.d.Time = number(t)
	case len(a.body) == 0 && !strings.HasPrefix(t, "## "):
		if t != "" {
			a.desc = append(a.desc, t)
		}
	default:
		a.body = append(a.body, line)
	}
}

func hasAnyPrefix(s string, prefixes ...string) bool {
	return slices.ContainsFunc(prefixes, func(p string) bool { return strings.HasPrefix(s, p) })
}

// cleanAnswer drops thinking (also an unpaired </think> or a cut-off
// <think>), any words before a fenced block and the fences themselves.
func cleanAnswer(s string) string {
	if i := strings.LastIndex(s, "</think>"); i >= 0 {
		s = s[i+len("</think>"):]
	} else if i := strings.Index(s, "<think>"); i >= 0 {
		s = s[:i]
	}
	lines := strings.Split(strings.TrimSpace(s), "\n")
	start := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), "```") })
	if start < 0 {
		return strings.TrimSpace(s)
	}
	lines = lines[start+1:]
	if end := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(strings.TrimSpace(l), "```") }); end >= 0 {
		lines = lines[:end]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func number(line string) int {
	n, _ := strconv.Atoi(firstInt.FindString(line))
	return n
}
