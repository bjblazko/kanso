// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/selfcare"
)

type focusPage struct {
	// Tips are Selfcare's first-aid steps; a break shows one at random.
	Tips []careTip
	// Saved is the note the parked thoughts went into, after saving.
	Saved string
}

// focusPage is the timer. It runs in the browser only: settings and the
// running phase stay on the device, the gong and the notification come from
// this tab while it is open (ADR-021).
func (s *Server) focusPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("focus.title")
	d := focusPage{Saved: r.URL.Query().Get("saved")}
	for _, t := range selfcare.Tips() {
		d.Tips = append(d.Tips, tipIn(t, v.Lang))
	}
	v.Data = d
	s.render(w, r, http.StatusOK, "focus", v)
}

// focusNote keeps the thoughts parked during a round as a private note.
func (s *Server) focusNote(w http.ResponseWriter, r *http.Request) {
	text := strings.TrimSpace(r.PostFormValue("text"))
	if text == "" {
		http.Redirect(w, r, "/focus", http.StatusSeeOther)
		return
	}
	v := s.newView(r, "", nil)
	u, _ := auth.UserFrom(r.Context())
	body := v.T("focus.park_title") + " – " + time.Now().Format("02.01.2006 15:04") + "\n\n" + text + "\n"
	n, _, err := s.Notes.Create(u.Sub, "", body, nil, 0)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/focus?saved="+n.Meta.ID, http.StatusSeeOther)
}
