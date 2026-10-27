// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
	"strconv"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/todos"
)

type shareRow struct {
	Name  string
	Group bool
}

type sharePage struct {
	List   todos.List
	Shares []shareRow
	Users  map[string]string // sub → name, everyone but the owner
	Groups []string
	Error  string
}

// ownTodoList loads the signed-in person's own list {id}; anyone else,
// including people it is shared with, gets 404.
func (s *Server) ownTodoList(w http.ResponseWriter, r *http.Request) (todos.List, bool) {
	u, _ := auth.UserFrom(r.Context())
	l, err := s.Todos.Get(u.Sub, r.PathValue("id"), s.groups())
	if err != nil || l.Owner != u.Sub {
		s.notFound(w, r)
		return todos.List{}, false
	}
	return l, true
}

func (s *Server) shareTodoForm(w http.ResponseWriter, r *http.Request) {
	if l, ok := s.ownTodoList(w, r); ok {
		s.renderShare(w, r, http.StatusOK, l, "")
	}
}

func (s *Server) renderShare(w http.ResponseWriter, r *http.Request, status int, l todos.List, errKey string) {
	data := sharePage{List: l, Users: map[string]string{}}
	for _, sh := range l.Shared {
		data.Shares = append(data.Shares, shareRow{Name: s.shareName(sh), Group: sh.Group != ""})
	}
	for _, u := range s.Users.List() {
		if u.Sub != l.Owner {
			data.Users[u.Sub] = u.Name
		}
	}
	for g := range s.groups() {
		data.Groups = append(data.Groups, g)
	}
	slices.Sort(data.Groups)
	v := s.newView(r, "", nil)
	if errKey != "" {
		data.Error = v.T(errKey)
	}
	v.Title, v.Data = v.T("todos.share_title", l.Name), data
	s.render(w, r, status, "todo_share", v)
}

func (s *Server) shareTodo(w http.ResponseWriter, r *http.Request) {
	l, ok := s.ownTodoList(w, r)
	if !ok {
		return
	}
	shares, errKey := s.sharesFromForm(l, r)
	if errKey == "" && s.Todos.SetShares(l.Owner, l.ID, shares) != nil {
		errKey = "settings.invalid"
	}
	if errKey != "" {
		s.renderShare(w, r, http.StatusBadRequest, l, errKey)
		return
	}
	http.Redirect(w, r, "/todos/"+l.ID+"/share", http.StatusSeeOther)
}

// sharesFromForm returns the shares with the form's change, or an i18n error key.
func (s *Server) sharesFromForm(l todos.List, r *http.Request) ([]todos.Share, string) {
	shares := slices.Clone(l.Shared)
	switch r.PostFormValue("action") {
	case "add":
		m, ok := s.whoFromForm(r)
		sh := todos.Share{User: m.User, Group: m.Group}
		if !ok || sh.User == l.Owner {
			return nil, "settings.unknown_member"
		}
		if !slices.Contains(shares, sh) {
			shares = append(shares, sh)
		}
		return shares, ""
	case "remove":
		i, err := strconv.Atoi(r.PostFormValue("index"))
		if err != nil || i < 0 || i >= len(shares) {
			return nil, "settings.invalid"
		}
		return slices.Delete(shares, i, i+1), ""
	}
	return nil, "settings.invalid"
}

func (s *Server) confirmDeleteTodo(w http.ResponseWriter, r *http.Request) {
	l, ok := s.ownTodoList(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", l)
	v.Title = v.T("todos.delete_confirm", l.Name)
	s.render(w, r, http.StatusOK, "todo_delete", v)
}

func (s *Server) deleteTodo(w http.ResponseWriter, r *http.Request) {
	l, ok := s.ownTodoList(w, r)
	if !ok {
		return
	}
	if err := s.Todos.Delete(l.Owner, l.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/todos", http.StatusSeeOther)
}
