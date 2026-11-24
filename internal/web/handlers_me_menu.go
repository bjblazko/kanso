// SPDX-License-Identifier: Apache-2.0

package web

import "net/http"

// menuPage lets the person choose and order the rail's parts on this device;
// menu-settings.js keeps the choice in the browser (ADR-020).
func (s *Server) menuPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("menu.title")
	s.render(w, r, http.StatusOK, "me_menu", v)
}
