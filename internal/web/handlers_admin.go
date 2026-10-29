// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
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

// person is someone known to Kanso, for choosing in a form.
type person struct{ Sub, Name string }

type groupRow struct {
	Name       string
	Members    []person
	Candidates []person // known people not in the group yet
}

type groupsPage struct {
	Groups []groupRow
	Error  string
}

func (s *Server) groupsPage(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) {
		s.renderGroups(w, r, http.StatusOK, "")
	}
}

func (s *Server) renderGroups(w http.ResponseWriter, r *http.Request, status int, errKey string) {
	var people []person
	for _, u := range s.Users.List() {
		people = append(people, person{u.Sub, u.Name})
	}
	slices.SortFunc(people, func(a, b person) int { return strings.Compare(a.Name, b.Name) })
	data := groupsPage{Groups: groupRows(s.groups(), people)}
	v := s.newView(r, "", nil)
	if errKey != "" {
		data.Error = v.T(errKey)
	}
	v.Title, v.Data = v.T("nav.groups"), data
	s.render(w, r, status, "admin_groups", v)
}

// groupRows lists the groups by name, each with its members and the people
// who could still join.
func groupRows(g access.Groups, people []person) []groupRow {
	var rows []groupRow
	for name, subs := range g {
		row := groupRow{Name: name}
		for _, sub := range subs { // in the group, but never signed in
			if !slices.ContainsFunc(people, func(p person) bool { return p.Sub == sub }) {
				row.Members = append(row.Members, person{sub, sub})
			}
		}
		for _, p := range people {
			if slices.Contains(subs, p.Sub) {
				row.Members = append(row.Members, p)
			} else {
				row.Candidates = append(row.Candidates, p)
			}
		}
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b groupRow) int { return strings.Compare(a.Name, b.Name) })
	return rows
}

// errGroup carries the i18n key of a refused group change.
type errGroup string

func (e errGroup) Error() string { return string(e) }

func (s *Server) updateGroups(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	err := s.Repo.Locked(func() error {
		g := s.groups()
		if errKey := s.applyGroupChange(g, r); errKey != "" {
			return errGroup(errKey)
		}
		if err := access.SaveGroups(s.Cfg.DataDir, g); err != nil {
			return err
		}
		return s.Repo.Commit(u.Author(), "Gruppen geändert", access.ConfigFile)
	})
	var refused errGroup
	switch {
	case errors.As(err, &refused):
		s.renderGroups(w, r, http.StatusBadRequest, string(refused))
	case err != nil:
		s.serverError(w, r, err)
	default:
		http.Redirect(w, r, "/admin/groups", http.StatusSeeOther)
	}
}

// applyGroupChange changes g as the form says, or returns why it may not.
func (s *Server) applyGroupChange(g access.Groups, r *http.Request) string {
	name := strings.TrimSpace(r.PostFormValue("group"))
	sub := r.PostFormValue("sub")
	if _, exists := g[name]; !exists && r.PostFormValue("action") != "create" {
		return "groups.unknown"
	}
	switch r.PostFormValue("action") {
	case "create":
		return createGroup(g, name)
	case "add":
		return s.addToGroup(g, name, sub)
	case "remove":
		g[name] = slices.DeleteFunc(g[name], func(x string) bool { return x == sub })
		return ""
	case "delete":
		delete(g, name)
		return ""
	}
	return "settings.invalid"
}

// maxGroupName is the longest group name, in characters.
const maxGroupName = 40

// validGroupName allows names as people write them ("Großeltern", "Kita
// Gruppe"): letters, digits, spaces, "-" and "_".
func validGroupName(name string) bool {
	if name == "" || utf8.RuneCountInString(name) > maxGroupName {
		return false
	}
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != ' ' && c != '-' && c != '_' {
			return false
		}
	}
	return true
}

func createGroup(g access.Groups, name string) string {
	if !validGroupName(name) {
		return "groups.name_invalid"
	}
	if _, exists := g[name]; exists {
		return "groups.exists"
	}
	g[name] = []string{}
	return ""
}

func (s *Server) addToGroup(g access.Groups, name, sub string) string {
	if _, known := s.Users.Get(sub); !known {
		return "settings.unknown_member"
	}
	if !slices.Contains(g[name], sub) {
		g[name] = append(g[name], sub)
	}
	return ""
}
