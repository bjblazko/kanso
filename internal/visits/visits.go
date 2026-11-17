// SPDX-License-Identifier: Apache-2.0

// Package visits remembers when each person last opened the start page, so
// its summary can begin where they left off. The times live in the state
// directory, never in Git: a visit is not content (ADR-018).
package visits

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// Grace is how long a visit lasts: opening the start page again within
	// it keeps the same window.
	Grace = time.Hour
	// Floor is the shortest window the summary covers.
	Floor = 24 * time.Hour
)

// visit is one person's last two visits.
type visit struct {
	Previous time.Time `json:"previous"`
	Last     time.Time `json:"last"`
}

// Store keeps the visits in <stateDir>/visits.json.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore opens the visits in stateDir.
func NewStore(stateDir string) *Store { return &Store{path: filepath.Join(stateDir, "visits.json")} }

// Touch records a visit of sub at now and returns the previous visit (zero
// on the first). A visit within Grace of the last one changes nothing.
func (s *Store) Touch(sub string, now time.Time) (time.Time, error) {
	if sub == "" {
		return time.Time{}, errors.New("visits: no person")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	all := s.read()
	v := all[sub]
	if !v.Last.IsZero() && now.Sub(v.Last) <= Grace {
		return v.Previous, nil
	}
	v.Previous, v.Last = v.Last, now.UTC()
	all[sub] = v
	return v.Previous, s.write(all)
}

// Since is where a summary starts: the previous visit, but at least Floor ago.
func Since(previous, now time.Time) time.Time {
	floor := now.Add(-Floor)
	if previous.IsZero() || previous.After(floor) {
		return floor
	}
	return previous
}

// read returns the stored visits; a missing or broken file is none.
func (s *Store) read() map[string]visit {
	all := map[string]visit{}
	b, err := os.ReadFile(s.path)
	if err == nil && json.Unmarshal(b, &all) != nil {
		all = map[string]visit{}
	}
	return all
}

// write replaces the file atomically, readable by this process only.
func (s *Store) write(all map[string]visit) error {
	b, err := json.Marshal(all)
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".visits-*") // CreateTemp makes 0600
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), s.path)
}
