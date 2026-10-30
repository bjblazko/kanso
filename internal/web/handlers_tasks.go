// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/tasks"
)

const tasksHere = "/tasks" // view.Here for the tasks area

// boardTab is one entry of the board switcher.
type boardTab struct {
	ID, Name string
	Current  bool
}

// card is one task as the board shows it.
type card struct {
	Task tasks.Task
	Due  tasks.DueState
	// DueText is "überfällig", "heute", "morgen" or the date; "" without one.
	DueText string
	// DueClass is "overdue" (red) or "soon" (today, tomorrow: yellow); "" otherwise.
	DueClass string
	// Claimed names who took the task, "dir" for the viewer; "" when nobody.
	Claimed string
}

// column is a Kanban column or a matrix quadrant.
type column struct {
	Key, Label, Hint string
	Cards            []card
}

type boardPage struct {
	Board      tasks.Board
	Name       string
	Tabs       []boardTab
	View       string // "kanban" or "matrix"
	Columns    []column
	Empty      bool
	HiddenDone bool // Kanban hides done tasks older than 14 days
	AllDone    bool
}

func (s *Server) actor(r *http.Request) tasks.Actor {
	u, _ := auth.UserFrom(r.Context())
	return tasks.Actor{Sub: u.Sub, Author: u.Author(), Groups: s.groups()}
}

func boardName(v view, b tasks.Board) string {
	if b.Private() {
		return v.T("tasks.my_board")
	}
	return b.Name
}

// tasksHome shows the person's own board.
func (s *Server) tasksHome(w http.ResponseWriter, r *http.Request) {
	bs, err := s.Tasks.Boards(s.actor(r))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.renderBoard(w, r, bs, bs[0])
}

func (s *Server) showBoard(w http.ResponseWriter, r *http.Request) {
	bs, err := s.Tasks.Boards(s.actor(r))
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	i := slices.IndexFunc(bs, func(b tasks.Board) bool { return b.ID == r.PathValue("board") })
	if i < 0 {
		s.notFound(w, r)
		return
	}
	s.renderBoard(w, r, bs, bs[i])
}

func (s *Server) renderBoard(w http.ResponseWriter, r *http.Request, bs []tasks.Board, b tasks.Board) {
	ts, err := s.Tasks.Tasks(s.actor(r), b.ID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, "", nil)
	q := r.URL.Query()
	p := boardPage{Board: b, Name: boardName(v, b), View: "kanban", Empty: len(ts) == 0, AllDone: q.Get("done") == "all"}
	for _, x := range bs {
		p.Tabs = append(p.Tabs, boardTab{ID: x.ID, Name: boardName(v, x), Current: x.ID == b.ID})
	}
	if q.Get("view") == "matrix" {
		p.View, p.Columns = "matrix", s.matrixColumns(v, ts)
	} else {
		p.Columns, p.HiddenDone = s.kanbanColumns(v, ts, p.AllDone)
	}
	v.Title, v.Data, v.Here = p.Name, p, tasksHere
	v.Audience = ptr(s.boardAudience(v, b))
	s.render(w, r, http.StatusOK, "tasks_board", v)
}

func (s *Server) kanbanColumns(v view, ts []tasks.Task, allDone bool) ([]column, bool) {
	now := time.Now()
	k := tasks.Kanban(ts, now, allDone)
	var cols []column
	for _, st := range tasks.Statuses {
		cols = append(cols, column{Key: string(st), Label: v.T("tasks.status." + string(st)), Cards: s.cards(v, k[st], now)})
	}
	return cols, len(tasks.Kanban(ts, now, true)[tasks.Done]) > len(k[tasks.Done])
}

var quadrantKeys = map[tasks.Quadrant]string{tasks.DoNow: "do_now", tasks.Plan: "plan", tasks.Quick: "quick", tasks.Later: "later"}

func (s *Server) matrixColumns(v view, ts []tasks.Task) []column {
	m := tasks.Matrix(ts)
	var cols []column
	for _, q := range tasks.Quadrants {
		key := quadrantKeys[q]
		cols = append(cols, column{Key: key, Label: v.T("tasks.quadrant." + key), Hint: v.T("tasks.quadrant_hint." + key),
			Cards: s.cards(v, m[q], time.Now())})
	}
	return cols
}

func (s *Server) cards(v view, ts []tasks.Task, now time.Time) []card {
	out := make([]card, 0, len(ts))
	for _, t := range ts {
		c := card{Task: t, Due: tasks.DueOf(t.Meta.Due, now)}
		c.DueText, c.DueClass = dueText(v, t.Meta.Due, c.Due), dueClass[c.Due]
		switch t.Meta.Claimed {
		case "":
		case v.User.Sub:
			c.Claimed = v.T("tasks.claimed_you")
		default:
			c.Claimed = v.T("tasks.claimed_by", s.userName(t.Meta.Claimed))
		}
		out = append(out, c)
	}
	return out
}

var dueClass = map[tasks.DueState]string{tasks.DueOverdue: "overdue", tasks.DueToday: "soon", tasks.DueTomorrow: "soon"}

// cardArgs hands a card and the page's view to the card template.
type cardArgs struct {
	Card card
	V    view
}

func dueText(v view, due string, st tasks.DueState) string {
	switch st {
	case tasks.DueOverdue:
		return v.T("tasks.overdue")
	case tasks.DueToday:
		return v.T("tasks.today")
	case tasks.DueTomorrow:
		return v.T("tasks.tomorrow")
	case tasks.DueLater:
		d, _ := time.Parse("2006-01-02", due)
		return v.T("tasks.due_on", d.Format("02.01.2006"))
	}
	return ""
}

// boardAudience names who sees a board: only you, or the group's members.
func (s *Server) boardAudience(v view, b tasks.Board) audience {
	if b.Private() {
		return onlyYou(v)
	}
	var you bool
	var names []string
	for _, sub := range s.groups()[b.Owner.Group] {
		if sub == v.User.Sub {
			you = true
		} else {
			names = append(names, s.userName(sub))
		}
	}
	slices.Sort(names)
	if you {
		names = append([]string{v.T("visibility.you")}, names...)
	}
	return audience{Text: v.T("visibility.board", strings.Join(names, ", "))}
}
