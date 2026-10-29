// SPDX-License-Identifier: Apache-2.0

package web

import (
	"log/slog"
	"net/http"
	"slices"
	"strconv"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/spaces"
)

// groups reads the current groups on every request, so changes apply at once.
func (s *Server) groups() access.Groups {
	g, err := access.LoadGroups(s.Cfg.DataDir)
	if err != nil {
		slog.Error("load groups", "err", err)
		return access.Groups{}
	}
	return g
}

// spaceFor resolves the {space} of the request and the caller's role in it.
// It answers 404 itself when the space is missing or invisible to the caller.
func (s *Server) spaceFor(w http.ResponseWriter, r *http.Request) (spaces.Space, access.Role, bool) {
	_, p := auth.UserFrom(r.Context())
	sp, ok := s.Spaces.Get(r.PathValue("space"))
	role := s.Spaces.RoleOf(p, sp.Slug, s.groups())
	if !ok || !access.Can(role, access.Read) {
		s.notFound(w, r)
		return spaces.Space{}, access.None, false
	}
	return sp, role, true
}

// home lists the visible spaces, each with who else sees it.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	seen := map[string]audience{}
	for _, sp := range v.Nav {
		seen[sp.Slug] = s.spaceAudience(v, sp)
	}
	v.Data = seen
	s.render(w, r, http.StatusOK, "home", v)
}

type spacePage struct {
	Space spaces.Space
	Role  access.Role
}

// CanManage tells templates whether to offer the settings.
func (p spacePage) CanManage() bool { return access.Can(p.Role, access.ManageSpace) }

// CanWrite tells templates whether to offer editing.
func (p spacePage) CanWrite() bool { return access.Can(p.Role, access.Write) }

func (s *Server) showSpace(w http.ResponseWriter, r *http.Request) {
	sp, role, ok := s.spaceFor(w, r)
	if !ok {
		return
	}
	v := s.inSpace(s.newView(r, sp.Name, spacePage{Space: sp, Role: role}), sp.Slug, "")
	v.Audience = ptr(s.spaceAudience(v, sp))
	s.render(w, r, http.StatusOK, "space", v)
}

type settingsPage struct {
	Space  spaces.Space
	Icons  []string
	Users  map[string]string // sub → name
	Groups []string
	Roles  []access.Role
	Error  string
}

func (s *Server) spaceSettings(w http.ResponseWriter, r *http.Request) {
	sp, role, ok := s.spaceFor(w, r)
	if !ok {
		return
	}
	if !access.Can(role, access.ManageSpace) {
		s.forbidden(w, r)
		return
	}
	s.renderSettings(w, r, http.StatusOK, sp, "")
}

func (s *Server) renderSettings(w http.ResponseWriter, r *http.Request, status int, sp spaces.Space, errKey string) {
	data := settingsPage{Space: sp, Icons: spaces.Icons, Users: map[string]string{},
		Roles: []access.Role{access.Viewer, access.Editor, access.Owner}}
	for _, u := range s.Users.List() {
		data.Users[u.Sub] = u.Name
	}
	for g := range s.groups() {
		data.Groups = append(data.Groups, g)
	}
	slices.Sort(data.Groups)
	v := s.newView(r, sp.Name, data)
	if errKey != "" {
		data.Error = v.T(errKey)
		v.Data = data
	}
	s.render(w, r, status, "space_settings", v)
}

func (s *Server) updateSpaceSettings(w http.ResponseWriter, r *http.Request) {
	sp, role, ok := s.spaceFor(w, r)
	if !ok {
		return
	}
	if !access.Can(role, access.ManageSpace) {
		s.forbidden(w, r)
		return
	}
	changed, errKey := s.applySettings(sp, r)
	if errKey != "" {
		s.renderSettings(w, r, http.StatusBadRequest, sp, errKey)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Spaces.Update(u.Author(), changed); err != nil {
		s.renderSettings(w, r, http.StatusBadRequest, sp, "settings.invalid")
		return
	}
	http.Redirect(w, r, "/s/"+sp.Slug+"/settings", http.StatusSeeOther)
}

// applySettings returns the space with the form's change, or an i18n error key.
func (s *Server) applySettings(sp spaces.Space, r *http.Request) (spaces.Space, string) {
	switch r.PostFormValue("action") {
	case "save":
		sp.Name, sp.Description, sp.Icon = r.PostFormValue("name"), r.PostFormValue("description"), r.PostFormValue("icon")
		return sp, ""
	case "add":
		m, errKey := s.memberFromForm(r)
		sp.Members = append(slices.Clone(sp.Members), m)
		return sp, errKey
	case "remove":
		i, err := strconv.Atoi(r.PostFormValue("index"))
		if err != nil || i < 0 || i >= len(sp.Members) {
			return sp, "settings.invalid"
		}
		sp.Members = slices.Delete(slices.Clone(sp.Members), i, i+1)
		return sp, ""
	}
	return sp, "settings.invalid"
}

func (s *Server) memberFromForm(r *http.Request) (access.Member, string) {
	role, err := access.ParseRole(r.PostFormValue("role"))
	if err != nil {
		return access.Member{}, "settings.invalid"
	}
	m, ok := s.whoFromForm(r)
	if !ok {
		return access.Member{}, "settings.unknown_member"
	}
	m.Role = role
	return m, ""
}

// whoFromForm reads the known person or group a form names (fields kind, who).
func (s *Server) whoFromForm(r *http.Request) (access.Member, bool) {
	who := r.PostFormValue("who")
	switch r.PostFormValue("kind") {
	case "user":
		_, ok := s.Users.Get(who)
		return access.Member{User: who}, ok
	case "group":
		_, ok := s.groups()[who]
		return access.Member{Group: who}, ok
	}
	return access.Member{}, false
}
