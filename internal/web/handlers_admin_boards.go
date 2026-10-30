// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"slices"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/tasks"
)

type adminBoardsPage struct {
	Boards  []tasks.Board
	Groups  []string
	Confirm *tasks.Board // board whose deletion waits for confirmation
	Error   string
}

func (s *Server) boardsPage(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) {
		s.renderAdminBoards(w, r, http.StatusOK, adminBoardsPage{})
	}
}

func (s *Server) renderAdminBoards(w http.ResponseWriter, r *http.Request, status int, data adminBoardsPage) {
	v := s.newView(r, "", nil)
	data.Boards = s.Tasks.GroupBoards()
	for name := range s.groups() {
		data.Groups = append(data.Groups, name)
	}
	slices.Sort(data.Groups)
	v.Title, v.Data = v.T("tasks.admin_boards"), data
	s.render(w, r, status, "admin_boards", v)
}

// updateBoards creates, renames or (after confirmation) deletes a group board.
func (s *Server) updateBoards(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	id := r.PostFormValue("id")
	var err error
	switch r.PostFormValue("action") {
	case "create":
		if _, known := s.groups()[r.PostFormValue("group")]; !known {
			err = tasks.ErrInvalid
		} else {
			_, err = s.Tasks.CreateGroupBoard(u.Author(), r.PostFormValue("name"), r.PostFormValue("group"))
		}
	case "rename":
		err = s.Tasks.RenameBoard(u.Author(), id, r.PostFormValue("name"))
	case "delete":
		s.confirmDeleteBoard(w, r, id)
		return
	case "delete_confirmed":
		err = s.Tasks.DeleteBoard(u.Author(), id)
	default:
		err = tasks.ErrInvalid
	}
	s.boardsChanged(w, r, err)
}

func (s *Server) confirmDeleteBoard(w http.ResponseWriter, r *http.Request, id string) {
	i := slices.IndexFunc(s.Tasks.GroupBoards(), func(b tasks.Board) bool { return b.ID == id })
	if i < 0 {
		s.notFound(w, r)
		return
	}
	s.renderAdminBoards(w, r, http.StatusOK, adminBoardsPage{Confirm: &s.Tasks.GroupBoards()[i]})
}

func (s *Server) boardsChanged(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, tasks.ErrNotFound):
		s.notFound(w, r)
	case errors.Is(err, tasks.ErrInvalid):
		v := s.newView(r, "", nil)
		s.renderAdminBoards(w, r, http.StatusBadRequest, adminBoardsPage{Error: v.T("tasks.admin_invalid")})
	case err != nil:
		s.serverError(w, r, err)
	default:
		http.Redirect(w, r, "/admin/boards", http.StatusSeeOther)
	}
}
