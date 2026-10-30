// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/wiki"
)

// inheritedShare is who a page above shares with; it applies here too.
type inheritedShare struct {
	From    wiki.PageMeta
	Members []access.Member
}

// CanManageSpace tells the template whether to link the space's settings.
func (p pageSharePage) CanManageSpace() bool { return p.SpaceManager }

type pageSharePage struct {
	SpaceManager bool
	pageCtx
	Members     []access.Member
	Inherited   []inheritedShare
	Descendants int
	Users       map[string]string
	Groups      []string
	Roles       []access.Role
	Error       string
}

func (s *Server) pageShare(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.ManageSpace)
	if ok {
		s.renderPageShare(w, r, http.StatusOK, pc, "")
	}
}

func (s *Server) renderPageShare(w http.ResponseWriter, r *http.Request, status int, pc pageCtx, errKey string) {
	data := pageSharePage{SpaceManager: access.Can(s.Spaces.RoleOf(pc.Who, pc.Space.Slug, s.groups()), access.ManageSpace), pageCtx: pc, Members: pc.Page.Meta.Members, Descendants: s.Wiki.Descendants(pc.Page.Meta.ID),
		Users: map[string]string{}, Roles: []access.Role{access.Viewer, access.Editor, access.Owner}}
	for _, m := range s.Wiki.Chain(pc.Page.Meta.ID)[1:] {
		if len(m.Members) > 0 {
			data.Inherited = append(data.Inherited, inheritedShare{From: m, Members: m.Members})
		}
	}
	for _, u := range s.Users.List() {
		data.Users[u.Sub] = u.Name
	}
	for g := range s.groups() {
		data.Groups = append(data.Groups, g)
	}
	slices.Sort(data.Groups)
	v := s.newView(r, pc.Page.Meta.Title, nil)
	if errKey != "" {
		data.Error = v.T(errKey)
	}
	v.Data = data
	s.render(w, r, status, "page_share", s.inSpace(v, pc.Space.Slug, pc.Page.Meta.ID))
}

// updatePageShare adds or removes one member of the page.
func (s *Server) updatePageShare(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.ManageSpace)
	if !ok {
		return
	}
	members, errKey := s.changedMembers(pc.Page.Meta.Members, r)
	if errKey != "" {
		s.renderPageShare(w, r, http.StatusBadRequest, pc, errKey)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Wiki.SetMembers(u.Author(), pc.Page.Meta.ID, members); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/p/"+pc.Page.Meta.ID+"/share", http.StatusSeeOther)
}

// changedMembers applies the form's add or remove to members, or returns an i18n error key.
func (s *Server) changedMembers(members []access.Member, r *http.Request) ([]access.Member, string) {
	switch r.PostFormValue("action") {
	case "add":
		m, errKey := s.memberFromForm(r)
		return append(slices.Clone(members), m), errKey
	case "remove":
		i, err := strconv.Atoi(r.PostFormValue("index"))
		if err != nil || i < 0 || i >= len(members) {
			return members, "settings.invalid"
		}
		return slices.Delete(slices.Clone(members), i, i+1), ""
	}
	return members, "settings.invalid"
}
