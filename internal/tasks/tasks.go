// SPDX-License-Identifier: Apache-2.0

// Package tasks keeps boards of tasks: one private board per person and
// boards shared by a group, each task one Markdown file in the Git history.
package tasks

import (
	"errors"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds all boards, relative to the data directory.
const Dir = "boards"

const (
	boardFile    = "board.yaml"
	tasksDir     = "tasks"
	maxTitle     = 200
	maxBody      = 100 << 10
	maxWaiting   = 120
	maxTasks     = 2000
	maxBoardName = 80
	dateLayout   = "2006-01-02"
)

var (
	// ErrNotFound means the board or task does not exist for this person.
	ErrNotFound = errors.New("not found")
	// ErrBadID means an id that cannot name a board or task.
	ErrBadID = errors.New("invalid id")
	// ErrInvalid means a value outside the limits: blank title, bad status or date.
	ErrInvalid = errors.New("invalid task")
	// ErrConflict means the task changed since it was opened.
	ErrConflict = errors.New("task changed meanwhile")
	// ErrPrivate means claiming a task on a private board.
	ErrPrivate = errors.New("private board")
	// ErrFull means the board holds the maximum number of tasks.
	ErrFull = errors.New("board full")
)

var idPattern = regexp.MustCompile(`^[a-z2-7]{12}$`)

// Status is the Kanban column of a task.
type Status string

// The fixed columns, in board order.
const (
	Open    Status = "open"
	Doing   Status = "doing"
	Waiting Status = "waiting"
	Done    Status = "done"
)

// Statuses lists the columns in board order.
var Statuses = []Status{Open, Doing, Waiting, Done}

func (s Status) valid() bool { return slices.Contains(Statuses, s) }

// Owner is the person or the group a board belongs to (exactly one is set).
type Owner struct {
	User  string `yaml:"user,omitempty"`
	Group string `yaml:"group,omitempty"`
}

// Board is a set of tasks with one owner.
type Board struct {
	ID      string    `yaml:"-"`
	Name    string    `yaml:"name,omitempty"`
	Owner   Owner     `yaml:"owner"`
	Created time.Time `yaml:"created"`
}

// Private reports whether the board belongs to one person.
func (b Board) Private() bool { return b.Owner.User != "" }

// Meta is a task's frontmatter.
type Meta struct {
	ID         string     `yaml:"id"`
	Title      string     `yaml:"title"`
	Status     Status     `yaml:"status"`
	Urgent     bool       `yaml:"urgent"`
	Important  bool       `yaml:"important"`
	Due        string     `yaml:"due,omitempty"`
	WaitingFor string     `yaml:"waiting_for,omitempty"`
	Claimed    string     `yaml:"claimed,omitempty"`
	Note       string     `yaml:"note,omitempty"`
	Author     string     `yaml:"author"`
	Created    time.Time  `yaml:"created"`
	Updated    time.Time  `yaml:"updated"`
	DoneAt     *time.Time `yaml:"done_at,omitempty"`
}

// Task is one task on one board.
type Task struct {
	Board string
	Meta  Meta
	Body  string
}

// Actor is the person asking, with what Git records for them and the groups
// that decide which boards they see. Admins have no extra rights here.
type Actor struct {
	Sub    string
	Author store.Author
	Groups access.Groups
}

func (a Actor) sees(b Board) bool {
	if b.Private() {
		return b.Owner.User == a.Sub
	}
	return slices.Contains(a.Groups[b.Owner.Group], a.Sub)
}

// New is a task to create. An empty ID gets a new one; an unknown or
// invisible Board means the person's own board.
type New struct {
	ID, Board, Title  string
	Urgent, Important bool
	Due, Body, Note   string
}

// Edit is everything the edit form changes.
type Edit struct {
	Title, Body       string
	Status            Status
	Urgent, Important bool
	Due, WaitingFor   string
}

// Store reads and writes boards; this process is the only writer.
type Store struct {
	dataDir string
	repo    *store.Repo
	mu      sync.Mutex // serialises finding or creating a person's own board
	now     func() time.Time
}

// NewStore opens the boards below dataDir.
func NewStore(dataDir string, repo *store.Repo) *Store {
	return &Store{dataDir: dataDir, repo: repo, now: time.Now}
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

// checkTask validates what a person may type into a task.
func checkTask(title, body, due, waiting string) error {
	t := strings.TrimSpace(title)
	if t == "" || utf8.RuneCountInString(t) > maxTitle || len(body) > maxBody ||
		utf8.RuneCountInString(waiting) > maxWaiting {
		return ErrInvalid
	}
	if due != "" {
		if _, err := time.Parse(dateLayout, due); err != nil {
			return ErrInvalid
		}
	}
	return nil
}
