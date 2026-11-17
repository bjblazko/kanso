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

type homePage struct {
	Audiences map[string]audience // by space slug
	Deleted   string              // name of the space just deleted
}

// spaceNamed tells whether a space is called name.
func (s *Server) spaceNamed(name string) bool {
	for _, sp := range s.Spaces.List() {
		if sp.Name == name {
			return true
		}
	}
	return false
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
	if sp.Cookbook { // its recipes live in the kitchen (ADR-016)
		http.Redirect(w, r, "/recipes", http.StatusSeeOther)
		return
	}
	v := s.inSpace(s.newView(r, sp.Name, spacePage{Space: sp, Role: role}), sp.Slug, "")
	v.Audience = ptr(s.spaceAudienceFor(v, sp, role))
	s.render(w, r, http.StatusOK, "space", v)
}

type settingsPage struct {
	Space  spaces.Space
	Icons  []string
	Users  map[string]string // sub → name
	Groups []string
	Roles  []access.Role
	Error  string
	Back   string // the space, or the admins' list for an admin who cannot read it
}

// spaceToManage resolves the {space} for whoever may manage it: its owners, and
// admins even without reading it (ADR-014). Readers get 403, everyone else 404.
func (s *Server) spaceToManage(w http.ResponseWriter, r *http.Request) (spaces.Space, bool) {
	_, p := auth.UserFrom(r.Context())
	sp, ok := s.Spaces.Get(r.PathValue("space"))
	role := s.Spaces.RoleOf(p, sp.Slug, s.groups())
	switch {
	case ok && (access.Can(role, access.ManageSpace) || p.Admin):
		return sp, true
	case ok && access.Can(role, access.Read):
		s.forbidden(w, r)
	default:
		s.notFound(w, r)
	}
	return spaces.Space{}, false
}

func (s *Server) spaceSettings(w http.ResponseWriter, r *http.Request) {
	if sp, ok := s.spaceToManage(w, r); ok {
		s.renderSettings(w, r, http.StatusOK, sp, "")
	}
}

func (s *Server) renderSettings(w http.ResponseWriter, r *http.Request, status int, sp spaces.Space, errKey string) {
	data := settingsPage{Space: sp, Icons: spaces.Icons, Users: map[string]string{},
		Roles: []access.Role{access.Viewer, access.Editor, access.Owner}, Back: "/s/" + sp.Slug}
	if _, p := auth.UserFrom(r.Context()); !access.Can(s.Spaces.RoleOf(p, sp.Slug, s.groups()), access.Read) {
		data.Back = "/admin/spaces"
	}
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
	sp, ok := s.spaceToManage(w, r)
	if !ok {
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
