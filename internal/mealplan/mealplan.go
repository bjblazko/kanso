// SPDX-License-Identifier: Apache-2.0

// Package mealplan keeps cooking plans: one per group, a file per week with
// the meals of each day, in the Git history (ADR-012).
package mealplan

import (
	"cmp"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds all plans, relative to the data directory.
const Dir = "mealplans"

const (
	planFile    = "plan.yaml"
	maxName     = 80
	maxPerDay   = 20
	maxServings = 99
)

// Errors callers can tell apart.
var (
	ErrNotFound = errors.New("meal plan not found")
	ErrInvalid  = errors.New("invalid meal plan")
)

var (
	idPattern  = regexp.MustCompile(`^[a-z2-7]{12}$`)
	dayPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
)

// Plan is the cooking plan of one group.
type Plan struct {
	ID      string    `yaml:"-"`
	Name    string    `yaml:"name"`
	Group   string    `yaml:"group"`
	Created time.Time `yaml:"created"`
}

// Entry is one meal on one day. Fill and List are set once it was added to a shopping list.
type Entry struct {
	ID       string `yaml:"id"`
	Recipe   string `yaml:"recipe"`
	Title    string `yaml:"title"`
	Servings int    `yaml:"servings"`
	Fill     string `yaml:"fill,omitempty"`
	List     string `yaml:"list,omitempty"`
}

// Week holds the meals of one ISO week by day ("2006-01-02").
type Week struct {
	Key  string             `yaml:"-"`
	Days map[string][]Entry `yaml:"days"`
}

// Store reads and writes plans.
type Store struct {
	dataDir string
	repo    *store.Repo
	now     func() time.Time
}

// NewStore opens the plans below dataDir.
func NewStore(dataDir string, repo *store.Repo) *Store {
	return &Store{dataDir: dataDir, repo: repo, now: time.Now}
}

func (s *Store) rel(parts ...string) string { return filepath.Join(append([]string{Dir}, parts...)...) }

func (s *Store) read(id string) (Plan, error) {
	if !idPattern.MatchString(id) {
		return Plan{}, ErrNotFound
	}
	var p Plan
	err := store.ReadYAML(filepath.Join(s.dataDir, s.rel(id, planFile)), &p)
	if errors.Is(err, fs.ErrNotExist) {
		return Plan{}, ErrNotFound
	}
	p.ID = id
	return p, err
}

// All returns every plan by name; for administration.
func (s *Store) All() []Plan {
	files, _ := filepath.Glob(filepath.Join(s.dataDir, Dir, "*", planFile))
	var out []Plan
	for _, f := range files {
		if p, err := s.read(filepath.Base(filepath.Dir(f))); err == nil {
			out = append(out, p)
		}
	}
	slices.SortFunc(out, func(a, b Plan) int { return cmp.Compare(a.Name, b.Name) })
	return out
}

// For returns the plans of the groups sub belongs to.
func (s *Store) For(g access.Groups, sub string) []Plan {
	var out []Plan
	for _, p := range s.All() {
		if slices.Contains(g[p.Group], sub) {
			out = append(out, p)
		}
	}
	return out
}

// Get returns plan id if sub is in its group.
func (s *Store) Get(g access.Groups, sub, id string) (Plan, error) {
	p, err := s.read(id)
	if err != nil || !slices.Contains(g[p.Group], sub) {
		return Plan{}, ErrNotFound
	}
	return p, nil
}

// Create makes a plan for the members of group.
func (s *Store) Create(a store.Author, name, group string) (Plan, error) {
	name = strings.TrimSpace(name)
	if name == "" || utf8.RuneCountInString(name) > maxName || strings.TrimSpace(group) == "" {
		return Plan{}, ErrInvalid
	}
	p := Plan{ID: store.NewID(), Name: name, Group: group, Created: s.now().UTC().Truncate(time.Second)}
	return p, s.repo.Locked(func() error {
		if err := store.WriteYAML(filepath.Join(s.dataDir, s.rel(p.ID, planFile)), p); err != nil {
			return err
		}
		return s.repo.Commit(a, "kochplan: „"+name+"“ angelegt", s.rel(p.ID))
	})
}

// Delete removes a plan with all its weeks; Git keeps them.
func (s *Store) Delete(a store.Author, id string) error {
	p, err := s.read(id)
	if err != nil {
		return err
	}
	return s.repo.Locked(func() error {
		if err := os.RemoveAll(filepath.Join(s.dataDir, s.rel(id))); err != nil {
			return err
		}
		return s.repo.Commit(a, "kochplan: „"+p.Name+"“ gelöscht", s.rel(id))
	})
}

func (s *Store) weekPath(planID, key string) string {
	return filepath.Join(s.dataDir, s.rel(planID, key+".yaml"))
}

// Week reads one week of a plan; a week nobody planned is empty.
func (s *Store) Week(planID, key string) (Week, error) {
	if _, ok := parseKey(key); !ok {
		return Week{}, ErrInvalid
	}
	if _, err := s.read(planID); err != nil {
		return Week{}, err
	}
	w := Week{Key: key, Days: map[string][]Entry{}}
	err := store.ReadYAML(s.weekPath(planID, key), &w)
	if errors.Is(err, fs.ErrNotExist) {
		err = nil
	}
	if w.Days == nil {
		w.Days = map[string][]Entry{}
	}
	return w, err
}

// Update lets fn change one week and commits it.
func (s *Store) Update(a store.Author, planID, key string, fn func(*Week) error) error {
	if _, ok := parseKey(key); !ok {
		return ErrInvalid
	}
	return s.repo.Locked(func() error {
		w, err := s.Week(planID, key)
		if err != nil {
			return err
		}
		if err := fn(&w); err != nil {
			return err
		}
		if err := store.WriteYAML(s.weekPath(planID, key), w); err != nil {
			return err
		}
		return s.repo.Commit(a, "kochplan: Woche "+key+" geändert", s.rel(planID, key+".yaml"))
	})
}

// Add puts e on day with a new id; it reports false for a full day or a bad entry.
func (w *Week) Add(day string, e Entry) bool {
	if !dayPattern.MatchString(day) || len(w.Days[day]) >= maxPerDay || e.Servings < 1 || e.Servings > maxServings {
		return false
	}
	e.ID = store.NewID()
	w.Days[day] = append(w.Days[day], e)
	return true
}

// Find returns the entry with id and its day.
func (w *Week) Find(id string) (Entry, string, bool) {
	for day, es := range w.Days {
		if i := slices.IndexFunc(es, func(e Entry) bool { return e.ID == id }); i >= 0 {
			return es[i], day, true
		}
	}
	return Entry{}, "", false
}

// Set replaces the entry with the same id.
func (w *Week) Set(e Entry) bool {
	_, day, ok := w.Find(e.ID)
	if ok {
		w.Days[day][slices.IndexFunc(w.Days[day], func(x Entry) bool { return x.ID == e.ID })] = e
	}
	return ok
}

// Remove takes the entry with id off the plan.
func (w *Week) Remove(id string) bool {
	_, day, ok := w.Find(id)
	if ok {
		w.Days[day] = slices.DeleteFunc(w.Days[day], func(e Entry) bool { return e.ID == id })
		if len(w.Days[day]) == 0 {
			delete(w.Days, day)
		}
	}
	return ok
}
