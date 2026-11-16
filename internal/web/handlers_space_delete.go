// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/wiki"
)

// spaceDelete is what the confirmation says will be lost.
type spaceDelete struct {
	Space   spaces.Space
	Stats   wiki.SpaceStats
	Planned int // meal plan entries cooking a recipe of the space
	Error   string
}

// NeedsName tells whether the name must be typed: only when something is lost.
func (d spaceDelete) NeedsName() bool { return d.Stats.Pages > 0 }

func (s *Server) spaceDeleteData(sp spaces.Space) spaceDelete {
	st := s.Wiki.SpaceStats(sp.Slug)
	ids := make(map[string]bool, len(st.IDs))
	for _, id := range st.IDs {
		ids[id] = true
	}
	return spaceDelete{Space: sp, Stats: st, Planned: s.MealPlans.Uses(ids)}
}

func (s *Server) renderSpaceDelete(w http.ResponseWriter, r *http.Request, status int, d spaceDelete) {
	s.render(w, r, status, "space_delete", s.newView(r, d.Space.Name, d))
}

func (s *Server) deleteSpaceForm(w http.ResponseWriter, r *http.Request) {
	if sp, ok := s.spaceToManage(w, r); ok {
		s.renderSpaceDelete(w, r, http.StatusOK, s.spaceDeleteData(sp))
	}
}

func (s *Server) deleteSpace(w http.ResponseWriter, r *http.Request) {
	sp, ok := s.spaceToManage(w, r)
	if !ok {
		return
	}
	d := s.spaceDeleteData(sp)
	v := s.newView(r, "", nil)
	if d.NeedsName() && strings.TrimSpace(r.PostFormValue("name")) != sp.Name {
		d.Error = v.T("space.delete_wrong_name")
		s.renderSpaceDelete(w, r, http.StatusBadRequest, d)
		return
	}
	if holder, busy := s.firstLock(d.Stats.IDs); busy {
		d.Error = v.T("space.delete_locked", holder)
		s.renderSpaceDelete(w, r, http.StatusConflict, d)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Spaces.Delete(u.Author(), sp.Slug, func() { s.Wiki.ForgetSpace(sp.Slug) }); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/?deleted="+url.QueryEscape(sp.Name), http.StatusSeeOther)
}

// firstLock names someone editing one of the pages ids.
func (s *Server) firstLock(ids []string) (string, bool) {
	for _, id := range ids {
		if l, ok := s.Locks.Holder(id); ok {
			return l.Name, true
		}
	}
	return "", false
}
