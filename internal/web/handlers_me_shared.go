// SPDX-License-Identifier: Apache-2.0

package web

import (
	"cmp"
	"net/http"
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/todos"
)

// sharedItem is one thing someone else sees too, with who that is.
type sharedItem struct {
	Name, Href string
	Kind       string // its kind, singular, translated
	Audience   audience
	People     []string // subs of everyone besides the viewer who sees it
	kind       string   // i18n key of Kind
}

// sharedGroup is one heading of the overview, translated; "" is a list without one.
type sharedGroup struct {
	Heading string
	Items   []sharedItem
}

// sharedOverview is "Was andere sehen"; notes are always private and never appear.
type sharedOverview struct {
	By     string // "" (by kind), "doc" or "person"
	Groups []sharedGroup
}

// sharedViews are the groupings the overview offers, the default first.
var sharedViews = []struct{ By, Label string }{
	{"", "me.shared_by_kind"}, {"doc", "me.shared_by_doc"}, {"person", "me.shared_by_person"},
}

// kindHeadings are the i18n keys of the kind headings, by the key of the kind.
var kindHeadings = map[string]string{
	"me.kind_space": "me.shared_spaces", "me.kind_page": "me.shared_pages", "me.kind_list": "me.shared_lists",
	"me.kind_board": "me.shared_boards", "me.kind_plan": "me.shared_plans",
}

// add keeps an item only when someone besides the viewer sees it.
func (s *Server) add(v view, items []sharedItem, kind, name, href string, a audience, members []access.Member) []sharedItem {
	if a.Private {
		return items
	}
	var people []string
	for sub := range s.readers(members) {
		if sub != v.User.Sub {
			people = append(people, sub)
		}
	}
	return append(items, sharedItem{Name: name, Href: href, Kind: v.T(kind), Audience: a, People: people, kind: kind})
}

func (s *Server) sharedPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("me.shared")
	items := s.sharedItems(v, r)
	o := sharedOverview{By: r.URL.Query().Get("by")}
	switch o.By {
	case "doc":
		o.Groups = byDocument(items)
	case "person":
		o.Groups = s.byPerson(items)
	default:
		o.By, o.Groups = "", byKind(v, items)
	}
	v.Data = o
	s.render(w, r, http.StatusOK, "me_shared", v)
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
		for _, b := range boards {
			m := []access.Member{{Group: b.Owner.Group, Role: access.Viewer}}
			out = s.add(v, out, "me.kind_board", b.Name, "/tasks/b/"+b.ID, s.boardAudience(v, b), m)
		}
	}
	for _, pl := range s.MealPlans.For(g, p.Sub) {
		m := []access.Member{{Group: pl.Group, Role: access.Viewer}}
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

// listMembers are a to-do list's owner and everyone it is shared with.
func listMembers(l todos.List) []access.Member {
	members := []access.Member{{User: l.Owner, Role: access.Owner}}
	for _, sh := range l.Shared {
		members = append(members, access.Member{User: sh.User, Group: sh.Group, Role: access.Viewer})
	}
	return members
}

// listAudience names a to-do list's owner and everyone it is shared with.
func (s *Server) listAudience(v view, l todos.List) audience {
	return s.membersAudience(v, listMembers(l))
}

// byKind groups items under their kind, in the order sharedItems found them.
func byKind(v view, items []sharedItem) []sharedGroup {
	var out []sharedGroup
	for _, it := range items {
		heading := v.T(kindHeadings[it.kind])
		if n := len(out); n > 0 && out[n-1].Heading == heading {
			out[n-1].Items = append(out[n-1].Items, it)
			continue
		}
		out = append(out, sharedGroup{Heading: heading, Items: []sharedItem{it}})
	}
	return out
}

// byDocument is one list of all items, by name.
func byDocument(items []sharedItem) []sharedGroup {
	if len(items) == 0 {
		return nil
	}
	slices.SortStableFunc(items, func(a, b sharedItem) int { return byName(a.Name, b.Name) })
	return []sharedGroup{{Items: items}}
}

// byPerson lists, for each other person, what they see, people by name.
func (s *Server) byPerson(items []sharedItem) []sharedGroup {
	seen := map[string][]sharedItem{}
	for _, it := range items {
		for _, sub := range it.People {
			seen[sub] = append(seen[sub], it)
		}
	}
	var out []sharedGroup
	for sub, its := range seen {
		out = append(out, sharedGroup{Heading: s.userName(sub), Items: its})
	}
	slices.SortFunc(out, func(a, b sharedGroup) int { return byName(a.Heading, b.Heading) })
	return out
}

// byName orders names alphabetically, ignoring case.
func byName(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) }
