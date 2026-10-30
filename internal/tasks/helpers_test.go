// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

var family = access.Groups{"familie": {"anna", "ben"}}

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

func actor(sub string) Actor {
	return Actor{Sub: sub, Author: store.Author{Name: strings.ToUpper(sub[:1]) + sub[1:], Email: sub + "@example.org"}, Groups: family}
}

func admin() store.Author { return store.Author{Name: "Admin", Email: "admin@example.org"} }

func ownBoard(t *testing.T, s *Store, a Actor) Board {
	t.Helper()
	bs, err := s.Boards(a)
	if err != nil || len(bs) == 0 || !bs[0].Private() {
		t.Fatalf("Boards = %v, %v", bs, err)
	}
	return bs[0]
}

func familyBoard(t *testing.T, s *Store) Board {
	t.Helper()
	b, err := s.CreateGroupBoard(admin(), "Familie", "familie")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func mustCreate(t *testing.T, s *Store, a Actor, n New) Task {
	t.Helper()
	tk, _, err := s.Create(a, n)
	if err != nil {
		t.Fatal(err)
	}
	return tk
}

// gitLog returns "author|subject" lines for path, newest first.
func gitLog(t *testing.T, s *Store, path string) []string {
	t.Helper()
	out, err := exec.Command("git", "-C", s.dataDir, "log", "--format=%an|%s", "--", path).Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Fields(strings.ReplaceAll(strings.TrimSpace(string(out)), " ", "_"))
}

func at(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		panic(err)
	}
	return t
}
