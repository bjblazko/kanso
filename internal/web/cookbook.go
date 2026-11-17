// SPDX-License-Identifier: Apache-2.0

package web

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/store"
)

// kansoAuthor records what Kanso changes by itself.
var kansoAuthor = store.Author{Name: "Kanso", Email: "kanso@localhost"}

// adoptCookbook marks the cookbook once (ADR-016): when none is marked yet and
// exactly one space holds recipes, and nothing but recipes, it becomes it.
func (s *Server) adoptCookbook() {
	if _, ok := s.Spaces.Cookbook(); ok {
		return
	}
	recipes, pages := map[string]bool{}, map[string]bool{}
	for _, id := range s.Wiki.IDs() {
		if pg, ok := s.Wiki.ByID(id); ok && pg.Meta.IsRecipe() {
			recipes[pg.Space] = true
		} else if ok {
			pages[pg.Space] = true
		}
	}
	if len(recipes) != 1 {
		return
	}
	for slug := range recipes {
		if pages[slug] {
			return
		}
		if err := s.Spaces.SetCookbook(kansoAuthor, slug); err != nil {
			slog.Warn("cookbook: marking the recipe space", "space", slug, "err", err)
		}
	}
}

// wikiSpaces are the spaces p may read, without the cookbook: its recipes live in the kitchen.
func (s *Server) wikiSpaces(p access.Principal) []spaces.Space {
	return slices.DeleteFunc(s.Spaces.Visible(p, s.groups()), func(sp spaces.Space) bool { return sp.Cookbook })
}

// cookbookFor is the cookbook, and the i18n key of why who may not create
// recipes in it ("" when they may).
func (s *Server) cookbookFor(who access.Principal) (spaces.Space, string) {
	cb, ok := s.Spaces.Cookbook()
	switch {
	case !ok:
		return cb, "recipe.no_cookbook"
	case !access.Can(s.Spaces.RoleOf(who, cb.Slug, s.groups()), access.Write):
		return cb, "recipe.no_cookbook_write"
	}
	return cb, ""
}

// createCookbook lets an admin make the cookbook when there is none.
func (s *Server) createCookbook(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	if _, ok := s.Spaces.Cookbook(); !ok {
		u, _ := auth.UserFrom(r.Context())
		v := s.newView(r, "", nil)
		sp, err := s.Spaces.Create(u.Author(), u.Sub, v.T("recipe.cookbook_name"), "")
		if err == nil {
			err = s.Spaces.SetCookbook(u.Author(), sp.Slug)
		}
		if err != nil {
			s.serverError(w, r, err)
			return
		}
	}
	http.Redirect(w, r, "/recipes", http.StatusSeeOther)
}
