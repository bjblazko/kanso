// SPDX-License-Identifier: Apache-2.0

package mealplan

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

var (
	admin  = store.Author{Name: "Admin", Email: "admin@example.org"}
	family = access.Groups{"familie": {"anna", "ben"}, "garten": {"carla"}}
)

func newStore(t *testing.T) *Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return NewStore(repo.Dir(), repo)
}

func TestPlansBelongToGroups(t *testing.T) {
	s := newStore(t)
	p, err := s.Create(admin, "Familienküche", "familie")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(admin, " ", "familie"); !errors.Is(err, ErrInvalid) {
		t.Error("empty name accepted")
	}
	if got := s.For(family, "anna"); len(got) != 1 || got[0].ID != p.ID || got[0].Name != "Familienküche" {
		t.Fatalf("anna %+v", got)
	}
	if len(s.For(family, "carla")) != 0 {
		t.Error("carla sees the family plan")
	}
	if _, err := s.Get(family, "carla", p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("carla get: %v", err)
	}
	if len(s.All()) != 1 {
		t.Error("All")
	}
	if err := s.Delete(admin, p.ID); err != nil || len(s.All()) != 0 {
		t.Fatalf("delete %v", err)
	}
}

func TestUpdateWeekCommits(t *testing.T) {
	s := newStore(t)
	p, _ := s.Create(admin, "Küche", "familie")
	err := s.Update(admin, p.ID, "2026-W46", func(w *Week) error {
		w.Add("2026-11-10", Entry{Recipe: "r1", Title: "Curry", Servings: 4})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.Week(p.ID, "2026-W46")
	if err != nil || len(w.Days["2026-11-10"]) != 1 || w.Days["2026-11-10"][0].ID == "" {
		t.Fatalf("%+v %v", w, err)
	}
	if _, err := os.Stat(filepath.Join(s.dataDir, Dir, p.ID, "2026-W46.yaml")); err != nil {
		t.Fatal(err)
	}
	out, _ := exec.Command("git", "-C", s.dataDir, "log", "--format=%s").Output()
	if len(out) == 0 {
		t.Error("no commit")
	}
	empty, err := s.Week(p.ID, "2026-W47")
	if err != nil || len(empty.Days) != 0 {
		t.Fatalf("missing week %+v %v", empty, err)
	}
	e, day, ok := w.Find(w.Days["2026-11-10"][0].ID)
	if !ok || day != "2026-11-10" || e.Title != "Curry" {
		t.Fatal("Find")
	}
	if !w.Remove(e.ID) || len(w.Days["2026-11-10"]) != 0 {
		t.Fatal("Remove")
	}
}

func TestParseWeek(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC) // Saturday, week 45
	for key, want := range map[string]string{"2026-W46": "2026-W46", "2026-W99": "2026-W45", "abc": "2026-W45", "": "2026-W45", "2025-W53": "2026-W45", "2026-W53": "2026-W53", "2020-W53": "2020-W53"} {
		_, got := ParseWeek(key, now)
		if got != want {
			t.Errorf("%q → %q, want %q", key, got, want)
		}
	}
	start, _ := ParseWeek("2026-W46", now)
	days := Days(start)
	if len(days) != 7 || days[0].Format("2006-01-02") != "2026-11-09" || days[6].Format("2006-01-02") != "2026-11-15" {
		t.Fatalf("%v", days)
	}
	if WeekKey(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)) != "2026-W53" {
		t.Error("ISO week across the new year")
	}
	if Shift("2026-W52", 1) != "2026-W53" || Shift("2026-W46", -1) != "2026-W45" {
		t.Error("Shift")
	}
}
