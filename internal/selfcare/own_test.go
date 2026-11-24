// SPDX-License-Identifier: Apache-2.0

package selfcare

import (
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

func newOwnStore(t *testing.T) *Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestOwnIdeas(t *testing.T) {
	s := newOwnStore(t)
	if err := s.Add("anna", "  Hund knuddeln ", "Fünf Minuten, ohne Handy.", "calm"); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("anna", "Tee", "", "lift"); err != nil {
		t.Fatal(err)
	}
	own := s.Own("anna")
	if len(own) != 2 || own[0].Title != "Hund knuddeln" || own[0].ID == "" || len(s.Own("ben")) != 0 {
		t.Fatalf("own %+v", own)
	}
	if err := s.Delete("anna", own[0].ID); err != nil || len(s.Own("anna")) != 1 {
		t.Fatalf("delete: %v %v", err, s.Own("anna"))
	}
	if err := s.Add("anna", "ok", "", "loud"); !errors.Is(err, ErrInvalid) {
		t.Errorf("unknown kind: %v", err)
	}
	if tip := own[1].Tip(); tip.ID != "own-"+own[1].ID || tip.Title.In("en") != "Tee" || !tip.Lift || tip.Gentle {
		t.Fatalf("tip %+v", tip)
	}
}

func TestOwnIdeasRefused(t *testing.T) {
	s := newOwnStore(t)
	for _, c := range []struct{ sub, title, body string }{
		{"anna", " ", ""}, {"anna", strings.Repeat("x", 81), ""}, {"anna", "ok", strings.Repeat("x", 301)}, {"../x", "ok", ""},
	} {
		if err := s.Add(c.sub, c.title, c.body, "calm"); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", c, err)
		}
	}
	for range maxOwn {
		_ = s.Add("ben", "Idee", "", "both")
	}
	if err := s.Add("ben", "eine zu viel", "", "calm"); !errors.Is(err, ErrInvalid) {
		t.Errorf("more than %d: %v", maxOwn, err)
	}
}

func TestPickFromIncludesOwn(t *testing.T) {
	pool := append(Tips(), Own{ID: "x", Title: "Hund"}.Tip())
	if got := PickFrom(pool, func(k int) int { return k - 1 }, ""); got.ID != "own-x" {
		t.Fatalf("got %q", got.ID)
	}
	one := []Tip{Own{ID: "x", Title: "Hund"}.Tip()}
	if got := PickFrom(one, func(int) int { return 0 }, "own-x"); got.ID != "own-x" {
		t.Fatalf("a pool of one must still answer, got %q", got.ID)
	}
}

func TestOnlyAndPractice(t *testing.T) {
	for _, tp := range Only(Tips(), true) {
		if !tp.Lift {
			t.Errorf("%s is not lifting", tp.ID)
		}
	}
	if len(Only(Tips(), true)) < 6 || len(Only(Tips(), false)) < 6 {
		t.Fatal("too few tips of a kind")
	}
	for _, l := range Levels {
		id, med := l.Practice()
		if _, ok := MeditationByID(id); med && !ok {
			t.Errorf("level %s: no meditation %q", l.Key, id)
		}
		if _, ok := ExerciseByID(id); !med && !ok {
			t.Errorf("level %s: no exercise %q", l.Key, id)
		}
	}
}
