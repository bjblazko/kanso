// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Ignore keeps folder dir (relative to the repository) out of the history by
// adding it to .gitignore, once, and commits that change as Kanso.
func (r *Repo) Ignore(dir, msg string) error {
	line := "/" + dir + "/"
	path := filepath.Join(r.dir, ".gitignore")
	b, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if slices.Contains(strings.Split(string(b), "\n"), line) {
		return nil
	}
	if len(b) > 0 && b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	if err := WriteAtomic(path, append(b, line+"\n"...)); err != nil {
		return err
	}
	return r.Commit(Author{Name: "Kanso", Email: "kanso@localhost"}, msg, ".gitignore")
}
