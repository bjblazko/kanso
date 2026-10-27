// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/wiki"
)

// pageCtx is a page together with the caller's standing in its space.
type pageCtx struct {
	Page      wiki.Page
	Space     spaces.Space
	Role      access.Role
	Ancestors []wiki.PageMeta
	Children  []*wiki.Node
	Targets   []moveTarget
	Files     []string
	HTML      template.HTML
	Info      markdown.Info
	LockedBy  string
}

// CanBreakLock tells templates whether to offer breaking an edit lock.
func (c pageCtx) CanBreakLock() bool { return access.Can(c.Role, access.BreakLock) }

// CanWrite tells templates whether to offer changes.
func (c pageCtx) CanWrite() bool { return access.Can(c.Role, access.Write) }

type moveTarget struct {
	ID, Label string
}

// pageFor loads {id} and checks that the caller may do action; it answers
// 404 for pages the caller cannot see and 403 for missing rights.
func (s *Server) pageFor(w http.ResponseWriter, r *http.Request, action access.Action) (pageCtx, bool) {
	p, ok := s.Wiki.ByID(r.PathValue("id"))
	sp, spOK := s.Spaces.Get(p.Space)
	_, who := auth.UserFrom(r.Context())
	role := s.Spaces.RoleOf(who, p.Space, s.groups())
	if !ok || !spOK || !access.Can(role, access.Read) {
		s.notFound(w, r)
		return pageCtx{}, false
	}
	if !access.Can(role, action) {
		s.forbidden(w, r)
		return pageCtx{}, false
	}
	return pageCtx{Page: p, Space: sp, Role: role}, true
}

func (s *Server) showPage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok {
		return
	}
	if err := s.fillPageView(&pc); err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.inSpace(s.newView(r, pc.Page.Meta.Title, pc), pc.Space.Slug, pc.Page.Meta.ID)
	s.render(w, r, http.StatusOK, "page_view", v)
}

// fillPageView adds what the read view shows around the page text.
func (s *Server) fillPageView(pc *pageCtx) error {
	html, info, err := s.MD.Render([]byte(pc.Page.Body), markdown.PageCtx{Base: "/p/" + pc.Page.Meta.ID})
	if err != nil {
		return err
	}
	pc.HTML, pc.Info = html, info
	pc.Files = s.Wiki.ListAttachments(pc.Page.Meta.ID)
	pc.Ancestors = s.Wiki.Ancestors(pc.Page.Meta.ID)
	pc.Children = childrenOf(s.Wiki.Tree(pc.Space.Slug), pc.Page.Meta.ID)
	if pc.CanWrite() {
		pc.Targets = s.moveTargets(*pc)
	}
	return nil
}

func childrenOf(nodes []*wiki.Node, id string) []*wiki.Node {
	for _, n := range nodes {
		if n.Meta.ID == id {
			return n.Children
		}
		if c := childrenOf(n.Children, id); c != nil {
			return c
		}
	}
	return nil
}

// moveTargets lists the places a page may move to: top level or any page
// outside its own subtree.
func (s *Server) moveTargets(pc pageCtx) []moveTarget {
	targets := []moveTarget{{ID: "", Label: pc.Space.Name}}
	var walk func(nodes []*wiki.Node, prefix string)
	walk = func(nodes []*wiki.Node, prefix string) {
		for _, n := range nodes {
			if n.Meta.ID == pc.Page.Meta.ID {
				continue
			}
			targets = append(targets, moveTarget{ID: n.Meta.ID, Label: prefix + n.Meta.Title})
			walk(n.Children, prefix+n.Meta.Title+" / ")
		}
	}
	walk(s.Wiki.Tree(pc.Space.Slug), pc.Space.Name+" / ")
	return targets
}

type newPageForm struct {
	Space  spaces.Space
	Parent string
	Error  string
}

func (s *Server) newPageForm(w http.ResponseWriter, r *http.Request) {
	sp, role, ok := s.spaceFor(w, r)
	if !ok {
		return
	}
	if !access.Can(role, access.Write) {
		s.forbidden(w, r)
		return
	}
	v := s.newView(r, "", newPageForm{Space: sp, Parent: r.URL.Query().Get("parent")})
	s.render(w, r, http.StatusOK, "page_new", s.inSpace(v, sp.Slug, ""))
}

func (s *Server) createPage(w http.ResponseWriter, r *http.Request) {
	sp, role, ok := s.spaceFor(w, r)
	if !ok {
		return
	}
	if !access.Can(role, access.Write) {
		s.forbidden(w, r)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	p, err := s.Wiki.Create(u.Author(), sp.Slug, r.PostFormValue("parent"), r.PostFormValue("title"))
	if err != nil {
		v := s.newView(r, "", nil)
		v.Data = newPageForm{Space: sp, Parent: r.PostFormValue("parent"), Error: v.T(pageErrorKey(err))}
		s.render(w, r, http.StatusBadRequest, "page_new", s.inSpace(v, sp.Slug, ""))
		return
	}
	http.Redirect(w, r, "/p/"+p.Meta.ID, http.StatusSeeOther)
}

func pageErrorKey(err error) string {
	switch {
	case errors.Is(err, wiki.ErrNoTitle):
		return "page.title_required"
	case errors.Is(err, wiki.ErrHasChildren):
		return "page.has_children"
	case errors.Is(err, wiki.ErrBadMove):
		return "page.bad_move"
	}
	return "error.generic"
}

func (s *Server) movePage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	order, err := strconv.Atoi(r.PostFormValue("order"))
	if err != nil {
		order = pc.Page.Meta.Order
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Wiki.Move(u.Author(), pc.Page.Meta.ID, r.PostFormValue("parent"), order); err != nil {
		s.fail(w, r, http.StatusConflict, pageErrorKey(err))
		return
	}
	http.Redirect(w, r, "/p/"+pc.Page.Meta.ID, http.StatusSeeOther)
}

func (s *Server) confirmDeletePage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	v := s.inSpace(s.newView(r, pc.Page.Meta.Title, pc), pc.Space.Slug, pc.Page.Meta.ID)
	s.render(w, r, http.StatusOK, "page_delete", v)
}

func (s *Server) deletePage(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Wiki.Delete(u.Author(), pc.Page.Meta.ID); err != nil {
		s.fail(w, r, http.StatusConflict, pageErrorKey(err))
		return
	}
	http.Redirect(w, r, "/s/"+pc.Space.Slug, http.StatusSeeOther)
}

// preview renders Markdown for the editor; the caller must be allowed to
// write the page the text belongs to.
func (s *Server) preview(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("id", r.PostFormValue("page"))
	if _, ok := s.pageFor(w, r, access.Write); !ok {
		return
	}
	html, _, err := s.MD.Render([]byte(r.PostFormValue("body")), markdown.PageCtx{Base: "/p/" + r.PathValue("id")})
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
