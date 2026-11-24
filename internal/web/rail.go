// SPDX-License-Identifier: Apache-2.0

package web

// railItem is one part of Kanso in the rail (ADR-005). Key names it in the
// person's menu settings, which the browser keeps per device (ADR-020).
type railItem struct {
	Key, Href, Icon, Label string
	Current                bool
}

// railParts are the parts in their default order; new parts go last, so a
// saved order elsewhere picks them up at its end.
var railParts = []struct{ key, href, icon, label string }{
	{"start", "/", "home", "nav.start"},
	{"notes", "/notes", "note", "nav.notes"},
	{"todos", "/todos", "check", "nav.todos"},
	{"tasks", "/tasks", "board", "nav.tasks"},
	{"wiki", "/wiki", "book", "nav.wiki"},
	{"kitchen", "/recipes", "pot", "nav.kitchen"},
	{"focus", "/focus", "timer", "nav.focus"},
}

// Rail lists the parts for the page, the current one marked.
func (v view) Rail() []railItem {
	out := make([]railItem, 0, len(railParts))
	for _, p := range railParts {
		out = append(out, railItem{Key: p.key, Href: p.href, Icon: p.icon, Label: v.T(p.label), Current: v.Module == p.key})
	}
	return out
}
