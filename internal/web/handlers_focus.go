// SPDX-License-Identifier: Apache-2.0

package web

import "net/http"

// focusPage is the Pomodoro timer. It runs in the browser only: settings and
// the running phase stay on the device, the sound and the notification come
// from this tab while it is open (ADR-021).
func (s *Server) focusPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("focus.title")
	s.render(w, r, http.StatusOK, "focus", v)
}
