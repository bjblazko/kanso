// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// Errors callers can tell apart.
var (
	ErrNotFound    = errors.New("page not found")
	ErrHasChildren = errors.New("page has sub-pages")
	ErrBadMove     = errors.New("a page cannot move into itself or below itself")
	ErrNoTitle     = errors.New("a page needs a title")
)

func isNotExist(err error) bool { return errors.Is(err, fs.ErrNotExist) }

// Create adds a page titled title below parentID ("" = top level of space).
func (w *Wiki) Create(a store.Author, space, parentID, title string) (Page, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Page{}, ErrNoTitle
	}
	var created Page
	err := w.repo.Locked(func() error {
		w.mu.Lock()
		defer w.mu.Unlock()
		parentDir, err := w.parentDir(space, parentID)
		if err != nil {
			return err
		}
		created, err = w.writeNew(a, space, parentDir, title)
		return err
	})
	if err == nil {
		w.notify(created, false)
	}
	return created, err
}

// parentDir resolves the folder new children go into; the caller holds w.mu.
func (w *Wiki) parentDir(space, parentID string) (string, error) {
	if parentID == "" {
		return "", nil
	}
	loc, ok := w.index[parentID]
	if !ok || loc.space != space {
		return "", ErrNotFound
	}
	return loc.dir, nil
}

func (w *Wiki) writeNew(a store.Author, space, parentDir, title string) (Page, error) {
	abs := filepath.Join(pagesRoot(w.dataDir, space), filepath.FromSlash(parentDir))
	dir := path.Join(parentDir, store.UniqueSlug(abs, store.Slugify(title)))
	now := time.Now().UTC().Truncate(time.Second)
	meta := PageMeta{ID: store.NewID(), Title: title, Order: w.nextOrder(space, parentDir),
		Created: now, Updated: now, Author: a.Name}
	p := Page{Meta: meta, Space: space, Dir: dir}
	if err := w.write(a, p, "Seite angelegt: "+title); err != nil {
		return Page{}, err
	}
	return p, nil
}

// nextOrder puts a new page after its siblings; the caller holds w.mu.
func (w *Wiki) nextOrder(space, parentDir string) int {
	highest := 0
	for _, loc := range w.index {
		if loc.space == space && parentOf(loc.dir) == parentDir && loc.meta.Order > highest {
			highest = loc.meta.Order
		}
	}
	return highest + 1
}

func parentOf(dir string) string {
	if p := path.Dir(dir); p != "." {
		return p
	}
	return ""
}

// write stores p and commits it; the caller holds the repo lock and w.mu.
func (w *Wiki) write(a store.Author, p Page, msg string) error {
	rel := pagePath(p.Space, p.Dir)
	err := store.WriteDoc(filepath.Join(w.dataDir, filepath.FromSlash(rel)), store.Doc[PageMeta]{Meta: p.Meta, Body: p.Body})
	if err != nil {
		return err
	}
	if err := w.repo.Commit(a, msg, rel); err != nil {
		return err
	}
	w.index[p.Meta.ID] = location{space: p.Space, dir: p.Dir, meta: p.Meta}
	return nil
}

// Save stores a new title and body; the folder name stays.
func (w *Wiki) Save(a store.Author, id, title, body string) error {
	return w.save(a, id, title, body, "Seite geändert: ")
}

func (w *Wiki) save(a store.Author, id, title, body, msg string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrNoTitle
	}
	var saved Page
	err := w.repo.Locked(func() error {
		p, ok := w.ByID(id)
		if !ok {
			return ErrNotFound
		}
		p.Meta.Title, p.Body = title, body
		p.Meta.Updated, p.Meta.Author = time.Now().UTC().Truncate(time.Second), a.Name
		w.mu.Lock()
		defer w.mu.Unlock()
		saved = p
		return w.write(a, p, msg+title)
	})
	if err == nil {
		w.notify(saved, false)
	}
	return err
}

// Move puts id below newParentID ("" = top level) at position order.
func (w *Wiki) Move(a store.Author, id, newParentID string, order int) error {
	return w.repo.Locked(func() error {
		p, ok := w.ByID(id)
		if !ok {
			return ErrNotFound
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		target, err := w.parentDir(p.Space, newParentID)
		if err != nil {
			return err
		}
		if target == p.Dir || strings.HasPrefix(target+"/", p.Dir+"/") {
			return ErrBadMove
		}
		return w.relocate(a, p, target, order)
	})
}

// relocate moves p's folder below target and rewrites its order; the caller
// holds the repo lock and w.mu.
func (w *Wiki) relocate(a store.Author, p Page, target string, order int) error {
	newDir := p.Dir
	if target != parentOf(p.Dir) {
		abs := filepath.Join(pagesRoot(w.dataDir, p.Space), filepath.FromSlash(target))
		newDir = path.Join(target, store.UniqueSlug(abs, path.Base(p.Dir)))
		from, to := path.Dir(pagePath(p.Space, p.Dir)), path.Dir(pagePath(p.Space, newDir))
		if err := w.repo.Move(a, from, to, "Seite verschoben: "+p.Meta.Title); err != nil {
			return err
		}
		w.reindexMoved(p.Space, p.Dir, newDir)
	}
	p.Dir, p.Meta.Order = newDir, order
	return w.write(a, p, "Seite einsortiert: "+p.Meta.Title)
}

// reindexMoved rewrites the locations of a moved folder and its descendants.
func (w *Wiki) reindexMoved(space, oldDir, newDir string) {
	for id, loc := range w.index {
		if loc.space == space && (loc.dir == oldDir || strings.HasPrefix(loc.dir, oldDir+"/")) {
			loc.dir = newDir + strings.TrimPrefix(loc.dir, oldDir)
			w.index[id] = loc
		}
	}
}

// Delete removes a page without sub-pages, including its attachments.
func (w *Wiki) Delete(a store.Author, id string) error {
	var removed Page
	err := w.repo.Locked(func() error {
		p, ok := w.ByID(id)
		if !ok {
			return ErrNotFound
		}
		w.mu.Lock()
		defer w.mu.Unlock()
		if w.hasChildren(p) {
			return ErrHasChildren
		}
		folder := path.Dir(pagePath(p.Space, p.Dir))
		if err := os.RemoveAll(filepath.Join(w.dataDir, filepath.FromSlash(folder))); err != nil {
			return err
		}
		delete(w.index, id)
		removed = p
		return w.repo.Commit(a, "Seite gelöscht: "+p.Meta.Title, folder)
	})
	if err == nil {
		w.notify(removed, true)
	}
	return err
}

// hasChildren reports whether p has sub-pages; the caller holds w.mu.
func (w *Wiki) hasChildren(p Page) bool {
	for _, loc := range w.index {
		if loc.space == p.Space && strings.HasPrefix(loc.dir, p.Dir+"/") {
			return true
		}
	}
	return false
}

func (w *Wiki) notify(p Page, deleted bool) {
	if w.OnChange != nil {
		w.OnChange(p, deleted)
	}
}
