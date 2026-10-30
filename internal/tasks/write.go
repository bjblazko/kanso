// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

func (s *Store) taskRel(boardID, taskID string) string {
	return filepath.Join(Dir, boardID, tasksDir, taskID+".md")
}

func (s *Store) readTask(boardID, taskID string) (Task, error) {
	if !idPattern.MatchString(taskID) {
		return Task{}, ErrNotFound
	}
	d, err := store.ReadDoc[Meta](filepath.Join(s.dataDir, s.taskRel(boardID, taskID)))
	if errors.Is(err, fs.ErrNotExist) {
		return Task{}, ErrNotFound
	}
	return Task{Board: boardID, Meta: d.Meta, Body: d.Body}, err
}

func (s *Store) writeTask(t Task) error {
	return store.WriteDoc(filepath.Join(s.dataDir, s.taskRel(t.Board, t.Meta.ID)), store.Doc[Meta]{Meta: t.Meta, Body: t.Body})
}

// Tasks returns every task of a board the person sees.
func (s *Store) Tasks(a Actor, boardID string) ([]Task, error) {
	if _, err := s.Board(a, boardID); err != nil {
		return nil, err
	}
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, boardID, tasksDir, "*.md"))
	out := make([]Task, 0, len(files))
	for _, f := range files {
		if t, err := s.readTask(boardID, strings.TrimSuffix(filepath.Base(f), ".md")); err == nil {
			out = append(out, t)
		}
	}
	return out, nil
}

// Get returns one task of a board the person sees.
func (s *Store) Get(a Actor, boardID, taskID string) (Task, error) {
	if _, err := s.Board(a, boardID); err != nil {
		return Task{}, err
	}
	return s.readTask(boardID, taskID)
}

// Create stores a new task in Offen. An id that already exists on a visible
// board returns that task with created == false, so a device may send the
// same task twice, even after someone moved it.
func (s *Store) Create(a Actor, n New) (t Task, created bool, err error) {
	if n.ID == "" {
		n.ID = store.NewID()
	}
	if !idPattern.MatchString(n.ID) {
		return Task{}, false, ErrBadID
	}
	if err := checkTask(n.Title, n.Body, n.Due, ""); err != nil {
		return Task{}, false, err
	}
	board, err := s.target(a, n.Board)
	if err != nil {
		return Task{}, false, err
	}
	err = s.repo.Locked(func() error {
		var err error
		t, created, err = s.createLocked(a, board, n)
		return err
	})
	return t, created, err
}

// target is the visible board id, or the person's own board.
func (s *Store) target(a Actor, id string) (string, error) {
	if b, err := s.Board(a, id); err == nil {
		return b.ID, nil
	}
	own, err := s.ownBoard(a)
	return own.ID, err
}

func (s *Store) createLocked(a Actor, board string, n New) (Task, bool, error) {
	if existing, err := s.findTask(a, n.ID); err == nil || !errors.Is(err, ErrNotFound) {
		return existing, false, err
	}
	if count, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, board, tasksDir, "*.md")); len(count) >= maxTasks {
		return Task{}, false, ErrFull
	}
	now := s.stamp(s.now().UTC().Add(-1))
	t := Task{Board: board, Body: n.Body, Meta: Meta{ID: n.ID, Title: strings.TrimSpace(n.Title), Status: Open,
		Urgent: n.Urgent, Important: n.Important, Due: n.Due, Note: n.Note, Author: a.Sub, Created: now, Updated: now}}
	if err := s.writeTask(t); err != nil {
		return Task{}, false, err
	}
	return t, true, s.repo.Commit(a.Author, commitMsg(t, "angelegt"), s.taskRel(board, t.Meta.ID))
}

// findTask looks for id on every board; one the person does not see is a bad id.
func (s *Store) findTask(a Actor, id string) (Task, error) {
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*", tasksDir, id+".md"))
	if len(files) == 0 {
		return Task{}, ErrNotFound
	}
	board := filepath.Base(filepath.Dir(filepath.Dir(files[0])))
	if _, err := s.Board(a, board); err != nil {
		return Task{}, ErrBadID
	}
	return s.readTask(board, id)
}

func commitMsg(t Task, what string) string { return "tasks: " + t.Meta.Title + " → " + what }

// change loads a task, lets fn change it and saves it as one commit.
func (s *Store) change(a Actor, boardID, taskID, what string, fn func(*Task) error) (Task, error) {
	var out Task
	err := s.repo.Locked(func() error {
		t, err := s.Get(a, boardID, taskID)
		if err != nil {
			return err
		}
		if err := fn(&t); err != nil {
			return err
		}
		t.Meta.Updated = s.stamp(t.Meta.Updated)
		if err := s.writeTask(t); err != nil {
			return err
		}
		out = t
		return s.repo.Commit(a.Author, commitMsg(t, what), s.taskRel(boardID, taskID))
	})
	return out, err
}

// setStatus moves t to st and keeps done_at in step.
func (s *Store) setStatus(t *Task, st Status) error {
	if !st.valid() {
		return ErrInvalid
	}
	switch {
	case st == Done && t.Meta.Status != Done:
		now := s.now().UTC().Truncate(time.Millisecond)
		t.Meta.DoneAt = &now
	case st != Done:
		t.Meta.DoneAt = nil
	}
	t.Meta.Status = st
	return nil
}

// Update saves the edit form; version is the Updated time the form was read at.
func (s *Store) Update(a Actor, boardID, taskID string, e Edit, version time.Time) (Task, error) {
	return s.change(a, boardID, taskID, "geändert", func(t *Task) error {
		if !t.Meta.Updated.Equal(version) {
			return ErrConflict
		}
		if err := checkTask(e.Title, e.Body, e.Due, e.WaitingFor); err != nil {
			return err
		}
		t.Meta.Title, t.Body, t.Meta.Due = strings.TrimSpace(e.Title), e.Body, e.Due
		t.Meta.Urgent, t.Meta.Important, t.Meta.WaitingFor = e.Urgent, e.Important, strings.TrimSpace(e.WaitingFor)
		return s.setStatus(t, e.Status)
	})
}

// SetStatus moves a task to another column.
func (s *Store) SetStatus(a Actor, boardID, taskID string, st Status) (Task, error) {
	return s.change(a, boardID, taskID, string(st), func(t *Task) error { return s.setStatus(t, st) })
}

// SetQuadrant moves a task to another quadrant.
func (s *Store) SetQuadrant(a Actor, boardID, taskID string, urgent, important bool) (Task, error) {
	return s.change(a, boardID, taskID, "Quadrant", func(t *Task) error {
		t.Meta.Urgent, t.Meta.Important = urgent, important
		return nil
	})
}

// Claim takes a task of a group board (claim) or gives it back.
func (s *Store) Claim(a Actor, boardID, taskID string, claim bool) (Task, error) {
	b, err := s.Board(a, boardID)
	if err != nil {
		return Task{}, err
	}
	if b.Private() {
		return Task{}, ErrPrivate
	}
	what, who := "freigegeben", ""
	if claim {
		what, who = "übernommen", a.Sub
	}
	return s.change(a, boardID, taskID, what, func(t *Task) error {
		t.Meta.Claimed = who
		return nil
	})
}

// Delete removes a task; the history keeps it.
func (s *Store) Delete(a Actor, boardID, taskID string) error {
	return s.repo.Locked(func() error {
		t, err := s.Get(a, boardID, taskID)
		if err != nil {
			return err
		}
		rel := s.taskRel(boardID, taskID)
		if err := os.Remove(filepath.Join(s.dataDir, rel)); err != nil {
			return err
		}
		return s.repo.Commit(a.Author, commitMsg(t, "gelöscht"), rel)
	})
}

// Move puts a task on another board the person sees.
func (s *Store) Move(a Actor, fromBoard, taskID, toBoard string) (Task, error) {
	to, err := s.Board(a, toBoard)
	if err != nil {
		return Task{}, err
	}
	var out Task
	err = s.repo.Locked(func() error {
		t, err := s.Get(a, fromBoard, taskID)
		if err != nil || fromBoard == to.ID {
			out = t
			return err
		}
		t.Board = to.ID
		out = t
		return s.repo.Move(a.Author, s.taskRel(fromBoard, taskID), s.taskRel(to.ID, taskID), commitMsg(t, "verschoben"))
	})
	return out, err
}
