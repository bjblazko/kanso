// SPDX-License-Identifier: Apache-2.0

package selfcare

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds each person's own first-aid ideas, relative to the data
// directory and, like notes, outside the Git history: deleting one removes it.
const Dir = "selfcare"

const (
	maxOwn      = 50
	maxOwnTitle = 80
	maxOwnBody  = 300
)

var (
	// ErrInvalid means a blank or too long idea, or too many.
	ErrInvalid = errors.New("invalid idea")
	safeSub    = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,128}$`)
)

// Own is one person's idea for first aid.
type Own struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	Body  string `yaml:"body,omitempty"`
	// Kind is "calm" (first aid), "lift" (between two things) or "both".
	Kind string `yaml:"kind,omitempty"`
}

// Kinds an idea can be, in the order the form offers them.
var Kinds = []string{"calm", "lift", "both"}

// Tip makes the idea a step like the built-in ones.
func (o Own) Tip() Tip {
	return Tip{ID: "own-" + o.ID, Title: Text{De: o.Title}, Body: Text{De: o.Body},
		Gentle: o.Kind != "lift", Lift: o.Kind == "lift" || o.Kind == "both"}
}

// Store keeps the ideas, one file per person.
type Store struct {
	dir string
	mu  sync.Mutex
}

// NewStore opens the ideas below dataDir and keeps them out of Git.
func NewStore(dataDir string, repo *store.Repo) (*Store, error) {
	return &Store{dir: filepath.Join(dataDir, Dir)}, repo.Ignore(Dir, "Eigene Selfcare-Ideen vom Verlauf ausgenommen")
}

func (s *Store) path(sub string) (string, error) {
	if !safeSub.MatchString(sub) {
		return "", ErrInvalid
	}
	return filepath.Join(s.dir, sub+".yaml"), nil
}

// Own lists a person's ideas, oldest first.
func (s *Store) Own(sub string) []Own {
	p, err := s.path(sub)
	if err != nil {
		return nil
	}
	var out []Own
	if err := store.ReadYAML(p, &out); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return out
}

// Add keeps a new idea for sub; kind is one of Kinds.
func (s *Store) Add(sub, title, body, kind string) error {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	if title == "" || utf8.RuneCountInString(title) > maxOwnTitle || utf8.RuneCountInString(body) > maxOwnBody || !slices.Contains(Kinds, kind) {
		return ErrInvalid
	}
	return s.change(sub, func(own []Own) ([]Own, error) {
		if len(own) >= maxOwn {
			return nil, ErrInvalid
		}
		return append(own, Own{ID: store.NewID(), Title: title, Body: body, Kind: kind}), nil
	})
}

// Delete removes one of sub's ideas; an unknown id changes nothing.
func (s *Store) Delete(sub, id string) error {
	return s.change(sub, func(own []Own) ([]Own, error) {
		return slices.DeleteFunc(own, func(o Own) bool { return o.ID == id }), nil
	})
}

func (s *Store) change(sub string, fn func([]Own) ([]Own, error)) error {
	p, err := s.path(sub)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	own, err := fn(s.Own(sub))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o750); err != nil {
		return err
	}
	return store.WriteYAML(p, own)
}

// PickFrom returns a random tip from pool other than the one shown before.
func PickFrom(pool []Tip, n func(int) int, before string) Tip {
	rest := slices.DeleteFunc(slices.Clone(pool), func(t Tip) bool { return t.ID == before })
	if len(rest) == 0 {
		return pool[0]
	}
	return rest[n(len(rest))]
}
