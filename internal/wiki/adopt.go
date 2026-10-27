// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// Adopt turns the folder srcDir (a note: index.md plus attachments) into a
// new page below parentID. The folder is moved, not copied; the page gets a
// fresh id and its history starts with this commit.
func (w *Wiki) Adopt(a store.Author, space, parentID, title, srcDir string) (Page, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Page{}, ErrNoTitle
	}
	var adopted Page
	err := w.repo.Locked(func() error {
		w.mu.Lock()
		defer w.mu.Unlock()
		parentDir, err := w.parentDir(space, parentID)
		if err != nil {
			return err
		}
		adopted, err = w.adoptInto(a, space, parentDir, title, srcDir)
		return err
	})
	if err == nil {
		w.notify(adopted, false)
	}
	return adopted, err
}

// adoptInto moves srcDir into place and writes the page; on failure the
// folder goes back where it came from. The caller holds the locks.
func (w *Wiki) adoptInto(a store.Author, space, parentDir, title, srcDir string) (Page, error) {
	src, err := store.ReadDoc[struct{}](filepath.Join(srcDir, pageFile))
	if err != nil {
		return Page{}, err
	}
	abs := filepath.Join(pagesRoot(w.dataDir, space), filepath.FromSlash(parentDir))
	dir := path.Join(parentDir, store.UniqueSlug(abs, store.Slugify(title)))
	dst := filepath.Join(pagesRoot(w.dataDir, space), filepath.FromSlash(dir))
	if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
		return Page{}, err
	}
	if err := os.Rename(srcDir, dst); err != nil {
		return Page{}, err
	}
	now := time.Now().UTC().Truncate(time.Second)
	p := Page{Space: space, Dir: dir, Body: src.Body, Meta: PageMeta{ID: store.NewID(), Title: title,
		Order: w.nextOrder(space, parentDir), Created: now, Updated: now, Author: a.Name}}
	if err := w.writeFolder(a, p, "Seite aus Notiz angelegt: "+title); err != nil {
		_ = os.Rename(dst, srcDir)
		return Page{}, err
	}
	return p, nil
}

// writeFolder stores p and commits its whole folder (text and attachments).
func (w *Wiki) writeFolder(a store.Author, p Page, msg string) error {
	rel := pagePath(p.Space, p.Dir)
	err := store.WriteDoc(filepath.Join(w.dataDir, filepath.FromSlash(rel)), store.Doc[PageMeta]{Meta: p.Meta, Body: p.Body})
	if err != nil {
		return err
	}
	if err := w.repo.Commit(a, msg, path.Dir(rel)); err != nil {
		return err
	}
	w.index[p.Meta.ID] = location{space: p.Space, dir: p.Dir, meta: p.Meta}
	return nil
}
