// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
)

// moveConfirm asks before a page leaves its space for one with other readers.
type moveConfirm struct {
	pageCtx
	Target, TargetName, Before, After string
	Order                             string
}

// mayLeaveSpace checks a move of pc into another space: nobody may be editing
// the page or a subpage, and a change of readers needs a confirmation. It
// answers the request itself when the move may not go ahead now.
func (s *Server) mayLeaveSpace(w http.ResponseWriter, r *http.Request, pc pageCtx, space, parent string) bool {
	for _, id := range s.Wiki.Subtree(pc.Page.Meta.ID) {
		if lk, held := s.Locks.Holder(id); held {
			v := s.newView(r, "", nil)
			v.Data, v.Title = v.T("page.move_locked", lk.Name), v.T("page.move_locked", lk.Name)
			s.render(w, r, http.StatusConflict, "error", v)
			return false
		}
	}
	if r.PostFormValue("confirm") == "1" {
		return true
	}
	v := s.newView(r, "", nil)
	before, after := s.pageAudience(v, pc).Text, s.audienceAt(v, pc, space, parent).Text
	if before == after {
		return true
	}
	target, _ := s.Spaces.Get(space)
	v.Title = v.T("page.move_confirm", pc.Page.Meta.Title, target.Name)
	v.Data = moveConfirm{pageCtx: pc, Target: r.PostFormValue("parent"), TargetName: target.Name,
		Before: before, After: after, Order: r.PostFormValue("order")}
	s.render(w, r, http.StatusOK, "page_move_confirm", s.inSpace(v, pc.Space.Slug, pc.Page.Meta.ID))
	return false
}

// audienceAt is who would read pc below parent in space: that space's members,
// the shares of parent and the pages above it, and the page's own shares.
func (s *Server) audienceAt(v view, pc pageCtx, space, parent string) audience {
	target, _ := s.Spaces.Get(space)
	members := slices.Clone(target.Members)
	for _, m := range s.Wiki.Chain(parent) {
		members = append(members, m.Members...)
	}
	return s.membersAudience(v, append(members, pc.Page.Meta.Members...))
}
