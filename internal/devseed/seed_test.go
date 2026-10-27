// SPDX-License-Identifier: Apache-2.0

package devseed

import (
	"errors"
	"os/exec"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
	"github.com/bjblazko/kanso/internal/wiki"
)

type stores struct {
	dir  string
	repo *store.Repo
	us   *users.Store
	sp   *spaces.Service
	wk   *wiki.Wiki
	ns   *notes.Store
}

func newStores(t *testing.T) stores {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	us, err := users.NewStore(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := spaces.NewService(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	wk, err := wiki.NewWiki(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	ns, err := notes.NewStore(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return stores{dir: repo.Dir(), repo: repo, us: us, sp: sp, wk: wk, ns: ns}
}

func seed(t *testing.T, s stores) error {
	t.Helper()
	return Seed(s.dir, s.repo, s.us, s.sp, s.wk, s.ns)
}

func TestSeedCreatesPeople(t *testing.T) {
	s := newStores(t)
	if err := seed(t, s); err != nil {
		t.Fatal(err)
	}
	for _, p := range People {
		if u, ok := s.us.Get(p.Sub); !ok || u.Name != p.Name {
			t.Errorf("person %s: %+v %v", p.Sub, u, ok)
		}
	}
}

func TestSeedGivesEachPersonTheirRoles(t *testing.T) {
	s := newStores(t)
	if err := seed(t, s); err != nil {
		t.Fatal(err)
	}
	g, err := access.LoadGroups(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		sub, space string
		want       access.Role
	}{
		{"ben", "haus", access.Owner},
		{"ben", "familie", access.Editor},
		{"lena", "familie", access.Editor},
		{"lena", "haus", access.Viewer},
		{"lena", "lenas-ecke", access.Owner},
		{"oma", "familie", access.Viewer},
		{"oma", "haus", access.None},
		{"gast", "familie", access.None},
	}
	for _, c := range cases {
		if got := s.sp.RoleOf(access.Principal{Sub: c.sub}, c.space, g); got != c.want {
			t.Errorf("%s in %s = %v, want %v", c.sub, c.space, got, c.want)
		}
	}
}

func TestSeedWritesPages(t *testing.T) {
	s := newStores(t)
	if err := seed(t, s); err != nil {
		t.Fatal(err)
	}
	for _, slug := range []string{"familie", "haus", "lenas-ecke"} {
		if len(s.wk.Tree(slug)) == 0 {
			t.Errorf("space %s has no pages", slug)
		}
	}
}

func TestSeedRefusesDataThatIsNotEmpty(t *testing.T) {
	s := newStores(t)
	if err := seed(t, s); err != nil {
		t.Fatal(err)
	}
	if err := seed(t, s); !errors.Is(err, ErrNotEmpty) {
		t.Errorf("second seed: %v", err)
	}
}

func TestExactlyOneAdmin(t *testing.T) {
	n := 0
	for _, p := range People {
		if p.Admin {
			n++
		}
	}
	if n != 1 {
		t.Errorf("admins = %d", n)
	}
}

func TestSeedWritesNotes(t *testing.T) {
	s := newStores(t)
	if err := seed(t, s); err != nil {
		t.Fatal(err)
	}
	if n := len(s.ns.List("anna")); n != 2 {
		t.Errorf("anna's notes = %d", n)
	}
	if n := len(s.ns.List("ben")); n != 1 {
		t.Errorf("ben's notes = %d", n)
	}
}
