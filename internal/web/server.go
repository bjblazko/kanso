// SPDX-License-Identifier: Apache-2.0

// Package web serves Kanso's pages.
package web

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/config"
	"github.com/bjblazko/kanso/internal/i18n"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/search"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
	"github.com/bjblazko/kanso/internal/wiki"
	assets "github.com/bjblazko/kanso/web"
)

// Deps are the services the web layer uses.
type Deps struct {
	Cfg    config.Config
	Auth   *auth.Auth
	I18n   *i18n.Bundle
	Users  *users.Store
	Repo   *store.Repo
	Spaces *spaces.Service
	Wiki   *wiki.Wiki
	MD     *markdown.Renderer
	Locks  *wiki.Locks
	Search *search.Index
}

// Server renders pages and routes requests.
type Server struct {
	Deps
	pages    map[string]*template.Template
	static   fs.FS
	version  string
	licenses []license
}

// New parses templates and prepares static files.
func New(d Deps) (*Server, error) {
	static, err := fs.Sub(assets.Files, "static")
	if err != nil {
		return nil, err
	}
	s := &Server{Deps: d, static: static}
	if s.version, err = hashFS(static); err != nil {
		return nil, err
	}
	if s.pages, err = parsePages(); err != nil {
		return nil, err
	}
	if s.licenses, err = loadLicenses(static); err != nil {
		return nil, err
	}
	return s, nil
}

var pageNames = []string{"home", "error", "licenses", "space", "space_new", "space_settings", "admin_groups", "page_view", "page_new", "page_delete", "page_edit", "page_history", "page_revision", "search"}

func parsePages() (map[string]*template.Template, error) {
	pages := map[string]*template.Template{}
	for _, name := range pageNames {
		t, err := template.New(name).Funcs(templateFuncs).ParseFS(assets.Files,
			"templates/layout.html", "templates/icons.html", "templates/tree.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("template %s: %w", name, err)
		}
		pages[name] = t
	}
	return pages, nil
}

type treeArgs struct {
	Nodes   []*wiki.Node
	Current string
}

var templateFuncs = template.FuncMap{
	"treeArgs": func(nodes []*wiki.Node, current string) treeArgs { return treeArgs{nodes, current} },
}

// hashFS fingerprints the static files for cache-busting URLs.
func hashFS(fsys fs.FS) (string, error) {
	h := sha256.New()
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		h.Write([]byte(p))
		h.Write(b)
		return err
	})
	return hex.EncodeToString(h.Sum(nil))[:12], err
}

// Handler returns the complete HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	s.Auth.Routes(mux)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "ok") })
	mux.Handle("GET /static/", http.StripPrefix("/static/", cacheForever(http.FileServerFS(s.static))))
	app := http.NewServeMux()
	s.routes(app)
	mux.Handle("/", s.Auth.RequireUser(app))
	return securityHeaders(mux)
}

func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /about/licenses", s.showLicenses)
	mux.HandleFunc("POST /me/lang", s.setLang)
	mux.HandleFunc("GET /search", s.search)
	mux.HandleFunc("GET /s/{space}", s.showSpace)
	mux.HandleFunc("GET /s/{space}/settings", s.spaceSettings)
	mux.HandleFunc("POST /s/{space}/settings", s.updateSpaceSettings)
	mux.HandleFunc("GET /s/{space}/new", s.newPageForm)
	mux.HandleFunc("POST /s/{space}/new", s.createPage)
	mux.HandleFunc("GET /p/{id}", s.showPage)
	mux.HandleFunc("POST /api/preview", s.preview)
	mux.HandleFunc("GET /p/{id}/edit", s.editPage)
	mux.HandleFunc("POST /p/{id}", s.savePage)
	mux.HandleFunc("POST /p/{id}/lock/heartbeat", s.heartbeat)
	mux.HandleFunc("POST /p/{id}/lock/break", s.breakLock)
	mux.HandleFunc("GET /p/{id}/history", s.showHistory)
	mux.HandleFunc("GET /p/{id}/history/{hash}", s.showRevision)
	mux.HandleFunc("POST /p/{id}/history/{hash}/restore", s.restoreRevision)
	mux.HandleFunc("POST /p/{id}/attachments", s.uploadAttachment)
	mux.HandleFunc("GET /p/{id}/attachments/{name}", s.serveAttachment)
	mux.HandleFunc("POST /p/{id}/attachments/{name}/delete", s.deleteAttachment)
	mux.HandleFunc("POST /p/{id}/move", s.movePage)
	mux.HandleFunc("GET /p/{id}/delete", s.confirmDeletePage)
	mux.HandleFunc("POST /p/{id}/delete", s.deletePage)
	mux.HandleFunc("GET /admin/spaces/new", s.newSpaceForm)
	mux.HandleFunc("POST /admin/spaces/new", s.createSpace)
	mux.HandleFunc("GET /admin/groups", s.groupsPage)
	mux.HandleFunc("POST /admin/groups", s.updateGroups)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { s.notFound(w, r) })
}

const csp = "default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; " +
	"object-src 'none'; base-uri 'none'; frame-ancestors 'none'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", csp)
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

// cacheForever serves files with a long cache lifetime and hides directories.
func cacheForever(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}
