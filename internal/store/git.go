// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const gitTimeout = 30 * time.Second

// Author is the person a change is recorded for.
type Author struct {
	Name  string
	Email string
}

func (a Author) valid() bool {
	bad := "\"<>\n\r\x00"
	return a.Name != "" && a.Email != "" &&
		!strings.ContainsAny(a.Name, bad) && !strings.ContainsAny(a.Email, bad)
}

// Revision is one commit that touched a file.
type Revision struct {
	Hash    string
	Author  Author
	Time    time.Time
	Message string
}

// Repo is the data directory as a Git repository.
type Repo struct {
	dir   string
	write sync.Mutex // serialises file changes plus their commit (Locked)
	git   sync.Mutex // serialises git index access
}

// OpenRepo opens dir, initialising a repository if there is none.
func OpenRepo(dir string) (*Repo, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	r := &Repo{dir: abs}
	if _, err := os.Stat(filepath.Join(abs, ".git")); err == nil {
		return r, nil
	}
	if err := os.MkdirAll(abs, dirMode); err != nil {
		return nil, err
	}
	return r, r.initialise()
}

func (r *Repo) initialise() error {
	steps := [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.name", "Kanso"},
		{"config", "user.email", "kanso@localhost"},
		{"commit", "-q", "--allow-empty", "-m", "Initialise Kanso data"},
	}
	for _, args := range steps {
		if _, err := r.run(args...); err != nil {
			return err
		}
	}
	return nil
}

// Dir returns the absolute path of the repository.
func (r *Repo) Dir() string { return r.dir }

// Locked runs fn while holding the write lock; use it around
// "change files, then Commit" so concurrent writers cannot interleave.
func (r *Repo) Locked(fn func() error) error {
	r.write.Lock()
	defer r.write.Unlock()
	return fn()
}

// Commit records the current state of paths (relative to Dir) for a.
// Nothing changed means nothing is committed.
func (r *Repo) Commit(a Author, msg string, paths ...string) error {
	if !a.valid() {
		return errors.New("invalid author")
	}
	r.git.Lock()
	defer r.git.Unlock()
	if _, err := r.run(append([]string{"add", "-A", "--"}, paths...)...); err != nil {
		return err
	}
	return r.commitStaged(a, msg, paths...)
}

// commitStaged commits what is staged for paths; the caller holds r.git.
func (r *Repo) commitStaged(a Author, msg string, paths ...string) error {
	if _, err := r.run(append([]string{"diff", "--cached", "--quiet", "--"}, paths...)...); err == nil {
		return nil
	}
	author := fmt.Sprintf("%s <%s>", a.Name, a.Email)
	_, err := r.run(append([]string{"commit", "-q", "--author", author, "-m", msg, "--"}, paths...)...)
	return err
}

// Move renames from to to and commits both paths.
func (r *Repo) Move(a Author, from, to, msg string) error {
	if !a.valid() {
		return errors.New("invalid author")
	}
	if err := os.MkdirAll(filepath.Dir(filepath.Join(r.dir, to)), dirMode); err != nil {
		return err
	}
	r.git.Lock()
	defer r.git.Unlock()
	if _, err := r.run("mv", "--", from, to); err != nil {
		return err
	}
	return r.commitStaged(a, msg, from, to)
}

const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// Log lists the revisions of path, newest first, following renames.
func (r *Repo) Log(path string, limit int) ([]Revision, error) {
	format := "--format=%H%x1f%an%x1f%ae%x1f%aI%x1f%s%x1e"
	out, err := r.run("log", "--follow", "-n", strconv.Itoa(limit), format, "--", path)
	if err != nil {
		return nil, err
	}
	var revs []Revision
	for _, rec := range strings.Split(string(out), recordSep) {
		if rev, ok := parseRevision(strings.TrimSpace(rec)); ok {
			revs = append(revs, rev)
		}
	}
	return revs, nil
}

func parseRevision(rec string) (Revision, bool) {
	f := strings.Split(rec, fieldSep)
	if len(f) != 5 {
		return Revision{}, false
	}
	t, _ := time.Parse(time.RFC3339, f[3])
	return Revision{Hash: f[0], Author: Author{Name: f[1], Email: f[2]}, Time: t, Message: f[4]}, true
}

// Show returns the content of path at revision hash.
func (r *Repo) Show(hash, path string) ([]byte, error) {
	return r.run("show", hash+":"+filepath.ToSlash(path))
}

// Diff returns the unified diff of path between two revisions.
func (r *Repo) Diff(from, to, path string) (string, error) {
	out, err := r.run("diff", "--no-color", from, to, "--", path)
	return string(out), err
}

func (r *Repo) run(args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
