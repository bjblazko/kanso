// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestScanEmptyRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	if out, err := exec.Command("git", "-C", dir, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	r, err := OpenRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if head, err := r.Head(); err != nil || head != "" {
		t.Errorf("head %q, %v", head, err)
	}
	sc, err := r.Scan()
	if err != nil || sc.Head != "" || len(sc.Changes) != 0 || len(sc.Live) != 0 {
		t.Errorf("scan %+v, %v", sc, err)
	}
}

func TestScanAddEditDelete(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a.txt", "hello")
	commit(t, r, anna, "a")
	write(t, r, "a.txt", "hello!")
	write(t, r, "b.txt", "bye")
	commit(t, r, ben, "edit a, add b")
	if err := os.Remove(filepath.Join(r.Dir(), "b.txt")); err != nil {
		t.Fatal(err)
	}
	commit(t, r, anna, "delete b")
	sc, err := r.Scan()
	if err != nil {
		t.Fatal(err)
	}
	want := []struct {
		path, email string
		size        int64
		deleted     bool
	}{{"a.txt", anna.Email, 5, false}, {"a.txt", ben.Email, 6, false}, {"b.txt", ben.Email, 3, false}, {"b.txt", anna.Email, 0, true}}
	if len(sc.Changes) != len(want) {
		t.Fatalf("changes %+v", sc.Changes)
	}
	for i, w := range want {
		c := sc.Changes[i]
		if c.Path != w.path || c.Email != w.email || c.Size != w.size || c.Deleted != w.deleted || c.When.IsZero() {
			t.Errorf("change %d = %+v, want %+v", i, c, w)
		}
	}
	if len(sc.Live) != 1 || sc.Live["a.txt"] != 6 {
		t.Errorf("live %v", sc.Live)
	}
	if sc.Deleted != 5+3 || sc.DeletedN != 2 {
		t.Errorf("deleted %d bytes in %d blobs", sc.Deleted, sc.DeletedN)
	}
	if head, _ := r.Head(); sc.Head != head || head == "" {
		t.Errorf("head %q vs %q", sc.Head, head)
	}
	if sc.PackBytes <= 0 {
		t.Errorf("pack bytes %d", sc.PackBytes)
	}
}

func TestScanReaddedBlobIsLive(t *testing.T) {
	r := newRepo(t)
	write(t, r, "a.txt", "same")
	commit(t, r, anna, "add")
	if err := os.Remove(filepath.Join(r.Dir(), "a.txt")); err != nil {
		t.Fatal(err)
	}
	commit(t, r, anna, "delete")
	write(t, r, "a.txt", "same")
	commit(t, r, anna, "add again")
	sc, err := r.Scan()
	if err != nil {
		t.Fatal(err)
	}
	if sc.DeletedN != 0 || sc.Deleted != 0 {
		t.Errorf("re-added blob counted as deleted: %d / %d", sc.DeletedN, sc.Deleted)
	}
}

func commit(t *testing.T, r *Repo, a Author, msg string) {
	t.Helper()
	if err := r.Commit(a, msg); err != nil {
		t.Fatal(err)
	}
}
