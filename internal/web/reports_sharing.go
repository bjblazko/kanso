// SPDX-License-Identifier: Apache-2.0

package web

import (
	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/tasks"
)

// sharedThing is something more than one person sees.
type sharedThing struct {
	kind   int // index into shareKinds
	people []string
	mine   bool // the viewer shares it (owns it), rather than it being shared with them
}

// shareKinds are the headings of the sharing counts, as on "Was andere sehen".
var shareKinds = []string{"me.shared_spaces", "me.shared_pages", "me.shared_lists", "me.shared_boards", "me.shared_plans"}

// sharingTab: who shares with whom, and how much the viewer shares.
func (s *Server) sharingTab(rb *reportBuild) {
	things := s.sharedThings(rb)
	s.networkCard(rb, things)
	sharesCard(rb, things)
}

// thingCollector gathers the shared things of one report.
type thingCollector struct {
	s   *Server
	rc  reportCtx
	all bool // everyone's scope: every thing, not only what the viewer sees
	out []sharedThing
}

// add keeps a thing when more than one person sees it.
func (c *thingCollector) add(kind int, members []access.Member, mine bool) {
	acc := c.s.readers(members)
	if len(acc) < 2 {
		return
	}
	t := sharedThing{kind: kind, mine: mine}
	for sub := range acc {
		t.people = append(t.people, sub)
	}
	c.out = append(c.out, t)
}

// sharedThings are the things the viewer sees (everything in everyone's scope)
// that more than one person sees.
func (s *Server) sharedThings(rb *reportBuild) []sharedThing {
	c := &thingCollector{s: s, rc: rb.rc, all: rb.rc.scope == scopeAll}
	c.spaces()
	c.pages()
	c.lists()
	c.groups()
	return c.out
}

func (c *thingCollector) spaces() {
	list := c.s.Spaces.Visible(c.rc.p, c.rc.g)
	if c.all {
		list = c.s.Spaces.List()
	}
	for _, sp := range list {
		c.add(0, sp.Members, c.s.readers(sp.Members)[c.rc.p.Sub] == access.Owner)
	}
}

func (c *thingCollector) pages() {
	for _, pg := range c.s.Wiki.Shared() {
		role := c.s.pageRole(c.rc.p, pg.Space, pg.Meta.ID, c.rc.g)
		sp, ok := c.s.Spaces.Get(pg.Space)
		if ok && (c.all || access.Can(role, access.Read)) {
			c.add(1, c.s.pageMembers(pageCtx{Page: pg, Space: sp, Role: role}), role == access.Owner)
		}
	}
}

func (c *thingCollector) lists() {
	lists := c.s.Todos.Visible(c.rc.p.Sub, c.rc.g)
	if c.all {
		lists = c.s.Todos.All()
	}
	for _, l := range lists {
		c.add(2, listMembers(l), l.Owner == c.rc.p.Sub)
	}
}

// groups adds group boards and meal plans; they belong to their group, never to the viewer.
func (c *thingCollector) groups() {
	viewer := tasks.Actor{Sub: c.rc.p.Sub, Groups: c.rc.g}
	for _, b := range c.s.Tasks.GroupBoards() {
		if c.all || viewer.Sees(b) {
			c.add(3, []access.Member{{Group: b.Owner.Group, Role: access.Editor}}, false)
		}
	}
	plans := c.s.MealPlans.For(c.rc.g, c.rc.p.Sub)
	if c.all {
		plans = c.s.MealPlans.All()
	}
	for _, pl := range plans {
		c.add(4, []access.Member{{Group: pl.Group, Role: access.Editor}}, false)
	}
}

type nodeData struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type linkData struct {
	A string `json:"a"`
	B string `json:"b"`
	N int    `json:"n"`
}

func (s *Server) networkCard(rb *reportBuild, things []sharedThing) {
	v := rb.rc.v
	groups := make([][]string, len(things))
	for i, t := range things {
		groups[i] = t.people
	}
	people, links := reports.Network(groups)
	card := reportCard{ID: "network", Chart: "network", Title: v.T("reports.network"), Why: v.T("reports.network_why"),
		Empty: len(links) == 0, Head: []string{v.T("reports.pair"), v.T("reports.things")}}
	data := struct {
		Nodes []nodeData `json:"nodes"`
		Links []linkData `json:"links"`
		Unit  string     `json:"unit"`
	}{Unit: v.T("reports.things")}
	for _, p := range people {
		data.Nodes = append(data.Nodes, nodeData{p, s.userName(p)})
	}
	for _, l := range links {
		data.Links = append(data.Links, linkData{l.A, l.B, l.N})
		card.Rows = append(card.Rows, []reportCell{text(s.userName(l.A) + " ↔ " + s.userName(l.B)), num(l.N)})
	}
	rb.card(card, data)
}

func sharesCard(rb *reportBuild, things []sharedThing) {
	v := rb.rc.v
	var mine, theirs [5]int
	for _, t := range things {
		if t.mine {
			mine[t.kind]++
		} else {
			theirs[t.kind]++
		}
	}
	card := reportCard{ID: "shares", Title: v.T("reports.shares"), Why: v.T("reports.shares_why"), Empty: len(things) == 0,
		Head: []string{v.T("reports.kind"), v.T("reports.i_share"), v.T("reports.shared_with_me")}}
	for i, key := range shareKinds {
		card.Rows = append(card.Rows, []reportCell{text(v.T(key)), num(mine[i]), num(theirs[i])})
	}
	rb.card(card, nil)
}
