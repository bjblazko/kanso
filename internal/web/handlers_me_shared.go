// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/todos"
)

// sharedItem is one thing someone else sees too, with who that is and how far they may go.
type sharedItem struct {
	Name, Href string
	kind       string                 // i18n key of its kind, singular
	Access     map[string]access.Role // everyone besides the viewer, by sub
	Change     string                 // where the viewer may change who sees it, if they may
}

// sharedRow is one line of the overview, every text translated.
type sharedRow struct {
	Name, Href string
	Kind       string // shown beside the name; "" where the heading says it
	Who        string // the line below or beside: names, roles or both
	Change     string
}

// sharedGroup is one heading of the overview; "" is a list without one.
type sharedGroup struct {
	Heading string
	Rows    []sharedRow
}

// sharedOverview is "Was andere sehen"; notes are always private and never appear.
type sharedOverview struct {
	By, Role string // grouping ("" = by kind) and filter ("" = every role)
	Groups   []sharedGroup
	Nothing  bool // nothing at all is shared, before filtering
}

// sharedTab is one link above the overview.
type sharedTab struct {
	Label, Href string
	Current     bool
}

// sharedViews are the groupings, the default first; sharedRoles the filters.
var (
	sharedViews = []struct{ By, Label string }{
		{"", "me.shared_by_kind"}, {"doc", "me.shared_by_doc"},
		{"person", "me.shared_by_person"}, {"role", "me.shared_by_role"},
	}
	sharedRoles = []access.Role{access.Viewer, access.Editor, access.Owner}
)

// kindHeadings are the i18n keys of the kind headings, by the key of the kind.
var kindHeadings = map[string]string{
	"me.kind_space": "me.shared_spaces", "me.kind_page": "me.shared_pages", "me.kind_list": "me.shared_lists",
	"me.kind_board": "me.shared_boards", "me.kind_plan": "me.shared_plans",
}

// add keeps an item only when someone besides the viewer sees it.
func (s *Server) add(v view, items []sharedItem, kind, name, href string, a audience, members []access.Member) []sharedItem {
	acc := s.readers(members)
	delete(acc, v.User.Sub)
	if a.Private || len(acc) == 0 {
		return items
	}
	return append(items, sharedItem{Name: name, Href: href, kind: kind, Access: acc, Change: a.Change})
}

func (s *Server) sharedPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("me.shared")
	q := r.URL.Query()
	o := sharedOverview{By: q.Get("by"), Role: q.Get("role")}
	role, err := access.ParseRole(o.Role)
	if err != nil {
		o.Role = ""
	}
	all := s.sharedItems(v, r)
	o.Nothing = len(all) == 0
	items := onlyRole(all, o.Role, role)
	switch o.By {
	case "doc":
		o.Groups = s.byDocument(v, items)
	case "person":
		o.Groups = s.byPerson(v, items)
	case "role":
		o.Groups = s.byRole(v, items)
	default:
		o.By, o.Groups = "", s.byKind(v, items)
	}
	v.Data = o
	s.render(w, r, http.StatusOK, "me_shared", v)
}

// onlyRole keeps of each item the people with exactly role, and the items left with anyone.
func onlyRole(items []sharedItem, key string, role access.Role) []sharedItem {
	if key == "" {
		return items
	}
	var out []sharedItem
	for _, it := range items {
		acc := map[string]access.Role{}
		for sub, r := range it.Access {
			if r == role {
				acc[sub] = r
			}
		}
		if len(acc) > 0 {
			it.Access = acc
			out = append(out, it)
		}
	}
	return out
}

// sharedItems is everything the viewer sees that someone else sees too, by kind.
func (s *Server) sharedItems(v view, r *http.Request) []sharedItem {
	_, p := auth.UserFrom(r.Context())
	g := s.groups()
	var out []sharedItem
	for _, sp := range s.Spaces.Visible(p, g) {
		out = s.add(v, out, "me.kind_space", sp.Name, "/s/"+sp.Slug, s.spaceAudience(v, sp), sp.Members)
	}
	out = append(out, s.sharedPages(v, p, g)...)
	for _, l := range s.Todos.Visible(p.Sub, g) {
		m := listMembers(l)
		out = s.add(v, out, "me.kind_list", l.Name, "/todos/"+l.ID, s.membersAudience(v, m), m)
	}
	if boards, err := s.Tasks.Boards(s.actor(r)); err == nil {
		for _, b := range boards { // everyone in the group works on the board
			m := []access.Member{{Group: b.Owner.Group, Role: access.Editor}}
			out = s.add(v, out, "me.kind_board", b.Name, "/tasks/b/"+b.ID, s.boardAudience(v, b), m)
		}
	}
	for _, pl := range s.MealPlans.For(g, p.Sub) { // everyone in the group plans
		m := []access.Member{{Group: pl.Group, Role: access.Editor}}
		out = s.add(v, out, "me.kind_plan", pl.Name, "/mealplan/"+pl.ID, s.membersAudience(v, m), m)
	}
	return out
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
		pc := pageCtx{Page: pg, Space: sp, Role: role}
		out = s.add(v, out, "me.kind_page", pg.Meta.Title, "/p/"+pg.Meta.ID, s.pageAudience(v, pc), s.pageMembers(pc))
	}
	return out
}

// listMembers are a to-do list's owner and everyone it is shared with, who tick and add items.
func listMembers(l todos.List) []access.Member {
	members := []access.Member{{User: l.Owner, Role: access.Owner}}
	for _, sh := range l.Shared {
		members = append(members, access.Member{User: sh.User, Group: sh.Group, Role: access.Editor})
	}
	return members
}

// listAudience names a to-do list's owner and everyone it is shared with.
func (s *Server) listAudience(v view, l todos.List) audience {
	return s.membersAudience(v, listMembers(l))
}
