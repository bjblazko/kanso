// SPDX-License-Identifier: Apache-2.0

package notes

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/store"
)

const titleRunes = 80

// Create stores a new note with its files. An empty id gets a new one; an id
// that already exists returns that note unchanged and created == false, so a
// device may safely send the same note twice.
func (s *Store) Create(owner, id, body string, files []Upload, max int64) (n Note, created bool, err error) {
	if id == "" {
		id = store.NewID()
	}
	dir, err := s.folder(owner, id)
	if err != nil {
		return Note{}, false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, err := read(owner, dir); err == nil {
		return existing, false, nil
	}
	n, err = s.write(owner, dir, id, body, files, max)
	if err != nil {
		_ = os.RemoveAll(dir)
		return Note{}, false, err
	}
	s.changed(n, false)
	return n, true, nil
}

func (s *Store) write(owner, dir, id, body string, files []Upload, max int64) (Note, error) {
	var links []string
	for _, f := range files {
		name, err := store.SaveUpload(filepath.Join(dir, attachmentsDir), f.Name, f.R, max)
		if err != nil {
			return Note{}, err
		}
		links = append(links, "![](attachments/"+name+")\n")
	}
	now := s.stamp(time.Time{})
	n := Note{Owner: owner, Meta: Meta{ID: id, Created: now, Updated: now}, Body: withLinks(body, links)}
	if err := store.WriteDoc(filepath.Join(dir, noteFile), store.Doc[Meta]{Meta: n.Meta, Body: n.Body}); err != nil {
		return Note{}, err
	}
	n.Attachments = listAttachments(dir)
	return n, nil
}

// withLinks appends image links below the text.
func withLinks(body string, links []string) string {
	if len(links) == 0 {
		return body
	}
	all := strings.Join(links, "")
	if strings.TrimSpace(body) == "" {
		return all
	}
	if !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return body + "\n" + all
}

// Save replaces the text of a note opened at version (its Updated time).
func (s *Store) Save(owner, id, body string, version time.Time) (Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.Get(owner, id)
	if err != nil {
		return Note{}, err
	}
	if !n.Meta.Updated.Equal(version) {
		return Note{}, ErrConflict
	}
	n.Body, n.Meta.Updated = body, s.stamp(n.Meta.Updated)
	dir, _ := s.folder(owner, id)
	if err := store.WriteDoc(filepath.Join(dir, noteFile), store.Doc[Meta]{Meta: n.Meta, Body: n.Body}); err != nil {
		return Note{}, err
	}
	s.changed(n, false)
	return n, nil
}

// AddAttachment stores a file next to an existing note.
func (s *Store) AddAttachment(owner, id, filename string, r io.Reader, max int64) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.Get(owner, id); err != nil {
		return "", err
	}
	dir, _ := s.folder(owner, id)
	return store.SaveUpload(filepath.Join(dir, attachmentsDir), filename, r, max)
}

// OpenAttachment opens a file of a note, refusing names outside it.
func (s *Store) OpenAttachment(owner, id, name string) (*os.File, error) {
	dir, err := s.Dir(owner, id)
	if err != nil {
		return nil, err
	}
	if !store.ValidAttachmentName(name) {
		return nil, ErrNotFound
	}
	f, err := os.Open(filepath.Join(dir, attachmentsDir, name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func listAttachments(dir string) []string {
	entries, _ := os.ReadDir(filepath.Join(dir, attachmentsDir))
	var names []string
	for _, e := range entries {
		if !e.IsDir() && store.ValidAttachmentName(e.Name()) {
			names = append(names, e.Name())
		}
	}
	return names
}

// Title is the first line of text, without list or heading marks, at most
// 80 characters; "" when the note has no text line (only photos).
func Title(n Note) string {
	for line := range strings.SplitSeq(n.Body, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimSpace(strings.TrimLeft(line, "#"))
		line = strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
		if line == "" || strings.HasPrefix(line, "![") {
			continue
		}
		if utf8.RuneCountInString(line) > titleRunes {
			line = string([]rune(line)[:titleRunes])
		}
		return line
	}
	return ""
}
