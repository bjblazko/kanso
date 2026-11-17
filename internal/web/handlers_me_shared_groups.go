// SPDX-License-Identifier: Apache-2.0

package web

import (
	"cmp"
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
)

// row is it as a line; withKind names its kind beside the name.
func (s *Server) row(v view, it sharedItem, withKind bool, who string) sharedRow {
	r := sharedRow{Name: it.Name, Href: it.Href, Who: who, Change: it.Change}
	if withKind {
		r.Kind = v.T(it.kind)
	}
	return r
}

// byKind groups items under their kind, in the order sharedItems found them.
func (s *Server) byKind(v view, items []sharedItem) []sharedGroup {
	var out []sharedGroup
	for _, it := range items {
		heading := v.T(kindHeadings[it.kind])
		r := s.row(v, it, false, s.whoByRole(v, it.Access))
		if n := len(out); n > 0 && out[n-1].Heading == heading {
			out[n-1].Rows = append(out[n-1].Rows, r)
			continue
		}
		out = append(out, sharedGroup{Heading: heading, Rows: []sharedRow{r}})
	}
	return out
}

// byDocument is one list of all items, by name.
func (s *Server) byDocument(v view, items []sharedItem) []sharedGroup {
	if len(items) == 0 {
		return nil
	}
	sorted := slices.Clone(items)
	slices.SortStableFunc(sorted, func(a, b sharedItem) int { return byName(a.Name, b.Name) })
	g := sharedGroup{}
	for _, it := range sorted {
		g.Rows = append(g.Rows, s.row(v, it, true, s.whoByRole(v, it.Access)))
	}
	return []sharedGroup{g}
}

// byPerson lists, for each other person, what they see and with which right, people by name.
func (s *Server) byPerson(v view, items []sharedItem) []sharedGroup {
	rows := map[string][]sharedRow{}
	for _, it := range items {
		for sub, r := range it.Access {
			row := s.row(v, it, true, "") // the right sits beside the kind
			row.Kind += " · " + roleLabel(v, r)
			rows[sub] = append(rows[sub], row)
		}
	}
	var out []sharedGroup
	for sub, rs := range rows {
		out = append(out, sharedGroup{Heading: s.userName(sub), Rows: rs})
	}
	slices.SortFunc(out, func(a, b sharedGroup) int { return byName(a.Heading, b.Heading) })
	return out
}

// byRole lists, from the highest right down, what others may do and who they are.
func (s *Server) byRole(v view, items []sharedItem) []sharedGroup {
	var out []sharedGroup
	for _, role := range slices.Backward(sharedRoles) {
		g := sharedGroup{Heading: roleLabel(v, role)}
		for _, it := range items {
			if names := s.namesWith(it.Access, role); len(names) > 0 {
				g.Rows = append(g.Rows, s.row(v, it, true, strings.Join(names, ", ")))
			}
		}
		if len(g.Rows) > 0 {
			out = append(out, g)
		}
	}
	return out
}

// whoByRole says who may do what: "Verwalten: Ben · Schreiben: Lena, Oma".
func (s *Server) whoByRole(v view, acc map[string]access.Role) string {
	var parts []string
	for _, role := range slices.Backward(sharedRoles) {
		if names := s.namesWith(acc, role); len(names) > 0 {
			parts = append(parts, roleLabel(v, role)+": "+strings.Join(names, ", "))
		}
	}
	return strings.Join(parts, " · ")
}

// namesWith are the names of everyone with exactly role, by name.
func (s *Server) namesWith(acc map[string]access.Role, role access.Role) []string {
	var names []string
	for sub, r := range acc {
		if r == role {
			names = append(names, s.userName(sub))
		}
	}
	slices.SortFunc(names, byName)
	return names
}

func roleLabel(v view, r access.Role) string { return v.T("role." + r.String()) }

// byName orders names alphabetically, ignoring case.
func byName(a, b string) int { return cmp.Compare(strings.ToLower(a), strings.ToLower(b)) }

// ViewTabs are the groupings above the overview; each keeps the filter.
func (o sharedOverview) ViewTabs(v view) []sharedTab {
	var tabs []sharedTab
	for _, sv := range sharedViews {
		tabs = append(tabs, sharedTab{v.T(sv.Label), sharedHref(sv.By, o.Role), sv.By == o.By})
	}
	return tabs
}

// RoleTabs are the filters by right; each keeps the grouping.
func (o sharedOverview) RoleTabs(v view) []sharedTab {
	tabs := []sharedTab{{v.T("me.shared_all_roles"), sharedHref(o.By, ""), o.Role == ""}}
	for _, r := range sharedRoles {
		tabs = append(tabs, sharedTab{roleLabel(v, r), sharedHref(o.By, r.String()), o.Role == r.String()})
	}
	return tabs
}

// sharedHref is the overview's address with grouping by and filter role.
func sharedHref(by, role string) string {
	var q []string
	if by != "" {
		q = append(q, "by="+by)
	}
	if role != "" {
		q = append(q, "role="+role)
	}
	if len(q) == 0 {
		return "/me/shared"
	}
	return "/me/shared?" + strings.Join(q, "&")
}
