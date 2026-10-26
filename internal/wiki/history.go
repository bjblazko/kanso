// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"strings"

	"github.com/bjblazko/kanso/internal/store"
)

// DiffLine is one line of a comparison: Kind is ' ', '+', '-' or '@'
// (the start of a changed section).
type DiffLine struct {
	Kind byte
	Text string
}

// History lists the revisions of page id, newest first.
func (w *Wiki) History(id string, limit int) ([]store.Revision, error) {
	p, ok := w.ByID(id)
	if !ok {
		return nil, ErrNotFound
	}
	return w.repo.Log(pagePath(p.Space, p.Dir), limit)
}

// revision finds hash in the history of id; other hashes are refused, so
// only this page's own past can be shown.
func (w *Wiki) revision(id, hash string) (store.Revision, error) {
	revs, err := w.History(id, 1000)
	if err != nil {
		return store.Revision{}, err
	}
	for _, r := range revs {
		if hash != "" && strings.HasPrefix(r.Hash, hash) && len(hash) >= 7 {
			return r, nil
		}
	}
	return store.Revision{}, ErrNotFound
}

// Revision returns page id as it was at hash.
func (w *Wiki) Revision(id, hash string) (Page, error) {
	rev, err := w.revision(id, hash)
	if err != nil {
		return Page{}, err
	}
	b, err := w.repo.Show(rev.Hash, rev.Path)
	if err != nil {
		return Page{}, err
	}
	doc, err := store.ParseDoc[PageMeta](b)
	return Page{Meta: doc.Meta, Body: doc.Body}, err
}

// Diff compares the text of page id between two of its revisions; the
// frontmatter (timestamps, author) is left out.
func (w *Wiki) Diff(id, from, to string) ([]DiffLine, error) {
	a, err := w.Revision(id, from)
	if err != nil {
		return nil, err
	}
	b, err := w.Revision(id, to)
	if err != nil {
		return nil, err
	}
	out, err := store.DiffText(a.Body, b.Body)
	return parseDiff(out), err
}

// parseDiff keeps the content lines of a unified diff and marks hunks.
func parseDiff(unified string) []DiffLine {
	var lines []DiffLine
	if unified == "" {
		return nil
	}
	inHunk := false
	for _, l := range strings.Split(strings.TrimSuffix(unified, "\n"), "\n") {
		switch {
		case strings.HasPrefix(l, "@@"):
			inHunk = true
			lines = append(lines, DiffLine{Kind: '@'})
		case !inHunk || l == `\ No newline at end of file`:
		case l == "":
			lines = append(lines, DiffLine{Kind: ' '})
		default:
			lines = append(lines, DiffLine{Kind: l[0], Text: l[1:]})
		}
	}
	return lines
}

// Restore makes the page look as it did at hash, as a new revision.
func (w *Wiki) Restore(a store.Author, id, hash string) error {
	old, err := w.Revision(id, hash)
	if err != nil {
		return err
	}
	return w.save(a, id, old.Meta.Title, old.Body, "Wiederhergestellt ("+hash[:7]+"): ")
}
