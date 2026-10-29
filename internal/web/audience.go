// SPDX-License-Identifier: Apache-2.0

package web

import (
	"slices"
	"strings"

	"github.com/bjblazko/kanso/internal/spaces"
)

// audience is the line that says who can see something.
type audience struct {
	Text    string
	Private bool
}

// ptr returns a pointer to a copy of v.
func ptr[T any](v T) *T { return &v }

// onlyYou is the audience of everything only its author sees.
func onlyYou(v view) audience { return audience{Text: v.T("visibility.only_you"), Private: true} }

// spaceAudience names who can read the space and its pages: the viewer first,
// then the others by name, members of a group as people. Admins read every
// space; Kanso learns who they are only at their sign-in, so they stay a role.
func (s *Server) spaceAudience(v view, sp spaces.Space) audience {
	g := s.groups()
	seen := map[string]bool{}
	for _, m := range sp.Members {
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
