// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/wiki"
)

type editPage struct {
	pageCtx
	Lock string
}

func (s *Server) editPage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	lk, err := s.Locks.Acquire(pc.Page.Meta.ID, u.Sub, u.Name)
	var locked *wiki.ErrLocked
	if errors.As(err, &locked) {
		s.showLocked(w, r, pc, locked.Holder)
		return
	}
	v := s.inSpace(s.newView(r, pc.Page.Meta.Title, editPage{pageCtx: pc, Lock: lk.Token}), pc.Space.Slug, pc.Page.Meta.ID)
	s.render(w, r, http.StatusOK, "page_edit", v)
}

// showLocked renders the page read-only with a note on who is editing it.
func (s *Server) showLocked(w http.ResponseWriter, r *http.Request, pc pageCtx, holder wiki.Lock) {
	if err := s.fillPageView(&pc); err != nil {
		s.serverError(w, r, err)
		return
	}
	pc.LockedBy = holder.Name
	s.render(w, r, http.StatusOK, "page_view", s.pageView(r, pc))
}

// savePage stores the editor's text if the lock still holds. Scripts get
// JSON; plain form posts are redirected.
func (s *Server) savePage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	id, token := pc.Page.Meta.ID, r.PostFormValue("lock")
	if s.Locks.Check(id, token) != nil {
		s.saveFailed(w, r, http.StatusConflict, "lock_lost")
		return
	}
	u, _ := auth.UserFrom(r.Context())
	body := strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n")
	if err := s.Wiki.Save(u.Author(), id, r.PostFormValue("title"), body); err != nil {
		s.saveFailed(w, r, http.StatusBadRequest, "invalid")
		return
	}
	if r.PostFormValue("release") != "" {
		s.Locks.Release(id, token)
	}
	if wantsJSON(r) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
		return
	}
	http.Redirect(w, r, "/p/"+id, http.StatusSeeOther)
}

func (s *Server) saveFailed(w http.ResponseWriter, r *http.Request, status int, code string) {
	if wantsJSON(r) {
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	key := "page.lock_lost"
	if code != "lock_lost" {
		key = "page.title_required"
	}
	s.fail(w, r, status, key)
}

func wantsJSON(r *http.Request) bool {
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) heartbeat(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	if err := s.Locks.Heartbeat(pc.Page.Meta.ID, r.PostFormValue("lock")); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "lock_lost"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) breakLock(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.BreakLock)
	if !ok {
		return
	}
	s.Locks.Break(pc.Page.Meta.ID)
	http.Redirect(w, r, "/p/"+pc.Page.Meta.ID+"/edit", http.StatusSeeOther)
}
