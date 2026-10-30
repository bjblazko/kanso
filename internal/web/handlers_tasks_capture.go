// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/tasks"
)

// dueChoices are the quick due dates; the page puts the device's dates in
// when it runs, these are for browsers without JavaScript.
type dueChoices struct{ Today, Tomorrow, Week string }

type taskNew struct {
	Boards   []boardTab
	Selected string
	Saved    string // name of the board the last task went to
	Due      dueChoices
	Title    string
	Error    string
}

// dueDates returns today, tomorrow and the coming Sunday (today on Sundays).
func dueDates(now time.Time) dueChoices {
	sunday := now.AddDate(0, 0, (7-int(now.Weekday()))%7)
	f := func(t time.Time) string { return t.Format("2006-01-02") }
	return dueChoices{Today: f(now), Tomorrow: f(now.AddDate(0, 0, 1)), Week: f(sunday)}
}

func (s *Server) newTaskForm(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	s.renderTaskCapture(w, r, http.StatusOK, taskNew{Selected: q.Get("board"), Saved: q.Get("saved")})
}

func (s *Server) renderTaskCapture(w http.ResponseWriter, r *http.Request, status int, data taskNew) {
	bs, err := s.Tasks.Boards(s.actor(r))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, "", nil)
	saved := data.Saved
	data.Saved, data.Due = "", dueDates(time.Now())
	for _, b := range bs {
		data.Boards = append(data.Boards, boardTab{ID: b.ID, Name: boardName(v, b), Current: b.ID == data.Selected})
		if b.ID == saved {
			data.Saved = boardName(v, b)
		}
	}
	if !slices.ContainsFunc(data.Boards, func(t boardTab) bool { return t.Current }) {
		data.Boards[0].Current = true
	}
	v.Title, v.Data, v.Here = v.T("tasks.new"), data, tasksHere
	s.render(w, r, status, "task_new", v)
}

func newFromForm(r *http.Request) tasks.New {
	due := r.PostFormValue("due")
	if due == "date" {
		due = r.PostFormValue("due_date")
	}
	return tasks.New{ID: r.PostFormValue("id"), Board: r.PostFormValue("board"), Title: r.PostFormValue("title"),
		Urgent: r.PostFormValue("urgent") != "", Important: r.PostFormValue("important") != "", Due: due}
}

// createTask stores a captured task; the outbox of the capture page sends the
// same form with an id and asks for JSON.
func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	n := newFromForm(r)
	t, created, err := s.Tasks.Create(s.actor(r), n)
	switch {
	case errors.Is(err, tasks.ErrInvalid), errors.Is(err, tasks.ErrBadID), errors.Is(err, tasks.ErrFull):
		s.captureFailed(w, r, n, err)
	case err != nil:
		s.serverError(w, r, err)
	case wantsJSON(r):
		status, word := http.StatusCreated, "created"
		if !created {
			status, word = http.StatusOK, "exists"
		}
		writeJSON(w, status, map[string]string{"id": t.Meta.ID, "board": t.Board, "status": word})
	default:
		q := url.Values{"board": {t.Board}, "saved": {t.Board}}
		http.Redirect(w, r, "/tasks/new?"+q.Encode(), http.StatusSeeOther)
	}
}

func (s *Server) captureFailed(w http.ResponseWriter, r *http.Request, n tasks.New, err error) {
	code, key := "invalid", "tasks.invalid"
	if errors.Is(err, tasks.ErrFull) {
		code, key = "full", "tasks.full"
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": code})
		return
	}
	v := s.newView(r, "", nil)
	s.renderTaskCapture(w, r, http.StatusBadRequest, taskNew{Selected: n.Board, Title: n.Title, Error: v.T(key)})
}
