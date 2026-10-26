// SPDX-License-Identifier: Apache-2.0

// Package access decides who may do what in a space.
package access

import (
	"fmt"
	"slices"
)

// Role is a person's standing in one space; higher includes lower.
type Role int

// Roles from least to most.
const (
	None Role = iota
	Viewer
	Editor
	Owner
)

var roleNames = map[Role]string{None: "none", Viewer: "viewer", Editor: "editor", Owner: "owner"}

func (r Role) String() string { return roleNames[r] }

// ParseRole reads viewer, editor or owner.
func ParseRole(s string) (Role, error) {
	for r, name := range roleNames {
		if name == s && r != None {
			return r, nil
		}
	}
	return None, fmt.Errorf("unknown role %q", s)
}

// MarshalText stores a role by name.
func (r Role) MarshalText() ([]byte, error) { return []byte(r.String()), nil }

// UnmarshalText reads a role by name.
func (r *Role) UnmarshalText(b []byte) error {
	parsed, err := ParseRole(string(b))
	*r = parsed
	return err
}

// Principal is the person making a request.
type Principal struct {
	Sub   string
	Admin bool
}

// Member grants Role to one user or one group (exactly one is set).
type Member struct {
	User  string `yaml:"user,omitempty"`
	Group string `yaml:"group,omitempty"`
	Role  Role   `yaml:"role"`
}

// Groups maps a group name to the subjects of its members.
type Groups map[string][]string

// Resolve returns p's highest role from members; admins are owners everywhere.
func Resolve(p Principal, members []Member, g Groups) Role {
	if p.Admin {
		return Owner
	}
	best := None
	for _, m := range members {
		if m.Role > best && m.applies(p.Sub, g) {
			best = m.Role
		}
	}
	return best
}

func (m Member) applies(sub string, g Groups) bool {
	if m.User != "" {
		return m.User == sub
	}
	return slices.Contains(g[m.Group], sub)
}

// Action is something a person wants to do in a space.
type Action int

// Actions, each with the minimum role it needs.
const (
	Read Action = iota
	Write
	ManageSpace
	BreakLock
)

var minimum = map[Action]Role{Read: Viewer, Write: Editor, ManageSpace: Owner, BreakLock: Owner}

// Can reports whether role r allows action a.
func Can(r Role, a Action) bool {
	need, ok := minimum[a]
	return ok && r >= need
}
