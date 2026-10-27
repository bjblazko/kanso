// SPDX-License-Identifier: Apache-2.0

// Package todos keeps simple todo lists: flat, shareable, one level of
// sub-items. Lists live in the data directory but outside the Git history and
// change through operations, so a device can tick items offline and send its
// changes later.
package todos

import (
	"cmp"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds all lists, relative to the data directory.
const Dir = "todos"

var (
	// ErrNotFound means the list does not exist or the person may not see it.
	ErrNotFound = errors.New("todo list not found")
	// ErrBadShare means a share names neither exactly one person nor one group.
	ErrBadShare = errors.New("invalid share")
)

var idPattern = regexp.MustCompile(`^[a-z2-7]{12}$`)

// Stamps are the device times (ms) of the last change of each field; the
// later change wins.
type Stamps struct {
	Text int64 `json:"text"`
	Done int64 `json:"done"`
	Sub  int64 `json:"sub"`
}

// Item is one line of a list. Sub marks a sub-item of the main item above it.
type Item struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	Done bool   `json:"done"`
	Sub  bool   `json:"sub"`
	At   Stamps `json:"at"`
}

// Share gives one person or one group access to a list.
type Share struct {
	User  string `json:"user,omitempty"`
	Group string `json:"group,omitempty"`
}

// List is one todo list.
type List struct {
	ID      string    `json:"id"`
	Name    string    `json:"name"`
	Owner   string    `json:"owner"`
	Shared  []Share   `json:"shared"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Items   []Item    `json:"items"`
	// Deleted remembers removed item ids (server time, ms), so a late change
	// from another device does not bring them back.
	Deleted map[string]int64 `json:"deleted,omitempty"`
}

// Store reads and writes lists; this process is the only writer.
type Store struct {
	dataDir string
	repo    *store.Repo
	mu      sync.Mutex
	now     func() time.Time
}

// NewStore opens the lists below dataDir and keeps them out of Git.
func NewStore(dataDir string, repo *store.Repo) (*Store, error) {
	s := &Store{dataDir: dataDir, repo: repo, now: time.Now}
	if err := os.MkdirAll(filepath.Join(dataDir, Dir), 0o750); err != nil {
		return nil, err
	}
	return s, repo.Ignore(Dir, "Todo-Listen vom Verlauf ausgenommen")
}

func (s *Store) path(id string) string { return filepath.Join(s.dataDir, Dir, id+".json") }

func (s *Store) read(id string) (List, error) {
	if !idPattern.MatchString(id) {
		return List{}, ErrNotFound
	}
	b, err := os.ReadFile(s.path(id))
	if errors.Is(err, fs.ErrNotExist) {
		return List{}, ErrNotFound
	} else if err != nil {
		return List{}, err
	}
	var l List
	return l, json.Unmarshal(b, &l)
}

// write stores l, dropping tombstones older than tombstoneAge.
func (s *Store) write(l List) error {
	limit := s.now().Add(-tombstoneAge).UnixMilli()
	for id, at := range l.Deleted {
		if at < limit {
			delete(l.Deleted, id)
		}
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteAtomic(s.path(l.ID), append(b, '\n'))
}

// canSee reports whether sub owns l or l is shared with sub or a group of sub.
func canSee(l List, sub string, g access.Groups) bool {
	if l.Owner == sub {
		return true
	}
	return slices.ContainsFunc(l.Shared, func(sh Share) bool {
		return sh.User == sub || (sh.Group != "" && slices.Contains(g[sh.Group], sub))
	})
}

// Get returns list id if sub may see it.
func (s *Store) Get(sub, id string, g access.Groups) (List, error) {
	l, err := s.read(id)
	if err != nil || !canSee(l, sub, g) {
		return List{}, ErrNotFound
	}
	return l, nil
}

// Visible returns the lists sub may see, most recently changed first.
func (s *Store) Visible(sub string, g access.Groups) []List {
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*.json"))
	var out []List
	for _, f := range files {
		if l, err := s.Get(sub, strings.TrimSuffix(filepath.Base(f), ".json"), g); err == nil {
			out = append(out, l)
		}
	}
	slices.SortFunc(out, func(a, b List) int {
		return cmp.Or(b.Updated.Compare(a.Updated), strings.Compare(a.ID, b.ID))
	})
	return out
}

// owned returns list id if owner owns it; the caller holds s.mu.
func (s *Store) owned(owner, id string) (List, error) {
	l, err := s.read(id)
	if err != nil || l.Owner != owner {
		return List{}, ErrNotFound
	}
	return l, nil
}

// SetShares replaces who else may see the list; only the owner may.
func (s *Store) SetShares(owner, id string, shares []Share) error {
	for _, sh := range shares {
		if (sh.User == "") == (sh.Group == "") {
			return ErrBadShare
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	l, err := s.owned(owner, id)
	if err != nil {
		return err
	}
	l.Shared = shares
	return s.write(l)
}

// Delete removes a list for good; only the owner may.
func (s *Store) Delete(owner, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.owned(owner, id); err != nil {
		return err
	}
	return os.Remove(s.path(id))
}

// Done counts the ticked items and all items of l.
func Done(l List) (done, total int) {
	for _, it := range l.Items {
		if it.Done {
			done++
		}
	}
	return done, len(l.Items)
}
