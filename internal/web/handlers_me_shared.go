// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/todos"
)

// sharedItem is one thing someone else sees too, with who that is.
type sharedItem struct {
	Name, Href string
	Audience   audience
}

// sharedOverview is "Was andere sehen", by kind; notes are always private and never appear.
type sharedOverview struct {
	Spaces, Pages, Lists, Boards, Plans []sharedItem
}

// Empty tells whether nothing is visible to anyone else.
func (o sharedOverview) Empty() bool {
	return len(o.Spaces)+len(o.Pages)+len(o.Lists)+len(o.Boards)+len(o.Plans) == 0
}

// sharedArgs feeds one section of the overview to its template.
type sharedArgs struct {
	Items   []sharedItem
	Heading string // i18n key
	V       view
}

// add keeps an item only when someone besides the viewer sees it.
func add(items []sharedItem, name, href string, a audience) []sharedItem {
	if a.Private {
		return items
	}
	return append(items, sharedItem{Name: name, Href: href, Audience: a})
}

func (s *Server) sharedPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("me.shared")
	_, p := auth.UserFrom(r.Context())
	g := s.groups()
	o := sharedOverview{}
	for _, sp := range s.Spaces.Visible(p, g) {
		o.Spaces = add(o.Spaces, sp.Name, "/s/"+sp.Slug, s.spaceAudience(v, sp))
	}
	o.Pages = s.sharedPages(v, p, g)
	for _, l := range s.Todos.Visible(p.Sub, g) {
		o.Lists = add(o.Lists, l.Name, "/todos/"+l.ID, s.listAudience(v, l))
	}
	if boards, err := s.Tasks.Boards(s.actor(r)); err == nil {
		for _, b := range boards {
			o.Boards = add(o.Boards, b.Name, "/tasks/b/"+b.ID, s.boardAudience(v, b))
		}
	}
	for _, pl := range s.MealPlans.For(g, p.Sub) {
		o.Plans = add(o.Plans, pl.Name, "/mealplan/"+pl.ID, s.membersAudience(v, []access.Member{{Group: pl.Group, Role: access.Viewer}}))
	}
	v.Data = o
	s.render(w, r, http.StatusOK, "me_shared", v)
}

// sharedPages are the pages with shares of their own that p may read.
func (s *Server) sharedPages(v view, p access.Principal, g access.Groups) []sharedItem {
	var out []sharedItem
	for _, pg := range s.Wiki.Shared() {
		role := s.pageRole(p, pg.Space, pg.Meta.ID, g)
		sp, ok := s.Spaces.Get(pg.Space)
		if !ok || !access.Can(role, access.Read) {
			continue
		}
		out = add(out, pg.Meta.Title, "/p/"+pg.Meta.ID, s.pageAudience(v, pageCtx{Page: pg, Space: sp, Role: role}))
	}
	return out
}

// listAudience names a to-do list's owner and everyone it is shared with.
func (s *Server) listAudience(v view, l todos.List) audience {
	members := []access.Member{{User: l.Owner, Role: access.Owner}}
	for _, sh := range l.Shared {
		members = append(members, access.Member{User: sh.User, Group: sh.Group, Role: access.Viewer})
	}
	return s.membersAudience(v, members)
}
