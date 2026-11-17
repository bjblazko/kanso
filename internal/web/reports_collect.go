// SPDX-License-Identifier: Apache-2.0

package web

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/tasks"
	"github.com/bjblazko/kanso/internal/todos"
	"github.com/bjblazko/kanso/internal/wiki"
)

// reportScope is whose documents a report counts.
type reportScope string

const (
	scopeOwn    reportScope = "own"    // what belongs to the viewer
	scopeShared reportScope = "shared" // what the viewer may read and is someone else's
	scopeSeen   reportScope = "seen"   // both
	scopeAll    reportScope = "all"    // everything; admins only
)

// reportScopes are the scopes in the order the page offers them.
var reportScopes = []reportScope{scopeOwn, scopeShared, scopeSeen, scopeAll}

// reportCtx is what collecting the documents of one report needs.
type reportCtx struct {
	v     view
	p     access.Principal
	g     access.Groups
	scope reportScope
	scan  store.Scan
	added map[string]added // path → who added it and when
}

// candidate is a document before the scope decides on it.
type candidate struct {
	doc      reports.Doc
	readable bool
}

// added is who first put a file into the repository, and when.
type added struct {
	sub  string
	when time.Time
}

// addedBy follows the history: a file's person is whoever added it first; a
// move keeps the person of the file it came from.
func (s *Server) addedBy(sc store.Scan) map[string]added {
	byEmail := map[string]string{}
	for _, u := range s.Users.List() {
		if u.Email != "" {
			byEmail[strings.ToLower(u.Email)] = u.Sub
		}
	}
	out := map[string]added{}
	for _, c := range sc.Changes {
		if c.Deleted {
			continue
		}
		if prev, ok := out[c.From]; ok && c.From != "" {
			out[c.Path] = prev
		} else if _, ok := out[c.Path]; !ok {
			out[c.Path] = added{sub: byEmail[strings.ToLower(c.Email)], when: c.When}
		}
	}
	return out
}

// reportDocs are the documents the scope covers, titles cleared where the viewer may not read.
func (s *Server) reportDocs(rc reportCtx) []reports.Doc {
	var all []candidate
	all = append(all, s.pageCandidates(rc)...)
	all = append(all, s.noteCandidates(rc)...)
	all = append(all, s.listCandidates(rc)...)
	all = append(all, s.taskCandidates(rc)...)
	var out []reports.Doc
	for _, c := range all {
		if !inScope(rc, c) {
			continue
		}
		if !c.readable {
			c.doc.Title, c.doc.Href = "", ""
		}
		out = append(out, c.doc)
	}
	return out
}

func inScope(rc reportCtx, c candidate) bool {
	own := c.doc.Person == rc.p.Sub
	switch rc.scope {
	case scopeOwn:
		return own
	case scopeShared:
		return c.readable && !own
	case scopeSeen:
		return own || c.readable
	}
	return true
}

// size is the file's size at HEAD, else on disk.
func (s *Server) size(rc reportCtx, rel string) int64 {
	if n, ok := rc.scan.Live[filepath.ToSlash(rel)]; ok {
		return n
	}
	if fi, err := os.Stat(filepath.Join(s.Cfg.DataDir, rel)); err == nil {
		return fi.Size()
	}
	return 0
}

func (s *Server) pageCandidates(rc reportCtx) []candidate {
	var out []candidate
	for _, id := range s.Wiki.IDs() {
		pg, ok := s.Wiki.ByID(id)
		if !ok {
			continue
		}
		readable := access.Can(s.pageRole(rc.p, pg.Space, id, rc.g), access.Read)
		kind := reports.Page
		if pg.Meta.IsRecipe() {
			kind = reports.Recipe
		}
		path := wiki.PagePath(pg)
		d := reports.Doc{Kind: kind, ID: id, Title: pg.Meta.Title, Href: "/p/" + id, Person: rc.added[path].sub,
			Space: pg.Space, Path: path, Created: pg.Meta.Created, Updated: pg.Meta.Updated, Bytes: s.size(rc, path)}
		out = append(out, candidate{d, readable})
		out = append(out, s.attachmentCandidates(rc, pg, readable)...)
	}
	return out
}

func (s *Server) attachmentCandidates(rc reportCtx, pg wiki.Page, readable bool) []candidate {
	var out []candidate
	for _, name := range s.Wiki.ListAttachments(pg.Meta.ID) {
		path := wiki.AttachmentPath(pg, name)
		a := rc.added[path]
		created := a.when
		if created.IsZero() {
			created = pg.Meta.Created
		}
		d := reports.Doc{Kind: reports.Attachment, ID: pg.Meta.ID + "/" + name, Title: name,
			Href: "/p/" + pg.Meta.ID + "/attachments/" + name, Person: a.sub, Space: pg.Space, Path: path,
			Created: created, Updated: created, Bytes: s.size(rc, path)}
		out = append(out, candidate{d, readable})
	}
	return out
}

func (s *Server) noteCandidates(rc reportCtx) []candidate {
	var out []candidate
	for _, n := range s.Notes.All() {
		readable := n.Owner == rc.p.Sub // notes are always private
		rel := filepath.Join(notes.Dir, n.Owner, n.Meta.ID)
		d := reports.Doc{Kind: reports.Note, ID: n.Meta.ID, Title: noteTitle(rc.v, n), Href: "/notes/" + n.Meta.ID,
			Person: n.Owner, Space: "notes", Path: filepath.Join(rel, "index.md"), Created: n.Meta.Created,
			Updated: n.Meta.Updated, Bytes: int64(len(n.Body))}
		out = append(out, candidate{d, readable})
		for _, name := range n.Attachments {
			path := filepath.Join(rel, "attachments", name)
			a := reports.Doc{Kind: reports.Attachment, ID: n.Meta.ID + "/" + name, Title: name,
				Href: "/notes/" + n.Meta.ID + "/attachments/" + name, Person: n.Owner, Space: "notes", Path: path,
				Created: n.Meta.Created, Updated: n.Meta.Updated, Bytes: s.size(rc, path)}
			out = append(out, candidate{a, readable})
		}
	}
	return out
}

func (s *Server) listCandidates(rc reportCtx) []candidate {
	visible := map[string]bool{}
	for _, l := range s.Todos.Visible(rc.p.Sub, rc.g) {
		visible[l.ID] = true
	}
	var out []candidate
	for _, l := range s.Todos.All() {
		path := filepath.Join(todos.Dir, l.ID+".json")
		d := reports.Doc{Kind: reports.List, ID: l.ID, Title: l.Name, Href: "/todos/" + l.ID, Person: l.Owner,
			Space: "lists", Path: path, Created: l.Created, Updated: l.Updated, Bytes: s.size(rc, path)}
		out = append(out, candidate{d, visible[l.ID]})
	}
	return out
}

func (s *Server) taskCandidates(rc reportCtx) []candidate {
	visible := map[string]bool{}
	viewer := tasks.Actor{Sub: rc.p.Sub, Groups: rc.g}
	for _, b := range s.Tasks.AllBoards() {
		visible[b.ID] = viewer.Sees(b)
	}
	var out []candidate
	for _, t := range s.Tasks.AllTasks() {
		path := tasks.TaskPath(t.Board, t.Meta.ID)
		d := reports.Doc{Kind: reports.Task, ID: t.Meta.ID, Title: t.Meta.Title, Href: "/tasks/b/" + t.Board + "/" + t.Meta.ID,
			Person: t.Meta.Author, Space: "boards", Path: path, Created: t.Meta.Created, Updated: t.Meta.Updated,
			Bytes: s.size(rc, path)}
		out = append(out, candidate{d, visible[t.Board]})
	}
	return out
}
