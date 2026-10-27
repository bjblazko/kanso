// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"regexp"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

const historyLimit = 200

var hashPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

type historyPage struct {
	pageCtx
	Revisions []store.Revision
}

type revisionPage struct {
	pageCtx
	Revision store.Revision
	Old      template.HTML
	Diff     []wiki.DiffLine
	IsLatest bool
}

func (s *Server) showHistory(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok {
		return
	}
	revs, err := s.Wiki.History(pc.Page.Meta.ID, historyLimit)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, pc.Page.Meta.Title, historyPage{pageCtx: pc, Revisions: revs})
	s.render(w, r, http.StatusOK, "page_history", s.inSpace(v, pc.Space.Slug, pc.Page.Meta.ID))
}

// hashFor validates {hash}; it answers 400 itself for malformed values.
func (s *Server) hashFor(w http.ResponseWriter, r *http.Request) (string, bool) {
	hash := r.PathValue("hash")
	if !hashPattern.MatchString(hash) {
		s.fail(w, r, http.StatusBadRequest, "history.bad_revision")
		return "", false
	}
	return hash, true
}

func (s *Server) showRevision(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	hash, hashOK := "", ok
	if ok {
		hash, hashOK = s.hashFor(w, r)
	}
	if !hashOK {
		return
	}
	data, err := s.revisionData(pc, hash)
	if errors.Is(err, wiki.ErrNotFound) {
		s.notFound(w, r)
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, pc.Page.Meta.Title, data)
	s.render(w, r, http.StatusOK, "page_revision", s.inSpace(v, pc.Space.Slug, pc.Page.Meta.ID))
}

// revisionData renders the old text and its difference to the current one.
func (s *Server) revisionData(pc pageCtx, hash string) (revisionPage, error) {
	revs, err := s.Wiki.History(pc.Page.Meta.ID, historyLimit)
	if err != nil || len(revs) == 0 {
		return revisionPage{}, wiki.ErrNotFound
	}
	old, err := s.Wiki.Revision(pc.Page.Meta.ID, hash)
	if err != nil {
		return revisionPage{}, err
	}
	data := revisionPage{pageCtx: pc}
	for _, rev := range revs {
		if len(rev.Hash) >= len(hash) && rev.Hash[:len(hash)] == hash {
			data.Revision = rev
		}
	}
	data.IsLatest = data.Revision.Hash == revs[0].Hash
	data.Old, _, err = s.MD.Render([]byte(old.Body), markdown.PageCtx{Base: "/p/" + pc.Page.Meta.ID})
	if err != nil {
		return data, err
	}
	data.Diff, err = s.Wiki.Diff(pc.Page.Meta.ID, data.Revision.Hash, revs[0].Hash)
	return data, err
}

// restoreRevision takes the edit lock for a moment, so a restore never
// overwrites someone who is editing right now.
func (s *Server) restoreRevision(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	hash, hashOK := "", ok
	if ok {
		hash, hashOK = s.hashFor(w, r)
	}
	if !hashOK {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	lk, err := s.Locks.Acquire(pc.Page.Meta.ID, u.Sub, u.Name)
	if err != nil {
		s.fail(w, r, http.StatusConflict, "history.locked")
		return
	}
	defer s.Locks.Release(pc.Page.Meta.ID, lk.Token)
	if err := s.Wiki.Restore(u.Author(), pc.Page.Meta.ID, hash); err != nil {
		s.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/p/"+pc.Page.Meta.ID, http.StatusSeeOther)
}
