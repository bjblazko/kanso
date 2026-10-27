// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIgnoreAddsLineOnceAndCommits(t *testing.T) {
	r := newRepo(t)
	if err := os.WriteFile(filepath.Join(r.Dir(), ".gitignore"), []byte("/tmp/"), 0o640); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := r.Ignore("todos", "Todos vom Verlauf ausgenommen"); err != nil {
			t.Fatal(err)
		}
	}
	b, _ := os.ReadFile(filepath.Join(r.Dir(), ".gitignore"))
	if string(b) != "/tmp/\n/todos/\n" {
		t.Fatalf(".gitignore = %q", b)
	}
	if out, _ := r.run("status", "--porcelain"); strings.TrimSpace(string(out)) != "" {
		t.Fatalf("not committed: %s", out)
	}
}
