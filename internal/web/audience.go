// SPDX-License-Identifier: Apache-2.0

package web

import (
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/spaces"
)

// audience is the line that says who can see something.
type audience struct {
	Text    string
	Private bool
	// Change links to where the viewer may change who sees it; "" when they may not.
	Change, ChangeText string
}

// changeAt offers the viewer to change who sees it at href.
func (a audience) changeAt(v view, href string) audience {
	a.Change, a.ChangeText = href, v.T("visibility.change")
	return a
}

// ptr returns a pointer to a copy of v.
func ptr[T any](v T) *T { return &v }

// onlyYou is the audience of everything only its author sees.
func onlyYou(v view) audience { return audience{Text: v.T("visibility.only_you"), Private: true} }

// spaceAudience names who can read the space and its pages: the viewer first,
// then the others by name, members of a group as people. Admins read every
// space; Kanso learns who they are only at their sign-in, so they stay a role.
func (s *Server) spaceAudience(v view, sp spaces.Space) audience {
	return s.membersAudience(v, sp.Members)
}

// membersAudience names everyone members give access, as spaceAudience does.
func (s *Server) membersAudience(v view, members []access.Member) audience {
	g := s.groups()
	seen := map[string]bool{}
	for _, m := range members {
		if m.User != "" {
			seen[m.User] = true
		}
		for _, sub := range g[m.Group] {
			seen[sub] = true
		}
	}
	var you bool
	var names []string
	for sub := range seen {
		if sub == v.User.Sub {
			you = true
		} else {
			names = append(names, s.userName(sub))
		}
	}
	slices.Sort(names)
	if you {
		names = append([]string{v.T("visibility.you")}, names...)
	}
	if len(names) == 0 {
		return audience{Text: v.T("visibility.admins_only")}
	}
	return audience{Text: v.T("visibility.space", strings.Join(names, ", "))}
}

// spaceAudienceFor is spaceAudience with a link to the settings for whoever may change them.
func (s *Server) spaceAudienceFor(v view, sp spaces.Space, role access.Role) audience {
	a := s.spaceAudience(v, sp)
	if access.Can(role, access.ManageSpace) {
		a = a.changeAt(v, "/s/"+sp.Slug+"/settings")
	}
	return a
}

// pageAudience names who reads a page: the space's members plus everyone the
// page or a page above it is shared with; managers get a link to the page's sharing.
func (s *Server) pageAudience(v view, pc pageCtx) audience {
	members := slices.Clone(pc.Space.Members)
	for _, m := range s.Wiki.Chain(pc.Page.Meta.ID) {
		members = append(members, m.Members...)
	}
	a := s.membersAudience(v, members)
	if access.Can(pc.Role, access.ManageSpace) {
		a = a.changeAt(v, "/p/"+pc.Page.Meta.ID+"/share")
	}
	return a
}
