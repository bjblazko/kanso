// SPDX-License-Identifier: Apache-2.0

// Package users keeps a profile file for everyone who has signed in.
package users

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// User is a person's profile.
type User struct {
	Sub       string    `yaml:"sub"`
	Name      string    `yaml:"name"`
	Email     string    `yaml:"email"`
	Lang      string    `yaml:"lang,omitempty"`
	LastLogin time.Time `yaml:"last_login"`
}

var authorUnsafe = strings.NewReplacer(`"`, "", "<", "", ">", "", "\n", " ", "\r", " ", "\x00", "")

// Author is how the user appears in the history; characters Git does not
// accept in an author are dropped, the display name stays as it is.
func (u User) Author() store.Author {
	name := strings.Join(strings.Fields(authorUnsafe.Replace(u.Name)), " ")
	email := authorUnsafe.Replace(u.Email)
	if name == "" {
		name = u.Sub
	}
	return store.Author{Name: name, Email: email}
}

const dir = "users"

var safeSub = regexp.MustCompile(`^[A-Za-z0-9._@-]{1,128}$`)

// Store holds all profiles in memory and on disk.
type Store struct {
	dataDir string
	repo    *store.Repo
	mu      sync.RWMutex
	users   map[string]User
}

// NewStore loads every profile below dataDir/users.
func NewStore(dataDir string, repo *store.Repo) (*Store, error) {
	s := &Store{dataDir: dataDir, repo: repo, users: map[string]User{}}
	files, err := filepath.Glob(filepath.Join(dataDir, dir, "*.yaml"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		var u User
		if err := store.ReadYAML(f, &u); err != nil {
			return nil, err
		}
		s.users[u.Sub] = u
	}
	return s, repo.Ignore(avatarDir, "Profilbilder vom Verlauf ausgenommen")
}

// Upsert stores a signed-in user, keeping their chosen language.
func (s *Store) Upsert(u User) (User, error) {
	if !validSub(u.Sub) {
		return User{}, fmt.Errorf("unsafe subject %q", u.Sub)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.users[u.Sub]; ok && u.Lang == "" {
		u.Lang = old.Lang
	}
	return u, s.save(u, "Profil aktualisiert")
}

func validSub(sub string) bool {
	return safeSub.MatchString(sub) && !strings.Contains(sub, "..")
}

// save writes and commits u; the caller holds s.mu.
func (s *Store) save(u User, msg string) error {
	rel := filepath.Join(dir, u.Sub+".yaml")
	err := s.repo.Locked(func() error {
		if err := store.WriteYAML(filepath.Join(s.dataDir, rel), u); err != nil {
			return err
		}
		return s.repo.Commit(u.Author(), msg, rel)
	})
	if err == nil {
		s.users[u.Sub] = u
	}
	return err
}

// Get returns the profile of sub.
func (s *Store) Get(sub string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[sub]
	return u, ok
}

// List returns all profiles sorted by name.
func (s *Store) List() []User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	list := make([]User, 0, len(s.users))
	for _, u := range s.users {
		list = append(list, u)
	}
	slices.SortFunc(list, func(a, b User) int { return strings.Compare(a.Name, b.Name) })
	return list
}

// Languages the interface supports.
var Languages = []string{"de", "en"}

// SetLang stores the interface language of sub.
func (s *Store) SetLang(sub, lang string) error {
	if !slices.Contains(Languages, lang) {
		return fmt.Errorf("unsupported language %q", lang)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[sub]
	if !ok {
		return errors.New("unknown user")
	}
	u.Lang = lang
	return s.save(u, "Sprache geändert")
}
