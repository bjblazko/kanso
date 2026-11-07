// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/wiki"
)

// recipeMetaFrom reads the editor's recipe fields; without the "recipe" box
// the page is no recipe and all recipe fields go.
func recipeMetaFrom(r *http.Request) func(*wiki.PageMeta) {
	tags := recipes.NormalizeTags(r.PostFormValue("tags")) // every page may have tags
	if r.PostFormValue("recipe") == "" {
		return func(m *wiki.PageMeta) { m.Type, m.Servings, m.Tags, m.Source, m.Time = "", 0, tags, "", 0 }
	}
	servings, minutes := positive(r.PostFormValue("servings")), positive(r.PostFormValue("time"))
	source := strings.TrimSpace(r.PostFormValue("source"))
	return func(m *wiki.PageMeta) {
		m.Type, m.Servings, m.Tags, m.Source, m.Time = wiki.TypeRecipe, min(servings, recipes.MaxServings), tags, source, minutes
	}
}

// positive reads a whole number; anything else, or below 1, is 0.
func positive(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// recognised renders how Kanso read a recipe's ingredients, for the editor preview.
func (s *Server) recognised(r *http.Request, body string) (template.HTML, error) {
	v := s.newView(r, "", recipes.Parse(body).Ingredients())
	var buf bytes.Buffer
	if err := s.pages["recipe_parsed"].ExecuteTemplate(&buf, "recipe-parsed", v); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil // #nosec G203 -- executed by html/template
}
