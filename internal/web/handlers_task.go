// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/tasks"
)

type taskView struct {
	Card      card
	BoardName string
	HTML      template.HTML
	Info      markdown.Info
	Private   bool
	// Note is shown only to the task's author: "link", "gone" or "".
	Note       string
	AuthorName string
}

type taskEdit struct {
	Task     tasks.Task
	Version  string
	Error    string
	Boards   []boardTab
	Statuses []column // Key = status, Label = its name
}

func taskPath(t tasks.Task) string { return "/tasks/b/" + t.Board + "/" + t.Meta.ID }

// taskFor loads the task of the URL; anyone who does not see it gets 404.
func (s *Server) taskFor(w http.ResponseWriter, r *http.Request) (tasks.Task, bool) {
	t, err := s.Tasks.Get(s.actor(r), r.PathValue("board"), r.PathValue("task"))
	if err != nil {
		s.notFound(w, r)
		return tasks.Task{}, false
	}
	return t, true
}

func (s *Server) showTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	b, _ := s.Tasks.Board(s.actor(r), t.Board)
	html, info, err := s.MD.Render([]byte(t.Body), markdown.PageCtx{Base: taskPath(t)})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, "", nil)
	tv := taskView{Card: s.cards(v, []tasks.Task{t}, time.Now())[0], BoardName: boardName(v, b), HTML: html, Info: info,
		Private: b.Private(), Note: s.noteState(v, t), AuthorName: s.userName(t.Meta.Author)}
	v.Title, v.Data, v.Here = t.Meta.Title, tv, tasksHere
	v.Audience = ptr(s.boardAudience(v, b))
	s.render(w, r, http.StatusOK, "task_view", v)
}

// noteState says whether the viewer, as the task's author, gets a link to the note it came from.
func (s *Server) noteState(v view, t tasks.Task) string {
	if t.Meta.Note == "" || t.Meta.Author != v.User.Sub {
		return ""
	}
	if _, err := s.Notes.Get(v.User.Sub, t.Meta.Note); errors.Is(err, notes.ErrNotFound) {
		return "gone"
	}
	return "link"
}

func (s *Server) editTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	s.renderTaskEditor(w, r, http.StatusOK, taskEdit{Task: t, Version: t.Meta.Updated.Format(time.RFC3339Nano)})
}

func (s *Server) renderTaskEditor(w http.ResponseWriter, r *http.Request, status int, data taskEdit) {
	v := s.newView(r, "", nil)
	bs, err := s.Tasks.Boards(s.actor(r))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	for _, b := range bs {
		data.Boards = append(data.Boards, boardTab{ID: b.ID, Name: boardName(v, b), Current: b.ID == data.Task.Board})
	}
	for _, st := range tasks.Statuses {
		data.Statuses = append(data.Statuses, column{Key: string(st), Label: v.T("tasks.status." + string(st))})
	}
	v.Title, v.Data, v.Here = data.Task.Meta.Title, data, tasksHere
	s.render(w, r, status, "task_edit", v)
}

func editFromForm(r *http.Request) tasks.Edit {
	return tasks.Edit{
		Title: r.PostFormValue("title"), Body: strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n"),
		Status: tasks.Status(r.PostFormValue("status")), Urgent: r.PostFormValue("urgent") != "",
		Important: r.PostFormValue("important") != "", Due: r.PostFormValue("due"), WaitingFor: r.PostFormValue("waiting_for"),
	}
}

// saveTask stores the edit form unless the task changed meanwhile, then moves
// it when another board was chosen.
func (s *Server) saveTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	a, e, to := s.actor(r), editFromForm(r), r.PostFormValue("board")
	if _, err := s.Tasks.Board(a, to); to != "" && err != nil {
		s.notFound(w, r)
		return
	}
	version, _ := time.Parse(time.RFC3339Nano, r.PostFormValue("version"))
	saved, err := s.Tasks.Update(a, t.Board, t.Meta.ID, e, version)
	if err == nil && to != "" {
		saved, err = s.Tasks.Move(a, t.Board, t.Meta.ID, to)
	}
	s.taskSaved(w, r, t, e, saved, err)
}

// taskSaved answers the edit form after saving.
func (s *Server) taskSaved(w http.ResponseWriter, r *http.Request, t tasks.Task, e tasks.Edit, saved tasks.Task, err error) {
	switch {
	case errors.Is(err, tasks.ErrConflict):
		s.taskFormError(w, r, t, e, http.StatusConflict, "conflict", "tasks.conflict")
	case errors.Is(err, tasks.ErrInvalid):
		s.taskFormError(w, r, t, e, http.StatusBadRequest, "invalid", "tasks.invalid")
	case err != nil:
		s.serverError(w, r, err)
	case wantsJSON(r):
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved", "url": taskPath(saved)})
	default:
		http.Redirect(w, r, taskPath(saved), http.StatusSeeOther)
	}
}

// taskFormError answers a failed save; the form keeps what was typed.
func (s *Server) taskFormError(w http.ResponseWriter, r *http.Request, t tasks.Task, e tasks.Edit, status int, code, key string) {
	if wantsJSON(r) {
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	typed := t
	typed.Body, typed.Meta.Title, typed.Meta.Status, typed.Meta.Due = e.Body, e.Title, e.Status, e.Due
	typed.Meta.Urgent, typed.Meta.Important, typed.Meta.WaitingFor = e.Urgent, e.Important, e.WaitingFor
	v := s.newView(r, "", nil)
	s.renderTaskEditor(w, r, status, taskEdit{Task: typed, Version: t.Meta.Updated.Format(time.RFC3339Nano), Error: v.T(key)})
}

func (s *Server) claimTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	_, err := s.Tasks.Claim(s.actor(r), t.Board, t.Meta.ID, r.PostFormValue("claim") == "1")
	switch {
	case errors.Is(err, tasks.ErrPrivate):
		s.fail(w, r, http.StatusBadRequest, "tasks.claim_private")
	case err != nil:
		s.serverError(w, r, err)
	default:
		http.Redirect(w, r, taskPath(t), http.StatusSeeOther)
	}
}

func (s *Server) confirmDeleteTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", nil)
	v.Title, v.Data, v.Here = v.T("tasks.delete"), t, tasksHere
	s.render(w, r, http.StatusOK, "task_delete", v)
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request) {
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	if err := s.Tasks.Delete(s.actor(r), t.Board, t.Meta.ID); err != nil {
		s.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/tasks/b/"+t.Board, http.StatusSeeOther)
}

// taskPreview renders Markdown for the task editor; ref is "<board>/<task>".
func (s *Server) taskPreview(w http.ResponseWriter, r *http.Request) {
	board, task, _ := strings.Cut(r.PostFormValue("task"), "/")
	r.SetPathValue("board", board)
	r.SetPathValue("task", task)
	t, ok := s.taskFor(w, r)
	if !ok {
		return
	}
	s.writePreview(w, r.PostFormValue("body"), markdown.PageCtx{Base: taskPath(t)})
}

// moveTask is what dragging a card does: a new column (status) or a new
// quadrant (urgent and important). It changes one field and always wins.
func (s *Server) moveTask(w http.ResponseWriter, r *http.Request) {
	a, board, id := s.actor(r), r.PathValue("board"), r.PathValue("task")
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
		return
	}
	var t tasks.Task
	err := tasks.ErrInvalid
	switch {
	case r.PostForm.Has("status"):
		t, err = s.Tasks.SetStatus(a, board, id, tasks.Status(r.PostFormValue("status")))
	case r.PostForm.Has("urgent") && r.PostForm.Has("important"):
		t, err = s.Tasks.SetQuadrant(a, board, id, r.PostFormValue("urgent") == "1", r.PostFormValue("important") == "1")
	}
	switch {
	case errors.Is(err, tasks.ErrNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "gone"})
	case errors.Is(err, tasks.ErrInvalid):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
	case err != nil:
		s.serverError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, map[string]any{"status": t.Meta.Status, "urgent": t.Meta.Urgent, "important": t.Meta.Important})
	}
}

// findTaskByID sends a link by task id to wherever the task is now; notes link this way,
// so the link survives moving the task to another board.
func (s *Server) findTaskByID(w http.ResponseWriter, r *http.Request) {
	t, err := s.Tasks.Find(s.actor(r), r.PathValue("task"))
	if err != nil {
		s.notFound(w, r)
		return
	}
	http.Redirect(w, r, taskPath(t), http.StatusSeeOther)
}
