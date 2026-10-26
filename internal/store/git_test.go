// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func newRepo(t *testing.T) *Repo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	r, err := OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func write(t *testing.T, r *Repo, name, content string) {
	t.Helper()
	if err := WriteAtomic(filepath.Join(r.Dir(), name), []byte(content)); err != nil {
		t.Fatal(err)
	}
}

var (
	anna = Author{Name: "Anna", Email: "anna@example.org"}
	ben  = Author{Name: "Ben", Email: "ben@example.org"}
)

func TestCommitAndLog(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a.md", "old\n")
	if err := r.Commit(anna, "first", "a.md"); err != nil {
		t.Fatal(err)
	}
	write(t, r, "a.md", "new\n")
	if err := r.Commit(ben, "second", "a.md"); err != nil {
		t.Fatal(err)
	}
	revs, err := r.Log("a.md", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(revs) != 2 || revs[0].Author.Name != "Ben" || revs[1].Author.Name != "Anna" {
		t.Fatalf("log = %+v", revs)
	}
	old, err := r.Show(revs[1].Hash, "a.md")
	if err != nil || string(old) != "old\n" {
		t.Errorf("show = %q, %v", old, err)
	}
}

func TestCommitWithoutChangesIsNoop(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a.md", "x\n")
	r.Commit(anna, "first", "a.md")
	if err := r.Commit(anna, "again", "a.md"); err != nil {
		t.Fatal(err)
	}
	revs, _ := r.Log("a.md", 10)
	if len(revs) != 1 {
		t.Errorf("revisions = %d", len(revs))
	}
}

func TestMoveKeepsHistory(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a/index.md", "x\n")
	r.Commit(anna, "first", "a")
	if err := r.Move(ben, "a", "b", "move"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(r.Dir(), "b", "index.md")); err != nil {
		t.Fatal(err)
	}
	revs, _ := r.Log("b/index.md", 10)
	if len(revs) != 2 {
		t.Errorf("history after move = %d revisions", len(revs))
	}
}

func TestCommitRejectsBadAuthor(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a.md", "x\n")
	for _, a := range []Author{{Name: "A\"B", Email: "a@b"}, {Name: "A\nB", Email: "a@b"}, {Name: "A", Email: "a>b"}} {
		if err := r.Commit(a, "m", "a.md"); err == nil {
			t.Errorf("author %+v accepted", a)
		}
	}
}

func TestOpenRepoIsIdempotent(t *testing.T) {
	r := newRepo(t)
	if _, err := OpenRepo(r.Dir()); err != nil {
		t.Fatal(err)
	}
}

func TestLogReportsPathPerRevision(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a/index.md", "eins\n")
	r.Commit(anna, "first", "a")
	if err := r.Move(ben, "a", "b", "move"); err != nil {
		t.Fatal(err)
	}
	write(t, r, "b/index.md", "zwei\n")
	r.Commit(ben, "second", "b")
	revs, err := r.Log("b/index.md", 10)
	if err != nil || len(revs) != 3 {
		t.Fatalf("log = %+v, %v", revs, err)
	}
	if revs[0].Path != "b/index.md" || revs[2].Path != "a/index.md" {
		t.Errorf("paths = %q … %q", revs[0].Path, revs[2].Path)
	}
	old, err := r.Show(revs[2].Hash, revs[2].Path)
	if err != nil || string(old) != "eins\n" {
		t.Errorf("show before move = %q, %v", old, err)
	}
}

func TestDiffText(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	diff, err := DiffText("eins\nzwei\n", "eins\ndrei\n")
	if err != nil || !strings.Contains(diff, "-zwei") || !strings.Contains(diff, "+drei") || strings.Contains(diff, "-eins") {
		t.Errorf("diff = %q, %v", diff, err)
	}
	if same, _ := DiffText("x\n", "x\n"); same != "" {
		t.Errorf("equal texts: %q", same)
	}
}
