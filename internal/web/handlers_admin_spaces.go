// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/spaces"
)

// spaceRow is one space in the admins' list: who reads it and how much it holds, never its content.
type spaceRow struct {
	Space    spaces.Space
	Audience audience
	Pages    int
	Orphan   bool // no members: nobody can read it
}

// adminSpaces lists every space for admins, to manage it without reading it (ADR-014).
func (s *Server) adminSpaces(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	v := s.newView(r, "", nil)
	v.Title = v.T("admin.spaces")
	var rows []spaceRow
	for _, sp := range s.Spaces.List() {
		rows = append(rows, spaceRow{Space: sp, Audience: s.spaceAudience(v, sp),
			Pages: s.Wiki.SpaceStats(sp.Slug).Pages, Orphan: len(sp.Members) == 0})
	}
	v.Data = rows
	s.render(w, r, http.StatusOK, "admin_spaces", v)
}

// claimSpace enters the admin as owner of a space nobody can read; a space with members is theirs to share.
func (s *Server) claimSpace(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	sp, ok := s.Spaces.Get(r.PostFormValue("space"))
	if r.PostFormValue("action") == "cookbook" && ok {
		s.moveCookbook(w, r, sp)
		return
	}
	if r.PostFormValue("action") != "claim" || !ok || len(sp.Members) > 0 {
		s.fail(w, r, http.StatusBadRequest, "settings.invalid")
		return
	}
	u, _ := auth.UserFrom(r.Context())
	sp.Members = []access.Member{{User: u.Sub, Role: access.Owner}}
	if err := s.Spaces.Update(u.Author(), sp); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/s/"+sp.Slug, http.StatusSeeOther)
}

// moveCookbook makes sp the cookbook (ADR-016).
func (s *Server) moveCookbook(w http.ResponseWriter, r *http.Request, sp spaces.Space) {
	u, _ := auth.UserFrom(r.Context())
	if err := s.Spaces.SetCookbook(u.Author(), sp.Slug); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/admin/spaces", http.StatusSeeOther)
}
