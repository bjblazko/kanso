// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/wiki"
)

// oldestPages is how many pages "Am längsten unverändert" lists.
const oldestPages = 10

// wikiTab: how old the wiki is and how it is shaped.
func (s *Server) wikiTab(rb *reportBuild) {
	ageCard(rb)
	oldestCard(rb)
	s.treeCard(rb)
}

func ageCard(rb *reportBuild) {
	v := rb.rc.v
	bins := reports.Age(rb.docs, time.Now())
	card := reportCard{ID: "age", Chart: "age", Title: v.T("reports.age"), Why: v.T("reports.age_why"), Empty: true,
		Head: []string{v.T("reports.last_change"), v.T("reports.kind_page")}}
	data := struct {
		Rows  []chartRow `json:"rows"`
		Stale int        `json:"stale"` // bins from here on are old: shown in grey
		Unit  string     `json:"unit"`
	}{Stale: 4, Unit: v.T("reports.kind_page")}
	for _, b := range bins {
		label := v.T("reports.age_" + b.Key)
		data.Rows = append(data.Rows, chartRow{Label: label, Values: []int{b.N}})
		card.Rows = append(card.Rows, []reportCell{text(label), num(b.N)})
		card.Empty = card.Empty && b.N == 0
	}
	rb.card(card, data)
}

func oldestCard(rb *reportBuild) {
	v := rb.rc.v
	pages := reports.Oldest(rb.docs, oldestPages)
	card := reportCard{ID: "oldest", Title: v.T("reports.oldest"), Why: v.T("reports.oldest_why"), Empty: len(pages) == 0,
		Head: []string{v.T("reports.one_page"), v.T("reports.last_change")}}
	for _, p := range pages {
		card.Rows = append(card.Rows, []reportCell{docCell(v, p.Title, p.Href), text(p.Updated.Local().Format("02.01.2006"))})
	}
	rb.card(card, nil)
}

// treeNode is the wiki as a tree for the sunburst: space → page → subpage.
type treeNode struct {
	Name     string      `json:"name"`
	Bytes    int64       `json:"bytes,omitempty"`
	Children []*treeNode `json:"children,omitempty"`
}

func (s *Server) treeCard(rb *reportBuild) {
	v := rb.rc.v
	inScope := map[string]reports.Doc{}
	bytes := map[string]int64{} // page id → text plus attachments
	for _, d := range rb.docs {
		switch d.Kind {
		case reports.Page, reports.Recipe:
			inScope[d.ID] = d
			bytes[d.ID] += d.Bytes
		case reports.Attachment:
			parent, _, _ := strings.Cut(d.ID, "/")
			bytes[parent] += d.Bytes
		}
	}
	root := &treeNode{Name: v.T("reports.wiki")}
	for _, sp := range s.Spaces.List() {
		node := &treeNode{Name: sp.Name}
		node.Children = treeOf(v, s.Wiki.Tree(sp.Slug), inScope, bytes)
		if len(node.Children) > 0 {
			root.Children = append(root.Children, node)
		}
	}
	card := reportCard{ID: "tree", Chart: "sunburst", Title: v.T("reports.tree_title"), Why: v.T("reports.tree_why"),
		Empty: len(root.Children) == 0, Head: []string{v.T("reports.area"), v.T("reports.kind_page"), v.T("reports.size")}}
	for _, sp := range root.Children {
		n, b := count(sp)
		card.Rows = append(card.Rows, []reportCell{text(sp.Name), num(n), bytesCell(v, b)})
	}
	rb.card(card, root)
}

// treeOf keeps the pages in scope; a page outside lifts its children up.
func treeOf(v view, nodes []*wiki.Node, inScope map[string]reports.Doc, bytes map[string]int64) []*treeNode {
	var out []*treeNode
	for _, n := range nodes {
		children := treeOf(v, n.Children, inScope, bytes)
		d, ok := inScope[n.Meta.ID]
		if !ok {
			out = append(out, children...)
			continue
		}
		name := d.Title
		if name == "" {
			name = v.T("reports.no_access")
		}
		out = append(out, &treeNode{Name: name, Bytes: bytes[n.Meta.ID], Children: children})
	}
	return out
}

// count is the number of pages under n and their bytes.
func count(n *treeNode) (int, int64) {
	pages, b := 0, int64(0)
	for _, c := range n.Children {
		cp, cb := count(c)
		pages, b = pages+1+cp, b+c.Bytes+cb
	}
	return pages, b
}
