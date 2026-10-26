// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/store"
)

// requireAdmin answers 403 for everyone but admins.
func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if _, p := auth.UserFrom(r.Context()); !p.Admin {
		s.forbidden(w, r)
		return false
	}
	return true
}

func (s *Server) newSpaceForm(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) {
		s.page(w, r, "space_new", "", nil)
	}
}

func (s *Server) createSpace(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	sp, err := s.Spaces.Create(u.Author(), r.PostFormValue("name"), r.PostFormValue("description"))
	if err != nil {
		v := s.newView(r, "", nil)
		v.Data = v.T("space.name_required")
		s.render(w, r, http.StatusBadRequest, "space_new", v)
		return
	}
	http.Redirect(w, r, "/s/"+sp.Slug, http.StatusSeeOther)
}

type groupRow struct {
	Name    string
	Members []string // display names
	Subs    []string
}

type groupsPage struct {
	Groups []groupRow
	Users  map[string]string
}

func (s *Server) groupsPage(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	data := groupsPage{Users: map[string]string{}}
	for _, u := range s.Users.List() {
		data.Users[u.Sub] = u.Name
	}
	for name, subs := range s.groups() {
		row := groupRow{Name: name, Subs: subs}
		for _, sub := range subs {
			row.Members = append(row.Members, data.Users[sub])
		}
		data.Groups = append(data.Groups, row)
	}
	slices.SortFunc(data.Groups, func(a, b groupRow) int { return strings.Compare(a.Name, b.Name) })
	s.page(w, r, "admin_groups", "", data)
}

func (s *Server) updateGroups(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	err := s.Repo.Locked(func() error {
		g := s.groups()
		if !s.applyGroupChange(g, r) {
			return errBadRequest
		}
		if err := access.SaveGroups(s.Cfg.DataDir, g); err != nil {
			return err
		}
		return s.Repo.Commit(u.Author(), "Gruppen geändert", access.ConfigFile)
	})
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/admin/groups", http.StatusSeeOther)
}

func (s *Server) applyGroupChange(g access.Groups, r *http.Request) bool {
	name := strings.TrimSpace(r.PostFormValue("group"))
	sub := r.PostFormValue("sub")
	_, exists := g[name]
	switch r.PostFormValue("action") {
	case "create":
		if name == "" || exists || store.Slugify(name) != name {
			return false
		}
		g[name] = []string{}
	case "add":
		if _, ok := s.Users.Get(sub); !ok || !exists || slices.Contains(g[name], sub) {
			return false
		}
		g[name] = append(g[name], sub)
	case "remove":
		g[name] = slices.DeleteFunc(g[name], func(x string) bool { return x == sub })
	case "delete":
		delete(g, name)
	default:
		return false
	}
	return true
}
