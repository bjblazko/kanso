// SPDX-License-Identifier: Apache-2.0

package users

import (
	"os/exec"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

func newStore(t *testing.T) (*Store, *store.Repo) {
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
	return s, repo
}

func TestUpsertCreatesFileAndCommit(t *testing.T) {
	s, repo := newStore(t)
	if _, err := s.Upsert(User{Sub: "anna", Name: "Anna", Email: "anna@example.org"}); err != nil {
		t.Fatal(err)
	}
	revs, err := repo.Log("users/anna.yaml", 5)
	if err != nil || len(revs) != 1 || revs[0].Author.Name != "Anna" {
		t.Fatalf("log = %+v, %v", revs, err)
	}
}

func TestUpsertKeepsLanguage(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.Upsert(User{Sub: "anna", Name: "Anna", Email: "a@x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLang("anna", "en"); err != nil {
		t.Fatal(err)
	}
	u, _ := s.Upsert(User{Sub: "anna", Name: "Anna B.", Email: "a@x"})
	if u.Lang != "en" || u.Name != "Anna B." {
		t.Errorf("user = %+v", u)
	}
}

func TestSetLangRejectsUnknown(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.Upsert(User{Sub: "anna", Name: "Anna", Email: "a@x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLang("anna", "fr"); err == nil {
		t.Error("fr accepted")
	}
}

func TestListSortedAndReloaded(t *testing.T) {
	s, repo := newStore(t)
	if _, err := s.Upsert(User{Sub: "z", Name: "Zoe", Email: "z@x"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Upsert(User{Sub: "a", Name: "Anna", Email: "a@x"}); err != nil {
		t.Fatal(err)
	}
	again, _ := NewStore(repo.Dir(), repo)
	list := again.List()
	if len(list) != 2 || list[0].Name != "Anna" {
		t.Errorf("list = %+v", list)
	}
	if u, ok := again.Get("z"); !ok || u.Name != "Zoe" {
		t.Errorf("get = %+v %v", u, ok)
	}
}

func TestRejectsUnsafeSub(t *testing.T) {
	s, _ := newStore(t)
	for _, sub := range []string{"../x", "a/b", "", "a\\b"} {
		if _, err := s.Upsert(User{Sub: sub, Name: "X", Email: "x@x"}); err == nil {
			t.Errorf("sub %q accepted", sub)
		}
	}
}

func TestNamesWithQuotesCanSignIn(t *testing.T) {
	s, repo := newStore(t)
	u, err := s.Upsert(User{Sub: "timo", Name: `Timo "Blazko" <B.>`, Email: "t@x"})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if u.Name != `Timo "Blazko" <B.>` {
		t.Errorf("display name changed: %q", u.Name)
	}
	revs, _ := repo.Log("users/timo.yaml", 1)
	if len(revs) != 1 || revs[0].Author.Name != "Timo Blazko B." {
		t.Errorf("history author = %+v", revs)
	}
}
