// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/ai"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/recipeimport"
	"github.com/bjblazko/kanso/internal/recipes"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/wiki"
)

type importPage struct {
	Spaces []spaces.Space
	Space  string
	URL    string
	AIHost string // where page text goes when there is no recipe data; "" = no AI
	Error  string
}

type importPreview struct {
	Space       string
	Draft       recipeimport.Draft
	Body        string
	WithAI      bool
	Ingredients []recipes.Ingredient
	Photos      []previewPhoto
	Places      []string // "top", "step-1" … "step-n", "end"
}

func (s *Server) importForm(w http.ResponseWriter, r *http.Request) {
	s.renderImportForm(w, r, http.StatusOK, importPage{})
}

func (s *Server) renderImportForm(w http.ResponseWriter, r *http.Request, status int, page importPage) {
	_, who := auth.UserFrom(r.Context())
	page.Spaces = s.writableSpaces(who)
	if cfg := s.AI.Load(); cfg.Ready() {
		page.AIHost = cfg.Host()
	}
	v := s.newView(r, "", page)
	v.Title = v.T("import.title")
	s.render(w, r, status, "recipe_import", v)
}

// importRecipe fetches the page and shows a draft: from its recipe data, else from the AI.
func (s *Server) importRecipe(w http.ResponseWriter, r *http.Request) {
	page := importPage{URL: strings.TrimSpace(r.PostFormValue("url")), Space: r.PostFormValue("space")}
	if _, who := auth.UserFrom(r.Context()); !access.Can(s.Spaces.RoleOf(who, page.Space, s.groups()), access.Write) {
		s.forbidden(w, r) // fetching and asking the AI only for a recipe one may create
		return
	}
	v := s.newView(r, "", nil)
	draft, fetched, withAI, err := s.draftFrom(r.Context(), page.URL)
	if err != nil {
		page.Error = v.T(importErrorKey(err))
		s.renderImportForm(w, r, http.StatusBadRequest, page)
		return
	}
	body := draft.Markdown(v.Lang)
	v.Title, v.Data = draft.Title, importPreview{Space: page.Space, Draft: draft, Body: body, WithAI: withAI,
		Ingredients: recipes.Parse(body).Ingredients(), Photos: s.previewPhotos(r.Context(), draft, fetched),
		Places: places(len(draft.Steps))}
	s.render(w, r, http.StatusOK, "recipe_import_preview", v)
}

var errNoData = errors.New("no recipe data and no AI")

func (s *Server) draftFrom(ctx context.Context, url string) (recipeimport.Draft, recipeimport.Page, bool, error) {
	fctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	p, err := s.Fetch.Fetch(fctx, url)
	if err != nil {
		return recipeimport.Draft{}, p, false, err
	}
	if d, ok := recipeimport.FromJSONLD(p.HTML, p.URL); ok {
		return d, p, false, nil
	}
	cfg := s.AI.Load()
	if !cfg.Ready() {
		return recipeimport.Draft{}, p, false, errNoData
	}
	actx, cancel2 := context.WithTimeout(ctx, ai.Timeout)
	defer cancel2()
	d, err := recipeimport.FromAI(actx, ai.Client{}, cfg, p)
	return d, p, true, err
}

func importErrorKey(err error) string {
	switch {
	case errors.Is(err, recipeimport.ErrBlocked):
		return "import.err_blocked"
	case errors.Is(err, recipeimport.ErrNotHTML):
		return "import.err_not_html"
	case errors.Is(err, recipeimport.ErrFetch):
		return "import.err_fetch"
	case errors.Is(err, errNoData):
		return "import.err_no_data"
	case errors.Is(err, recipeimport.ErrNoRecipe):
		return "import.err_no_recipe"
	}
	return aiErrorKey(err)
}

// createImported makes the recipe page the person confirmed and opens its editor.
func (s *Server) createImported(w http.ResponseWriter, r *http.Request) {
	u, who := auth.UserFrom(r.Context())
	space := r.PostFormValue("space")
	if _, ok := s.Spaces.Get(space); !ok || !access.Can(s.Spaces.RoleOf(who, space, s.groups()), access.Write) {
		s.forbidden(w, r)
		return
	}
	p, err := s.Wiki.Create(u.Author(), space, "", r.PostFormValue("title"))
	if err == nil {
		body := strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n")
		placed, image := s.attachPhotos(r, u.Author(), p.Meta)
		body = recipeimport.InsertPhotos(body, placed)
		err = s.Wiki.SaveWith(u.Author(), p.Meta.ID, p.Meta.Title, body, func(m *wiki.PageMeta) {
			m.Type, m.Servings, m.Time = wiki.TypeRecipe, min(positive(r.PostFormValue("servings")), recipes.MaxServings), positive(r.PostFormValue("time"))
			m.Source, m.Image = strings.TrimSpace(r.PostFormValue("source")), image
		})
	}
	if err != nil {
		s.fail(w, r, http.StatusBadRequest, pageErrorKey(err))
		return
	}
	http.Redirect(w, r, "/p/"+p.Meta.ID+"/edit", http.StatusSeeOther)
}
