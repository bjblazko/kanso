// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/tasks"
)

type noteTask struct {
	noteView
	Boards []boardTab
}

// noteTaskForm asks on which board the task made from a note goes.
func (s *Server) noteTaskForm(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	bs, err := s.Tasks.Boards(s.actor(r))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, "", nil)
	data := noteTask{noteView: noteView{Note: n, Title: noteTitle(v, n)}}
	for i, b := range bs {
		data.Boards = append(data.Boards, boardTab{ID: b.ID, Name: boardName(v, b), Current: i == 0})
	}
	v.Title, v.Data, v.Here = v.T("notes.to_task"), data, notesHere
	s.render(w, r, http.StatusOK, "note_task", v)
}

// noteTaskID is the id of the task made from n, if that task still exists.
func (s *Server) noteTaskID(r *http.Request, n notes.Note) string {
	_, id, _ := strings.Cut(n.Meta.Task, "/")
	if _, err := s.Tasks.Find(s.actor(r), id); err != nil {
		return ""
	}
	return id
}

// noteToTask makes a task from a note: its first line is the subject, the rest
// the text. The note stays; both point at each other.
func (s *Server) noteToTask(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", nil)
	t, _, err := s.Tasks.Create(s.actor(r), taskFromNote(v, n, r.PostFormValue("board")))
	switch {
	case errors.Is(err, tasks.ErrInvalid):
		s.fail(w, r, http.StatusBadRequest, "notes.to_task_invalid")
		return
	case errors.Is(err, tasks.ErrFull):
		s.fail(w, r, http.StatusBadRequest, "tasks.full")
		return
	case err != nil:
		s.serverError(w, r, err)
		return
	}
	if err := s.Notes.LinkTask(n.Owner, n.Meta.ID, t.Board+"/"+t.Meta.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, taskPath(t)+"/edit", http.StatusSeeOther)
}

func taskFromNote(v view, n notes.Note, board string) tasks.New {
	title, rest := tasks.FromNote(n.Body)
	if title == "" { // only photos: the subject is the note's name; the photos stay with the note
		title = noteTitle(v, n)
	}
	return tasks.New{Board: board, Title: title, Body: rest, Note: n.Meta.ID}
}
