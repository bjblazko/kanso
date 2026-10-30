// SPDX-License-Identifier: Apache-2.0

// Package wiki stores pages as folders with an index.md, nested like the tree.
package wiki

import (
	"cmp"
	"io/fs"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// PageMeta is the frontmatter of index.md.
type PageMeta struct {
	ID      string    `yaml:"id"`
	Title   string    `yaml:"title"`
	Order   int       `yaml:"order"`
	Created time.Time `yaml:"created"`
	Updated time.Time `yaml:"updated"`
	Author  string    `yaml:"author"`
	// Members are shared this page and its subpages on top of the space (ADR-009).
	Members []access.Member `yaml:"members,omitempty"`
}

// Page is a page with its location: Dir is relative to the space's pages
// folder and uses forward slashes, e.g. "garten/rasen".
type Page struct {
	Meta  PageMeta
	Space string
	Dir   string
	Body  string
}

// Node is a page in the tree.
type Node struct {
	Meta     PageMeta
	Dir      string
	Children []*Node
}

type location struct {
	space, dir string
	meta       PageMeta
}

// Wiki indexes all pages by id; this process is the only writer.
type Wiki struct {
	dataDir string
	repo    *store.Repo
	mu      sync.RWMutex
	index   map[string]location
	// OnChange is called after a page was written (deleted = false) or removed.
	OnChange func(p Page, deleted bool)
}

const pageFile = "index.md"

// pagePath is the repository-relative path of a page's index.md.
func pagePath(space, dir string) string {
	return path.Join("spaces", space, "pages", dir, pageFile)
}

func pagesRoot(dataDir, space string) string {
	return filepath.Join(dataDir, "spaces", space, "pages")
}

// NewWiki scans every space for pages.
func NewWiki(dataDir string, repo *store.Repo) (*Wiki, error) {
	w := &Wiki{dataDir: dataDir, repo: repo, index: map[string]location{}}
	spaceDirs, err := filepath.Glob(filepath.Join(dataDir, "spaces", "*"))
	if err != nil {
		return nil, err
	}
	for _, sd := range spaceDirs {
		if err := w.scanSpace(filepath.Base(sd)); err != nil {
			return nil, err
		}
	}
	return w, nil
}

func (w *Wiki) scanSpace(space string) error {
	root := pagesRoot(w.dataDir, space)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != pageFile {
			return err
		}
		doc, err := store.ReadDoc[PageMeta](p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, filepath.Dir(p))
		w.index[doc.Meta.ID] = location{space: space, dir: filepath.ToSlash(rel), meta: doc.Meta}
		return nil
	})
	if err != nil && !isNotExist(err) {
		return err
	}
	return nil
}

// Tree returns the pages of space as a tree, sorted by order, then title.
func (w *Wiki) Tree(space string) []*Node {
	w.mu.RLock()
	defer w.mu.RUnlock()
	byDir := map[string]*Node{}
	for _, loc := range w.index {
		if loc.space == space {
			byDir[loc.dir] = &Node{Meta: loc.meta, Dir: loc.dir}
		}
	}
	var roots []*Node
	for dir, n := range byDir {
		if parent, ok := byDir[path.Dir(dir)]; ok && strings.Contains(dir, "/") {
			parent.Children = append(parent.Children, n)
		} else {
			roots = append(roots, n)
		}
	}
	sortNodes(roots)
	return roots
}

func sortNodes(nodes []*Node) {
	slices.SortFunc(nodes, func(a, b *Node) int {
		return cmp.Or(cmp.Compare(a.Meta.Order, b.Meta.Order), strings.Compare(strings.ToLower(a.Meta.Title), strings.ToLower(b.Meta.Title)))
	})
	for _, n := range nodes {
		sortNodes(n.Children)
	}
}

// ByID returns a page with its body.
func (w *Wiki) ByID(id string) (Page, bool) {
	w.mu.RLock()
	loc, ok := w.index[id]
	w.mu.RUnlock()
	if !ok {
		return Page{}, false
	}
	doc, err := store.ReadDoc[PageMeta](filepath.Join(w.dataDir, filepath.FromSlash(pagePath(loc.space, loc.dir))))
	if err != nil {
		return Page{}, false
	}
	return Page{Meta: doc.Meta, Space: loc.space, Dir: loc.dir, Body: doc.Body}, true
}

// Ancestors returns the pages above id, outermost first.
func (w *Wiki) Ancestors(id string) []PageMeta {
	w.mu.RLock()
	defer w.mu.RUnlock()
	loc, ok := w.index[id]
	if !ok {
		return nil
	}
	var out []PageMeta
	for dir := path.Dir(loc.dir); dir != "."; dir = path.Dir(dir) {
		if m, ok := w.findByDir(loc.space, dir); ok {
			out = append([]PageMeta{m}, out...)
		}
	}
	return out
}

// findByDir looks a page up by location; the caller holds w.mu.
func (w *Wiki) findByDir(space, dir string) (PageMeta, bool) {
	for _, loc := range w.index {
		if loc.space == space && loc.dir == dir {
			return loc.meta, true
		}
	}
	return PageMeta{}, false
}

// IDs returns the ids of all pages.
func (w *Wiki) IDs() []string {
	w.mu.RLock()
	defer w.mu.RUnlock()
	ids := make([]string, 0, len(w.index))
	for id := range w.index {
		ids = append(ids, id)
	}
	return ids
}
