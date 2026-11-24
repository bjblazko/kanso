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
	if err := s.Add("anna", "  Hund knuddeln ", "Fünf Minuten, ohne Handy."); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("anna", "Tee", ""); err != nil {
		t.Fatal(err)
	}
	own := s.Own("anna")
	if len(own) != 2 || own[0].Title != "Hund knuddeln" || own[0].ID == "" || len(s.Own("ben")) != 0 {
		t.Fatalf("own %+v", own)
	}
	if err := s.Delete("anna", own[0].ID); err != nil || len(s.Own("anna")) != 1 {
		t.Fatalf("delete: %v %v", err, s.Own("anna"))
	}
	if tip := own[1].Tip(); tip.ID != "own-"+own[1].ID || tip.Title.In("en") != "Tee" {
		t.Fatalf("tip %+v", tip)
	}
}

func TestOwnIdeasRefused(t *testing.T) {
	s := newOwnStore(t)
	for _, c := range []struct{ sub, title, body string }{
		{"anna", " ", ""}, {"anna", strings.Repeat("x", 81), ""}, {"anna", "ok", strings.Repeat("x", 301)}, {"../x", "ok", ""},
	} {
		if err := s.Add(c.sub, c.title, c.body); !errors.Is(err, ErrInvalid) {
			t.Errorf("%+v: %v", c, err)
		}
	}
	for range maxOwn {
		_ = s.Add("ben", "Idee", "")
	}
	if err := s.Add("ben", "eine zu viel", ""); !errors.Is(err, ErrInvalid) {
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
