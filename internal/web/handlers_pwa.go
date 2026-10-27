// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/bjblazko/kanso/internal/auth"
)

// Colours of the light theme (tokens.css) for the browser chrome.
const (
	themeBackground = "#F4F4F0"
	themeColor      = "#1E1F21"
)

// precacheList is what the service worker keeps on the device: the files the
// capture page needs, with the same fingerprinted URLs the pages use.
func (s *Server) precacheList() ([]string, error) {
	v := view{version: s.version}
	list := []string{v.Asset("css/fonts.css"), v.Asset("css/tokens.css"), v.Asset("css/app.css"),
		v.Asset("js/nav.js"), v.Asset("js/capture.js"), v.Asset("icons/icon.svg"), v.Asset("icons/apple-touch-icon.png")}
	fonts, err := fs.Glob(s.static, "fonts/*.woff2")
	for _, f := range fonts {
		list = append(list, "/static/"+f) // fonts.css loads them without a version
	}
	return list, err
}

// serviceWorker serves js/sw.js with the asset version and list in front.
func (s *Server) serviceWorker(w http.ResponseWriter, r *http.Request) {
	src, err := fs.ReadFile(s.static, "js/sw.js")
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	list, err := s.precacheList()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	precache, _ := json.Marshal(list)
	h := w.Header()
	h.Set("Content-Type", "text/javascript; charset=utf-8")
	h.Set("Cache-Control", "no-cache")
	h.Set("Service-Worker-Allowed", "/")
	_, _ = fmt.Fprintf(w, "const VERSION = %q;\nconst PRECACHE = %s;\n%s", s.version, precache, src)
}

type manifestIcon struct {
	Src     string `json:"src"`
	Sizes   string `json:"sizes"`
	Type    string `json:"type"`
	Purpose string `json:"purpose"`
}

// manifest makes Kanso installable; the home-screen icon opens the capture page.
func (s *Server) manifest(w http.ResponseWriter, _ *http.Request) {
	v := view{version: s.version}
	m := map[string]any{
		"name": "Hüpattl! Kanso", "short_name": "Kanso", "start_url": "/notes/new", "scope": "/",
		"display": "standalone", "background_color": themeBackground, "theme_color": themeColor,
		"icons": []manifestIcon{
			{Src: v.Asset("icons/icon-192.png"), Sizes: "192x192", Type: "image/png", Purpose: "any maskable"},
			{Src: v.Asset("icons/icon-512.png"), Sizes: "512x512", Type: "image/png", Purpose: "any maskable"},
		},
	}
	w.Header().Set("Content-Type", "application/manifest+json")
	w.Header().Set("Cache-Control", "no-cache")
	_ = json.NewEncoder(w).Encode(m)
}

// noteToken gives the offline capture page a fresh CSRF token; the cached
// page may carry one from an older session.
func (s *Server) noteToken(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"csrf": auth.CSRFToken(r.Context())})
}
