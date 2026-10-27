// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"

	"github.com/bjblazko/kanso/internal/store"
)

// ErrTooLarge means an upload exceeded the size limit.
var ErrTooLarge = store.ErrTooLarge

const attachmentsDir = "attachments"

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
		var err error
		if name, err = store.SaveUpload(dir, filename, r, max); err != nil {
			return err
		}
		return w.repo.Commit(a, "Anhang hinzugefügt: "+name, path.Join(attachmentDir(p), name))
	})
	return name, err
}

// attachmentPath resolves name below page id's attachments, refusing tricks.
func (w *Wiki) attachmentPath(id, name string) (string, Page, error) {
	p, ok := w.ByID(id)
	if !ok || !store.ValidAttachmentName(name) {
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
		if !e.IsDir() && store.ValidAttachmentName(e.Name()) {
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
