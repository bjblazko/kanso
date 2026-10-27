// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/wiki"
)

type notePromote struct {
	noteView
	Targets []moveTarget // ID is "<space>/<parent page id>"
	Error   string
}

func (s *Server) promoteNoteForm(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", nil)
	data := notePromote{noteView: noteView{Note: n, Title: noteTitle(v, n)}, Targets: s.promoteTargets(r)}
	if len(data.Targets) == 0 {
		data.Error = v.T("notes.promote_nowhere")
	}
	v.Title, v.Data, v.Here = v.T("notes.promote"), data, notesHere
	s.render(w, r, http.StatusOK, "note_promote", v)
}

// promoteTargets lists every space the person may write and its pages.
func (s *Server) promoteTargets(r *http.Request) []moveTarget {
	_, p := auth.UserFrom(r.Context())
	g := s.groups()
	var targets []moveTarget
	for _, sp := range s.Spaces.Visible(p, g) {
		if !access.Can(s.Spaces.RoleOf(p, sp.Slug, g), access.Write) {
			continue
		}
		targets = append(targets, moveTarget{ID: sp.Slug + "/", Label: sp.Name})
		var walk func(nodes []*wiki.Node, prefix string)
		walk = func(nodes []*wiki.Node, prefix string) {
			for _, n := range nodes {
				targets = append(targets, moveTarget{ID: sp.Slug + "/" + n.Meta.ID, Label: prefix + n.Meta.Title})
				walk(n.Children, prefix+n.Meta.Title+" / ")
			}
		}
		walk(s.Wiki.Tree(sp.Slug), sp.Name+" / ")
	}
	return targets
}

// promoteNote moves the note into the wiki as a new page.
func (s *Server) promoteNote(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	u, p := auth.UserFrom(r.Context())
	space, parent, found := strings.Cut(r.PostFormValue("where"), "/")
	if !found || !access.Can(s.Spaces.RoleOf(p, space, s.groups()), access.Write) {
		s.fail(w, r, http.StatusForbidden, "error.forbidden")
		return
	}
	var page wiki.Page
	err := s.Notes.MoveOut(n.Owner, n.Meta.ID, func(dir string) error {
		var err error
		page, err = s.Wiki.Adopt(u.Author(), space, parent, r.PostFormValue("title"), dir)
		return err
	})
	switch {
	case errors.Is(err, wiki.ErrNoTitle):
		s.fail(w, r, http.StatusBadRequest, "page.title_required")
	case errors.Is(err, wiki.ErrNotFound):
		s.fail(w, r, http.StatusBadRequest, "notes.promote_parent_gone")
	case err != nil:
		s.serverError(w, r, err)
	default:
		http.Redirect(w, r, "/p/"+page.Meta.ID, http.StatusSeeOther)
	}
}
