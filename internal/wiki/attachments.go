// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"

	"github.com/bjblazko/kanso/internal/store"
)

// ErrTooLarge means an upload exceeded the size limit.
var ErrTooLarge = errors.New("attachment too large")

const attachmentsDir = "attachments"

var (
	safeName = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9]{1,8})?$`)
)

// attachmentDir is the repository-relative attachments folder of p.
func attachmentDir(p Page) string {
	return path.Join(path.Dir(pagePath(p.Space, p.Dir)), attachmentsDir)
}

// AddAttachment stores r (at most max bytes) next to page id and returns
// the stored file name.
func (w *Wiki) AddAttachment(a store.Author, id, filename string, r io.Reader, max int64) (string, error) {
	var name string
	err := w.repo.Locked(func() error {
		p, ok := w.ByID(id)
		if !ok {
			return ErrNotFound
		}
		dir := filepath.Join(w.dataDir, filepath.FromSlash(attachmentDir(p)))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
		name = store.UniqueFile(dir, store.AttachmentName(filename))
		if err := writeLimited(filepath.Join(dir, name), r, max); err != nil {
			return err
		}
		return w.repo.Commit(a, "Anhang hinzugefügt: "+name, path.Join(attachmentDir(p), name))
	})
	return name, err
}

// uniqueFile returns name, or stem-2.ext, stem-3.ext … if dir holds it.
func writeLimited(dst string, r io.Reader, max int64) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > max {
		err = ErrTooLarge
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

// attachmentPath resolves name below page id's attachments, refusing tricks.
func (w *Wiki) attachmentPath(id, name string) (string, Page, error) {
	p, ok := w.ByID(id)
	if !ok || !safeName.MatchString(name) {
		return "", p, ErrNotFound
	}
	abs, err := store.Join(w.dataDir, filepath.FromSlash(attachmentDir(p)), name)
	return abs, p, err
}

// OpenAttachment opens a stored attachment for reading.
func (w *Wiki) OpenAttachment(id, name string) (*os.File, error) {
	abs, _, err := w.attachmentPath(id, name)
	if err != nil {
		return nil, err
	}
	return os.Open(abs)
}

// ListAttachments returns the attachment names of page id, sorted.
func (w *Wiki) ListAttachments(id string) []string {
	p, ok := w.ByID(id)
	if !ok {
		return nil
	}
	entries, _ := os.ReadDir(filepath.Join(w.dataDir, filepath.FromSlash(attachmentDir(p))))
	var names []string
	for _, e := range entries {
		if !e.IsDir() && safeName.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	slices.Sort(names)
	return names
}

// DeleteAttachment removes one attachment.
func (w *Wiki) DeleteAttachment(a store.Author, id, name string) error {
	return w.repo.Locked(func() error {
		abs, p, err := w.attachmentPath(id, name)
		if err != nil {
			return err
		}
		if err := os.Remove(abs); err != nil {
			return err
		}
		return w.repo.Commit(a, "Anhang gelöscht: "+name, path.Join(attachmentDir(p), name))
	})
}
