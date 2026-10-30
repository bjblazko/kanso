// SPDX-License-Identifier: Apache-2.0

// Package notes keeps each person's private notes. Notes are scratch paper:
// they live in the data directory but outside the Git history, so deleting a
// note removes it for good.
package notes

import (
	"cmp"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds all notes, relative to the data directory.
const Dir = "notes"

const (
	noteFile       = "index.md"
	attachmentsDir = "attachments"
)

var (
	// ErrNotFound means the note does not exist for this owner.
	ErrNotFound = errors.New("note not found")
	// ErrBadID means an owner or note id that cannot name a folder.
	ErrBadID = errors.New("invalid note id")
	// ErrConflict means the note changed since it was opened.
	ErrConflict = errors.New("note changed meanwhile")
	// ErrTooLarge means an upload exceeded the size limit.
	ErrTooLarge = store.ErrTooLarge
)

var (
	idPattern    = regexp.MustCompile(`^[a-z2-7]{12}$`)
	ownerPattern = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,128}$`)
)

// Meta is a note's frontmatter.
type Meta struct {
	ID      string    `yaml:"id"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
	// Task names the task made from this note, "<board-id>/<task-id>".
	Task string `yaml:"task,omitempty"`
}

// Note is one note of one person.
type Note struct {
	Owner       string
	Meta        Meta
	Body        string
	Attachments []string
}

// Upload is a file sent along with a new note.
type Upload struct {
	Name string
	R    io.Reader
}

// Store reads and writes notes; this process is the only writer.
type Store struct {
	dataDir string
	repo    *store.Repo
	mu      sync.Mutex
	now     func() time.Time

	// OnChange is called after a note was written or removed.
	OnChange func(n Note, deleted bool)
}

// NewStore opens the notes below dataDir and keeps them out of Git.
func NewStore(dataDir string, repo *store.Repo) (*Store, error) {
	s := &Store{dataDir: dataDir, repo: repo, now: time.Now}
	return s, repo.Ignore(Dir, "Notizen vom Verlauf ausgenommen")
}

// folder returns the folder of note id, refusing ids that are not ours.
func (s *Store) folder(owner, id string) (string, error) {
	if !ownerPattern.MatchString(owner) || owner == "." || owner == ".." || !idPattern.MatchString(id) {
		return "", ErrBadID
	}
	return filepath.Join(s.dataDir, Dir, owner, id), nil
}

// Get returns one note of owner.
func (s *Store) Get(owner, id string) (Note, error) {
	dir, err := s.folder(owner, id)
	if err != nil {
		return Note{}, ErrNotFound
	}
	return read(owner, dir)
}

func read(owner, dir string) (Note, error) {
	d, err := store.ReadDoc[Meta](filepath.Join(dir, noteFile))
	if errors.Is(err, fs.ErrNotExist) {
		return Note{}, ErrNotFound
	} else if err != nil {
		return Note{}, err
	}
	return Note{Owner: owner, Meta: d.Meta, Body: d.Body, Attachments: listAttachments(dir)}, nil
}

// List returns owner's notes, newest first.
func (s *Store) List(owner string) []Note {
	if !ownerPattern.MatchString(owner) || owner == "." || owner == ".." {
		return nil
	}
	return s.glob(filepath.Join(s.dataDir, Dir, owner, "*", noteFile))
}

// All returns every note of every owner, for the search index.
func (s *Store) All() []Note {
	return s.glob(filepath.Join(s.dataDir, Dir, "*", "*", noteFile))
}

func (s *Store) glob(pattern string) []Note {
	files, _ := filepath.Glob(pattern)
	var out []Note
	for _, f := range files {
		dir := filepath.Dir(f)
		if n, err := read(filepath.Base(filepath.Dir(dir)), dir); err == nil && idPattern.MatchString(n.Meta.ID) {
			out = append(out, n)
		}
	}
	slices.SortFunc(out, func(a, b Note) int {
		return cmp.Or(b.Meta.Created.Compare(a.Meta.Created), strings.Compare(a.Meta.ID, b.Meta.ID))
	})
	return out
}

// Delete removes a note and its attachments for good.
func (s *Store) Delete(owner, id string) error {
	return s.MoveOut(owner, id, os.RemoveAll)
}

// MoveOut hands the note's folder to move (which must take it away, e.g.
// into the wiki) and then reports the note as removed.
func (s *Store) MoveOut(owner, id string, move func(dir string) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.Get(owner, id)
	if err != nil {
		return err
	}
	dir, _ := s.folder(owner, id)
	if err := move(dir); err != nil {
		return err
	}
	s.changed(n, true)
	return nil
}

// Dir returns the absolute folder of an existing note.
func (s *Store) Dir(owner, id string) (string, error) {
	if _, err := s.Get(owner, id); err != nil {
		return "", err
	}
	return s.folder(owner, id)
}

func (s *Store) changed(n Note, deleted bool) {
	if s.OnChange != nil {
		s.OnChange(n, deleted)
	}
}

// stamp returns the time for Updated, always after previous so a save is
// never mistaken for the version it replaced.
func (s *Store) stamp(previous time.Time) time.Time {
	t := s.now().UTC().Truncate(time.Millisecond)
	if !t.After(previous) {
		t = previous.Add(time.Millisecond)
	}
	return t
}
