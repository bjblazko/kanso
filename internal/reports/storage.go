// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"cmp"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// Area is what one space (or "notes", "lists", "boards", "mealplans",
// "other") takes: text, attachments, and blobs Git keeps that are no longer current.
type Area struct {
	Name                       string
	Text, Attachments, Deleted int64
}

// Total is everything the area takes.
func (a Area) Total() int64 { return a.Text + a.Attachments + a.Deleted }

// Storage sums docs per area, largest first; with a history scan it adds
// what Git keeps of old versions and deleted files, by where they lived.
func Storage(docs []Doc, sc *store.Scan) []Area {
	by := map[string]*Area{}
	area := func(name string) *Area {
		if by[name] == nil {
			by[name] = &Area{Name: name}
		}
		return by[name]
	}
	for _, d := range docs {
		if d.Kind == Attachment {
			area(d.Space).Attachments += d.Bytes
		} else {
			area(d.Space).Text += d.Bytes
		}
	}
	if sc != nil {
		seen := map[string]bool{}
		for _, c := range sc.Changes {
			if c.Blob == "" || sc.LiveBlobs[c.Blob] || seen[c.Blob] {
				continue
			}
			seen[c.Blob] = true
			area(AreaOf(c.Path)).Deleted += c.Size
		}
	}
	out := make([]Area, 0, len(by))
	for _, a := range by {
		out = append(out, *a)
	}
	slices.SortFunc(out, func(a, b Area) int { return cmp.Or(cmp.Compare(b.Total(), a.Total()), cmp.Compare(a.Name, b.Name)) })
	return out
}

// AreaOf names the area of a repository path: a space's slug, or the kind of folder.
func AreaOf(path string) string {
	parts := strings.Split(path, "/")
	switch parts[0] {
	case "spaces":
		if len(parts) > 1 {
			return parts[1]
		}
	case "notes":
		return "notes"
	case "todos":
		return "lists"
	case "boards":
		return "boards"
	case "mealplans":
		return "mealplans"
	}
	return "other"
}

// SizePoint is the repository at the end of Month: Repo is every blob it ever
// held, Tree the files that were current then. Both uncompressed.
type SizePoint struct {
	Month      time.Time
	Repo, Tree int64
}

// StorageOverTime replays the history month by month until to.
func StorageOverTime(sc *store.Scan, to time.Time) []SizePoint {
	if len(sc.Changes) == 0 {
		return nil
	}
	r := replay{tree: map[string]int64{}, blobs: map[string]bool{}}
	var pts []SizePoint
	i := 0
	for m := monthOf(sc.Changes[0].When); !m.After(to); m = m.AddDate(0, 1, 0) {
		end := m.AddDate(0, 1, 0)
		for ; i < len(sc.Changes) && sc.Changes[i].When.Before(end); i++ {
			r.apply(sc.Changes[i])
		}
		pts = append(pts, SizePoint{Month: m, Repo: r.repo, Tree: r.sum})
	}
	return pts
}

// replay is the repository's files and blobs while walking its history.
type replay struct {
	tree      map[string]int64 // current path → size
	blobs     map[string]bool  // every blob seen
	repo, sum int64            // bytes of all blobs, bytes of the current files
}

func (r *replay) apply(c store.FileChange) {
	for _, p := range []string{c.From, c.Path} {
		if p != "" {
			r.sum -= r.tree[p]
			delete(r.tree, p)
		}
	}
	if c.Deleted {
		return
	}
	r.tree[c.Path], r.sum = c.Size, r.sum+c.Size
	if !r.blobs[c.Blob] {
		r.blobs[c.Blob], r.repo = true, r.repo+c.Size
	}
}

// AttachmentRef is an attachment with the Markdown of the document it belongs to.
type AttachmentRef struct {
	Doc    Doc    // the attachment
	Parent Doc    // its page or note
	Name   string // file name in the attachments folder
	Body   string // the parent's Markdown
}

// File is a file a report lists by size.
type File struct {
	Doc   Doc
	Name  string
	Bytes int64
}

// Orphans are attachments their document does not link to, largest first.
func Orphans(atts []AttachmentRef) []File {
	var out []File
	for _, a := range atts {
		if referenced(a.Body, a.Name) {
			continue
		}
		out = append(out, File{Doc: a.Doc, Name: a.Name, Bytes: a.Doc.Bytes})
	}
	slices.SortFunc(out, func(a, b File) int { return cmp.Compare(b.Bytes, a.Bytes) })
	return out
}

// referenced tells whether body links to the attachment, plain or URL-escaped.
func referenced(body, name string) bool {
	for _, form := range []string{name, url.PathEscape(name), strings.ReplaceAll(name, " ", "%20")} {
		if strings.Contains(body, "attachments/"+form) {
			return true
		}
	}
	return false
}

// Largest are the n largest docs.
func Largest(docs []Doc, n int) []Doc {
	sorted := slices.Clone(docs)
	slices.SortStableFunc(sorted, func(a, b Doc) int { return cmp.Compare(b.Bytes, a.Bytes) })
	return sorted[:min(n, len(sorted))]
}
