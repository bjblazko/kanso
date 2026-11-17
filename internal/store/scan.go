// SPDX-License-Identifier: Apache-2.0

package store

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// FileChange is one file added, changed or deleted by one commit.
type FileChange struct {
	When        time.Time
	Name, Email string // commit author
	Path, Blob  string // Blob is "" when deleted
	Size        int64  // bytes of Blob, 0 when deleted
	Deleted     bool
}

// Scan is the whole history of the repository, for reports.
type Scan struct {
	Head      string           // "" for a repository without commits
	Changes   []FileChange     // oldest first; a move is a delete plus an add
	Live      map[string]int64 // path → size at HEAD
	Deleted   int64            // bytes of blobs in the history but not at HEAD
	DeletedN  int              // number of those blobs
	PackBytes int64            // what the objects take on disk
}

// Head returns the current commit, "" when there is none yet.
func (r *Repo) Head() (string, error) {
	out, err := r.run("rev-parse", "--verify", "-q", "HEAD")
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	if all, err2 := r.run("rev-list", "-n", "1", "--all"); err2 == nil && len(bytes.TrimSpace(all)) == 0 {
		return "", nil
	}
	return "", err
}

// Scan reads the history of every file with sizes. It reads only and may run
// beside writes; a commit made meanwhile shows up in the next scan.
func (r *Repo) Scan() (Scan, error) {
	sc := Scan{Live: map[string]int64{}}
	head, err := r.Head()
	if err != nil || head == "" {
		return sc, err
	}
	sc.Head = head
	if sc.Changes, err = r.changes(head); err != nil {
		return sc, err
	}
	live, err := r.liveFiles(head, sc.Live)
	if err != nil {
		return sc, err
	}
	sizes, err := r.blobSizes(sc.Changes)
	if err != nil {
		return sc, err
	}
	sc.addSizes(sizes, live)
	sc.PackBytes, err = r.packBytes()
	return sc, err
}

// addSizes fills in the size of each change and sums the blobs no longer at HEAD.
func (sc *Scan) addSizes(sizes map[string]int64, live map[string]bool) {
	gone := map[string]bool{}
	for i, c := range sc.Changes {
		if c.Deleted {
			continue
		}
		sc.Changes[i].Size = sizes[c.Blob]
		if !live[c.Blob] && !gone[c.Blob] {
			gone[c.Blob] = true
			sc.Deleted += sizes[c.Blob]
			sc.DeletedN++
		}
	}
}

// changes lists every file change from the first commit to head.
func (r *Repo) changes(head string) ([]FileChange, error) {
	out, err := r.run("-c", "core.quotePath=false", "log", "--reverse", "--no-renames", "--raw", "--no-abbrev",
		"--format=%x00%ct%x09%ae%x09%an", head)
	if err != nil {
		return nil, err
	}
	var all []FileChange
	var who FileChange
	sc := bufio.NewScanner(bytes.NewReader(out))
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "\x00"):
			who = parseCommit(line[1:])
		case strings.HasPrefix(line, ":"):
			if c, ok := parseRaw(line, who); ok {
				all = append(all, c)
			}
		}
	}
	return all, sc.Err()
}

// parseCommit reads "unix-time<TAB>email<TAB>name".
func parseCommit(s string) FileChange {
	parts := strings.SplitN(s, "\t", 3)
	var c FileChange
	if len(parts) == 3 {
		secs, _ := strconv.ParseInt(parts[0], 10, 64)
		c.When, c.Email, c.Name = time.Unix(secs, 0).UTC(), parts[1], parts[2]
	}
	return c
}

// parseRaw reads ":mode mode old new STATUS<TAB>path" for the commit who.
func parseRaw(line string, who FileChange) (FileChange, bool) {
	meta, path, ok := strings.Cut(line, "\t")
	f := strings.Fields(meta)
	if !ok || len(f) < 5 || f[1] == "160000" || f[0] == ":160000" { // submodules are no files
		return who, false
	}
	c := who
	c.Path = unquote(path)
	if strings.HasPrefix(f[4], "D") {
		c.Deleted = true
	} else {
		c.Blob = f[3]
	}
	return c, true
}

// unquote undoes Git's C-style quoting of unusual path names.
func unquote(p string) string {
	if strings.HasPrefix(p, `"`) {
		if s, err := strconv.Unquote(p); err == nil {
			return s
		}
	}
	return p
}

// liveFiles fills sizes with the files at head and returns their blobs.
func (r *Repo) liveFiles(head string, sizes map[string]int64) (map[string]bool, error) {
	out, err := r.run("-c", "core.quotePath=false", "ls-tree", "-r", "-l", head)
	if err != nil {
		return nil, err
	}
	blobs := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		meta, path, ok := strings.Cut(line, "\t")
		f := strings.Fields(meta)
		if !ok || len(f) != 4 || f[1] != "blob" {
			continue
		}
		n, _ := strconv.ParseInt(f[3], 10, 64)
		sizes[unquote(path)], blobs[f[2]] = n, true
	}
	return blobs, nil
}

// blobSizes asks Git for the size of every blob the changes name.
func (r *Repo) blobSizes(changes []FileChange) (map[string]int64, error) {
	var in bytes.Buffer
	seen := map[string]bool{}
	for _, c := range changes {
		if c.Blob != "" && !seen[c.Blob] {
			seen[c.Blob] = true
			in.WriteString(c.Blob + "\n")
		}
	}
	sizes := map[string]int64{}
	if in.Len() == 0 {
		return sizes, nil
	}
	out, err := r.runInput(in.Bytes(), "cat-file", "--batch-check=%(objectname) %(objectsize)")
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if id, n, ok := strings.Cut(line, " "); ok {
			sizes[id], _ = strconv.ParseInt(n, 10, 64)
		}
	}
	return sizes, err
}

// packBytes is what the object store takes on disk, loose and packed.
func (r *Repo) packBytes() (int64, error) {
	out, err := r.run("count-objects", "-v")
	var kib int64
	for _, line := range strings.Split(string(out), "\n") {
		k, v, _ := strings.Cut(line, ": ")
		if k == "size" || k == "size-pack" {
			n, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
			kib += n
		}
	}
	return kib * 1024, err
}

// runInput runs git in the repository with stdin (nil for none).
func (r *Repo) runInput(stdin []byte, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), gitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", r.dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return out, nil
}
