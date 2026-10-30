// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/store"
)

func readBoardFile(path string, b *Board) error { return store.ReadYAML(path, b) }

func (s *Store) boardRel(id string) string { return filepath.Join(Dir, id) }

// readBoard loads board id; unknown or malformed ids are ErrNotFound.
func (s *Store) readBoard(id string) (Board, error) {
	if !idPattern.MatchString(id) {
		return Board{}, ErrNotFound
	}
	var b Board
	err := readBoardFile(filepath.Join(s.dataDir, Dir, id, boardFile), &b)
	if errors.Is(err, fs.ErrNotExist) {
		return Board{}, ErrNotFound
	}
	b.ID = id
	return b, err
}

// allBoards reads every board; unreadable ones are skipped.
func (s *Store) allBoards() []Board {
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*", boardFile))
	var out []Board
	for _, f := range files {
		if b, err := s.readBoard(filepath.Base(filepath.Dir(f))); err == nil {
			out = append(out, b)
		}
	}
	return out
}

// Boards returns the person's own board (made on first use), then the
// group boards they see, by name.
func (s *Store) Boards(a Actor) ([]Board, error) {
	own, err := s.ownBoard(a)
	if err != nil {
		return nil, err
	}
	out := []Board{own}
	for _, b := range s.GroupBoards() {
		if a.sees(b) {
			out = append(out, b)
		}
	}
	return out, nil
}

// Board returns board id if the person sees it.
func (s *Store) Board(a Actor, id string) (Board, error) {
	b, err := s.readBoard(id)
	if err != nil || !a.sees(b) {
		return Board{}, ErrNotFound
	}
	return b, nil
}

// ownBoard finds the person's private board or creates it, once.
func (s *Store) ownBoard(a Actor) (Board, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, b := range s.allBoards() {
		if b.Owner.User == a.Sub {
			return b, nil
		}
	}
	b := Board{ID: store.NewID(), Owner: Owner{User: a.Sub}, Created: s.now().UTC().Truncate(time.Millisecond)}
	return b, s.saveBoard(a.Author, b, "tasks: eigenes Board angelegt")
}

func (s *Store) saveBoard(author store.Author, b Board, msg string) error {
	return s.repo.Locked(func() error {
		if err := store.WriteYAML(filepath.Join(s.dataDir, Dir, b.ID, boardFile), b); err != nil {
			return err
		}
		return s.repo.Commit(author, msg, s.boardRel(b.ID))
	})
}

// GroupBoards returns every group board by name; for administration.
func (s *Store) GroupBoards() []Board {
	var out []Board
	for _, b := range s.allBoards() {
		if !b.Private() {
			out = append(out, b)
		}
	}
	slices.SortFunc(out, func(x, y Board) int { return cmp.Compare(x.Name, y.Name) })
	return out
}

func boardName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxBoardName {
		return "", ErrInvalid
	}
	return name, nil
}

// CreateGroupBoard makes a board for the members of group.
func (s *Store) CreateGroupBoard(author store.Author, name, group string) (Board, error) {
	name, err := boardName(name)
	if err != nil || strings.TrimSpace(group) == "" {
		return Board{}, ErrInvalid
	}
	b := Board{ID: store.NewID(), Name: name, Owner: Owner{Group: group}, Created: s.now().UTC().Truncate(time.Millisecond)}
	return b, s.saveBoard(author, b, "tasks: Board „"+name+"“ angelegt")
}

// groupBoard loads a group board; private boards are not found.
func (s *Store) groupBoard(id string) (Board, error) {
	b, err := s.readBoard(id)
	if err != nil || b.Private() {
		return Board{}, ErrNotFound
	}
	return b, nil
}

// RenameBoard renames a group board.
func (s *Store) RenameBoard(author store.Author, id, name string) error {
	b, err := s.groupBoard(id)
	if err != nil {
		return err
	}
	if b.Name, err = boardName(name); err != nil {
		return err
	}
	return s.saveBoard(author, b, "tasks: Board „"+b.Name+"“ umbenannt")
}

// DeleteBoard removes a group board with all its tasks; Git keeps them.
func (s *Store) DeleteBoard(author store.Author, id string) error {
	b, err := s.groupBoard(id)
	if err != nil {
		return err
	}
	return s.repo.Locked(func() error {
		if err := os.RemoveAll(filepath.Join(s.dataDir, Dir, b.ID)); err != nil {
			return err
		}
		return s.repo.Commit(author, "tasks: Board „"+b.Name+"“ gelöscht", s.boardRel(b.ID))
	})
}
