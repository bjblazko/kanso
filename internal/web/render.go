// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/i18n"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/users"
	"github.com/bjblazko/kanso/internal/wiki"
)

// view is what every template receives.
type view struct {
	Title string
	Lang  string
	User  users.User
	CSRF  string
	Query string
	Data  any
	Admin bool
	Nav   []spaces.Space
	Here  string // slug of the current space, for aria-current
	Page  string // id of the current page, for aria-current
	Tree  []*wiki.Node
	// Module is the part of Kanso the page belongs to ("notes", "wiki"), "" for
	// pages outside all parts (search, administration, licenses).
	Module string
	Dev    bool   // dev login: offer to switch person
	Path   string // request URI, to come back after switching

	bundle  *i18n.Bundle
	version string
}

// T translates key into the page language.
func (v view) T(key string, args ...any) string { return v.bundle.T(v.Lang, key, args...) }

// Asset returns a cache-busting URL for a static file.
func (v view) Asset(path string) string { return "/static/" + path + "?v=" + v.version }

func (s *Server) newView(r *http.Request, title string, data any) view {
	u, p := auth.UserFrom(r.Context())
	return view{
		Title: title, Lang: i18n.Negotiate(r.Header.Get("Accept-Language"), u.Lang),
		User: u, CSRF: auth.CSRFToken(r.Context()), Query: r.URL.Query().Get("q"), Data: data,
		Admin: p.Admin, Nav: s.Spaces.Visible(p, s.groups()),
		Module: moduleOf(r.URL.Path), Dev: s.Cfg.DevLogin, Path: r.URL.RequestURI(),
		bundle: s.I18n, version: s.version,
	}
}

// render executes a page into a buffer first, so errors never leave half a page.
func (s *Server) render(w http.ResponseWriter, r *http.Request, status int, page string, v view) {
	var buf bytes.Buffer
	if err := s.pages[page].ExecuteTemplate(&buf, "layout", v); err != nil {
		slog.Error("render", "page", page, "err", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (s *Server) page(w http.ResponseWriter, r *http.Request, page, title string, data any) {
	s.render(w, r, http.StatusOK, page, s.newView(r, title, data))
}

// inSpace marks the navigation position: the space, its tree, the page.
func (s *Server) inSpace(v view, space, pageID string) view {
	v.Here, v.Page, v.Tree = space, pageID, s.Wiki.Tree(space)
	return v
}

func (s *Server) fail(w http.ResponseWriter, r *http.Request, status int, key string) {
	v := s.newView(r, "", nil)
	v.Data = v.T(key)
	v.Title = v.T(key)
	s.render(w, r, status, "error", v)
}

func (s *Server) notFound(w http.ResponseWriter, r *http.Request) {
	s.fail(w, r, http.StatusNotFound, "error.not_found")
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request) {
	s.fail(w, r, http.StatusForbidden, "error.forbidden")
}

func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("request failed", "path", r.URL.Path, "err", err)
	s.fail(w, r, http.StatusInternalServerError, "error.generic")
}

var errBadRequest = errors.New("bad request")

// moduleOf names the part of Kanso a path belongs to.
func moduleOf(path string) string {
	switch {
	case path == "/notes" || strings.HasPrefix(path, "/notes/"):
		return "notes"
	case path == "/", strings.HasPrefix(path, "/s/"), strings.HasPrefix(path, "/p/"), strings.HasPrefix(path, "/admin/spaces/"):
		return "wiki"
	}
	return ""
}
