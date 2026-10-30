// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/wiki"
)

// pageRole is what p may do on page id in space: the highest of the space
// role and the members of the page and every page above it (ADR-009).
func (s *Server) pageRole(p access.Principal, space, id string, g access.Groups) access.Role {
	role := s.Spaces.RoleOf(p, space, g)
	for _, m := range s.Wiki.Chain(id) {
		role = max(role, access.Resolve(p, m.Members, g))
	}
	return role
}

// targetRole is the role for putting a page below parent ("" = top level) in space.
func (s *Server) targetRole(p access.Principal, space, parent string, g access.Groups) access.Role {
	if parent == "" {
		return s.Spaces.RoleOf(p, space, g)
	}
	if pg, ok := s.Wiki.ByID(parent); !ok || pg.Space != space {
		return access.None
	}
	return s.pageRole(p, space, parent, g)
}

// visibleTree is the space's page tree as p may see it: all of it with a
// space role, otherwise only the subtrees shared with p.
func (s *Server) visibleTree(p access.Principal, space string, g access.Groups) []*wiki.Node {
	tree := s.Wiki.Tree(space)
	if access.Can(s.Spaces.RoleOf(p, space, g), access.Read) {
		return tree
	}
	return sharedNodes(p, tree, g)
}

// sharedNodes keeps the nodes whose own members let p read them, with their
// subtrees; a node p cannot read gives way to its readable descendants.
func sharedNodes(p access.Principal, nodes []*wiki.Node, g access.Groups) []*wiki.Node {
	var out []*wiki.Node
	for _, n := range nodes {
		if access.Can(access.Resolve(p, n.Meta.Members, g), access.Read) {
			out = append(out, n)
			continue
		}
		out = append(out, sharedNodes(p, n.Children, g)...)
	}
	return out
}

// placeFor loads the space of the URL with the role for adding a page below
// parent: the space role at the top level, the page role below a page.
func (s *Server) placeFor(w http.ResponseWriter, r *http.Request, parent string) (spaces.Space, access.Role, bool) {
	_, p := auth.UserFrom(r.Context())
	sp, ok := s.Spaces.Get(r.PathValue("space"))
	role := s.targetRole(p, sp.Slug, parent, s.groups())
	if !ok || !access.Can(role, access.Read) {
		s.notFound(w, r)
		return spaces.Space{}, access.None, false
	}
	return sp, role, true
}

// writableTargets lists the pages of sp below which p may put a page, labelled
// with their path; the subtree of skip is left out (a page cannot move into itself).
func (s *Server) writableTargets(p access.Principal, sp spaces.Space, g access.Groups, skip string) []moveTarget {
	var out []moveTarget
	var walk func(nodes []*wiki.Node, prefix string, inherited access.Role)
	walk = func(nodes []*wiki.Node, prefix string, inherited access.Role) {
		for _, n := range nodes {
			if n.Meta.ID == skip {
				continue
			}
			role := max(inherited, access.Resolve(p, n.Meta.Members, g))
			if access.Can(role, access.Write) {
				out = append(out, moveTarget{ID: n.Meta.ID, Label: prefix + n.Meta.Title})
			}
			walk(n.Children, prefix+n.Meta.Title+" / ", role)
		}
	}
	walk(s.Wiki.Tree(sp.Slug), sp.Name+" / ", s.Spaces.RoleOf(p, sp.Slug, g))
	return out
}
