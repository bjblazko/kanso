// SPDX-License-Identifier: Apache-2.0

package spaces

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

var admin = store.Author{Name: "Admin", Email: "admin@example.org"}

func newService(t *testing.T) *Service {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	svc, err := NewService(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestCreateUsesSlugAndCommits(t *testing.T) {
	svc := newService(t)
	sp, err := svc.Create(admin, "Gesundheit Timo", "Arzttermine")
	if err != nil {
		t.Fatal(err)
	}
	if sp.Slug != "gesundheit-timo" || sp.Name != "Gesundheit Timo" || sp.Icon != DefaultIcon {
		t.Errorf("space = %+v", sp)
	}
	revs, _ := svc.repo.Log("spaces/gesundheit-timo/space.yaml", 5)
	if len(revs) != 1 || revs[0].Author.Name != "Admin" {
		t.Errorf("history = %+v", revs)
	}
	again, _ := svc.Create(admin, "Gesundheit Timo", "")
	if again.Slug != "gesundheit-timo-2" {
		t.Errorf("duplicate slug = %q", again.Slug)
	}
}

func TestCreateRejectsEmptyName(t *testing.T) {
	svc := newService(t)
	if _, err := svc.Create(admin, "   ", ""); err == nil {
		t.Error("empty name accepted")
	}
}

func TestReloadAndVisible(t *testing.T) {
	svc := newService(t)
	sp, _ := svc.Create(admin, "Haus", "")
	sp.Members = []access.Member{{Group: "familie", Role: access.Editor}, {User: "oma", Role: access.Viewer}}
	if err := svc.Update(admin, sp); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(admin, "Geheim", ""); err != nil {
		t.Fatal(err)
	}
	again, err := NewService(svc.dataDir, svc.repo)
	if err != nil {
		t.Fatal(err)
	}
	g := access.Groups{"familie": {"anna"}}
	if got := again.RoleOf(access.Principal{Sub: "anna"}, "haus", g); got != access.Editor {
		t.Errorf("anna role = %v", got)
	}
	vis := again.Visible(access.Principal{Sub: "oma"}, g)
	if len(vis) != 1 || vis[0].Slug != "haus" {
		t.Errorf("visible for oma = %+v", vis)
	}
	if len(again.Visible(access.Principal{Sub: "x", Admin: true}, g)) != 2 {
		t.Error("admin should see all spaces")
	}
}

func TestUpdateRejectsUnknownIconAndBadMember(t *testing.T) {
	svc := newService(t)
	sp, _ := svc.Create(admin, "Haus", "")
	sp.Icon = "rocket"
	if err := svc.Update(admin, sp); err == nil {
		t.Error("unknown icon accepted")
	}
	sp.Icon = DefaultIcon
	sp.Members = []access.Member{{User: "a", Group: "b", Role: access.Viewer}}
	if err := svc.Update(admin, sp); err == nil {
		t.Error("member with user and group accepted")
	}
}

func TestGetUnknown(t *testing.T) {
	svc := newService(t)
	if _, ok := svc.Get("../etc"); ok {
		t.Error("found ../etc")
	}
}

func TestDeleteRemovesFolderAndCommits(t *testing.T) {
	svc := newService(t)
	sp, _ := svc.Create(admin, "Haus", "")
	page := filepath.Join(svc.repo.Dir(), Dir, sp.Slug, "pages", "x", "index.md")
	if err := os.MkdirAll(filepath.Dir(page), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(page, []byte("x"), 0o640); err != nil {
		t.Fatal(err)
	}
	calls := 0
	if err := svc.Delete(admin, sp.Slug, func() { calls++ }); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(svc.repo.Dir(), Dir, sp.Slug)); !os.IsNotExist(err) {
		t.Errorf("folder still there: %v", err)
	}
	if _, ok := svc.Get(sp.Slug); ok || len(svc.List()) != 0 {
		t.Error("space still listed")
	}
	if calls != 1 {
		t.Errorf("also called %d times", calls)
	}
	revs, _ := svc.repo.Log("spaces/"+sp.Slug, 1)
	if len(revs) != 1 || revs[0].Message != "Bereich gelöscht: Haus" {
		t.Errorf("history = %+v", revs)
	}
}

func TestDeleteUnknownSpace(t *testing.T) {
	svc := newService(t)
	called := false
	if err := svc.Delete(admin, "nope", func() { called = true }); !errors.Is(err, ErrUnknown) {
		t.Errorf("err = %v", err)
	}
	if called {
		t.Error("also called for an unknown space")
	}
}
