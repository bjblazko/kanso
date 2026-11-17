// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"sort"
	"strings"

	"github.com/bjblazko/kanso/internal/ai"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/recipegen"
	"github.com/bjblazko/kanso/internal/recipeimport"
	"github.com/bjblazko/kanso/internal/recipes"
)

// generatedTag marks a recipe the AI made up and nobody has cooked yet.
const generatedTag = "ki-generiert"

// genChoice is one offered value of a wish; Key is its label's i18n key.
type genChoice struct {
	Value, Key string
	On         bool
}

type generatePage struct {
	Problem                              string // i18n key of why the person cannot generate
	AI                                   *aiTarget
	Error                                string
	Ingredients, Extra                   string
	Extend                               bool
	Servings                             int
	Temps, Heats, Styles, Times, Without []genChoice
}

// genRadiosArgs feed the template that offers one wish as radio buttons.
type genRadiosArgs struct {
	Name, Legend string
	Choices      []genChoice
	V            view
}

type hiddenField struct{ Name, Value string }

type generatePreview struct {
	Draft       recipeimport.Draft
	Body        string
	AI          *aiTarget
	Ingredients []recipes.Ingredient
	Wishes      []hiddenField // carried for "Neu würfeln"
	Again       string        // the form, filled in again
}

// genChoices offers keys under field, with the picked ones on.
func genChoices(field string, keys []string, picked ...string) []genChoice {
	out := make([]genChoice, 0, len(keys))
	for _, k := range keys {
		label := k
		if label == "" {
			label = "any"
		}
		out = append(out, genChoice{Value: k, Key: "gen." + field + "_" + label, On: slices.Contains(picked, k)})
	}
	return out
}

func newGeneratePage(w recipegen.Wishes) generatePage {
	return generatePage{Ingredients: strings.Join(w.Ingredients, ", "), Extra: w.Extra, Extend: w.Extend, Servings: w.Servings,
		Temps: genChoices("temp", recipegen.Temps, w.Temp), Heats: genChoices("heat", recipegen.Heats, w.Heat),
		Styles: genChoices("style", recipegen.Styles, w.Style), Times: genChoices("time", recipegen.Times, w.Time),
		Without: genChoices("without", recipegen.Withouts, w.Without...)}
}

// generateForm shows the wishes, filled from the query when "Wünsche ändern" led here.
func (s *Server) generateForm(w http.ResponseWriter, r *http.Request) {
	wishes, err := recipegen.Parse(r.URL.Query())
	if err != nil {
		wishes = recipegen.Wishes{Servings: 4}
	}
	s.renderGenerateForm(w, r, http.StatusOK, newGeneratePage(wishes), "")
}

func (s *Server) renderGenerateForm(w http.ResponseWriter, r *http.Request, status int, page generatePage, errKey string) {
	_, who := auth.UserFrom(r.Context())
	v := s.newView(r, "", nil)
	if _, problem := s.cookbookFor(who); problem != "" {
		page.Problem, status = problem, http.StatusForbidden
	} else if page.AI = s.aiTarget(r.Context()); page.AI == nil {
		page.Problem = "gen.err_no_ai"
	}
	if errKey != "" {
		page.Error = v.T(errKey)
	}
	v.Data, v.Title = page, v.T("gen.title")
	s.render(w, r, status, "recipe_generate", v)
}

// generateRecipe asks the AI for one recipe and shows it, not stored yet.
func (s *Server) generateRecipe(w http.ResponseWriter, r *http.Request) {
	_, who := auth.UserFrom(r.Context())
	if _, problem := s.cookbookFor(who); problem != "" {
		s.forbidden(w, r) // asking the AI only for a recipe one may create
		return
	}
	if err := r.ParseForm(); err != nil {
		s.fail(w, r, http.StatusBadRequest, "gen.err_invalid")
		return
	}
	wishes, err := recipegen.Parse(r.PostForm)
	if err != nil {
		s.renderGenerateForm(w, r, http.StatusBadRequest, newGeneratePage(recipegen.Wishes{Servings: 4}), "gen.err_invalid")
		return
	}
	if r.PostFormValue("action") == "reroll" {
		wishes = wishes.Reject(r.PostFormValue("rejected_now"))
	}
	v := s.newView(r, "", nil)
	draft, err := s.generate(r.Context(), wishes, v.Lang)
	if err != nil {
		status, key := generateError(err)
		s.renderGenerateForm(w, r, status, newGeneratePage(wishes), key)
		return
	}
	body := draft.Markdown(v.Lang)
	v.Title, v.Data = draft.Title, generatePreview{Draft: draft, Body: body, AI: s.aiTarget(r.Context()),
		Ingredients: recipes.Parse(body).Ingredients(), Wishes: hiddenFields(wishes), Again: "/recipes/generate?" + wishes.Values().Encode()}
	s.render(w, r, http.StatusOK, "recipe_generate_preview", v)
}

var errNoAI = errors.New("no AI connection")

func (s *Server) generate(ctx context.Context, wishes recipegen.Wishes, lang string) (recipeimport.Draft, error) {
	cfg := s.AI.Load()
	if !cfg.Ready() {
		return recipeimport.Draft{}, errNoAI
	}
	ctx, cancel := context.WithTimeout(ctx, ai.Timeout)
	defer cancel()
	return recipegen.Generate(ctx, ai.Client{}, cfg, wishes, lang)
}

func generateError(err error) (int, string) {
	switch {
	case errors.Is(err, errNoAI):
		return http.StatusBadRequest, "gen.err_no_ai"
	case errors.Is(err, recipeimport.ErrNoRecipe):
		return http.StatusBadRequest, "gen.err_no_recipe"
	}
	return http.StatusBadGateway, aiErrorKey(err)
}

// hiddenFields are the wishes as form fields, in a stable order.
func hiddenFields(w recipegen.Wishes) []hiddenField {
	vals := w.Values()
	keys := make([]string, 0, len(vals))
	for k := range vals {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []hiddenField
	for _, k := range keys {
		for _, x := range vals[k] {
			out = append(out, hiddenField{k, x})
		}
	}
	return out
}
