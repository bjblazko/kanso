// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strconv"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
)

// maxGrowthMonths is how far back the growth chart reaches.
const maxGrowthMonths = 36

// overviewTab: headline numbers, what there is, and how it grew.
func (s *Server) overviewTab(rb *reportBuild) {
	v := rb.rc.v
	t := reports.Totals(rb.docs, time.Now())
	rb.page.Tiles = []reportTile{
		{v.T("reports.tile_docs"), strconv.Itoa(t.Docs), v.T("reports.tile_docs_30", t.Docs30)},
		{v.T("reports.tile_bytes"), formatBytes(v, t.Bytes), v.T("reports.tile_bytes_att", formatBytes(v, t.AttachmentBytes))},
	}
	if rb.rc.scope == scopeAll && !rb.page.HistoryError {
		sc := rb.rc.scan
		rb.page.Tiles = append(rb.page.Tiles, reportTile{v.T("reports.tile_deleted"), formatBytes(v, sc.Deleted),
			v.T("reports.tile_deleted_n", sc.DeletedN, formatBytes(v, sc.PackBytes))})
	}
	s.inventoryCard(rb)
	growthCard(rb)
}

// kindsIn are the kinds that occur in docs, in report order.
func kindsIn(docs []reports.Doc) []reports.Kind {
	seen := map[reports.Kind]bool{}
	for _, d := range docs {
		seen[d.Kind] = true
	}
	var out []reports.Kind
	for _, k := range reports.Kinds {
		if seen[k] {
			out = append(out, k)
		}
	}
	return out
}

// chartSeries is data for a stacked chart: the series names, then one row per label.
type chartSeries struct {
	Keys []string   `json:"keys"`
	Rows []chartRow `json:"rows"`
}

type chartRow struct {
	Label  string `json:"label"`
	Values []int  `json:"values"`
}

func (s *Server) inventoryCard(rb *reportBuild) {
	v := rb.rc.v
	byPerson := rb.rc.scope == scopeAll
	kinds := kindsIn(rb.docs)
	card := reportCard{ID: "inventory", Chart: "bars", Title: v.T("reports.inventory"), Why: v.T("reports.inventory_why"),
		Empty: len(rb.docs) == 0, Head: []string{v.T("reports.kind"), v.T("reports.count")}}
	bars := reports.Inventory(rb.docs, byPerson, func(p string) string { return s.personName(v, p) }, v.T("reports.others"))
	data := chartSeries{}
	for _, k := range kinds {
		data.Keys = append(data.Keys, kindLabel(v, k))
	}
	for _, b := range bars {
		row := chartRow{Label: b.Label}
		for _, k := range kinds {
			row.Values = append(row.Values, b.Values[k])
		}
		data.Rows = append(data.Rows, row)
	}
	if byPerson {
		card.Chart, card.Why = "stacked-bars", v.T("reports.inventory_why_people")
		card.Head = append([]string{v.T("reports.person")}, data.Keys...)
	}
	card.Rows = seriesTable(data, byPerson)
	rb.card(card, data)
}

// seriesTable writes chart data as rows: one per row of data, or (one row of
// data, transposed) one per key.
func seriesTable(data chartSeries, perRow bool) [][]reportCell {
	var rows [][]reportCell
	if !perRow && len(data.Rows) == 1 {
		for i, k := range data.Keys {
			rows = append(rows, []reportCell{text(k), num(data.Rows[0].Values[i])})
		}
		return rows
	}
	for _, r := range data.Rows {
		cells := []reportCell{text(r.Label)}
		for _, n := range r.Values {
			cells = append(cells, num(n))
		}
		rows = append(rows, cells)
	}
	return rows
}

func growthCard(rb *reportBuild) {
	v := rb.rc.v
	now := time.Now()
	from := now
	for _, d := range rb.docs {
		if !d.Created.IsZero() && d.Created.Before(from) {
			from = d.Created
		}
	}
	if limit := now.AddDate(0, -maxGrowthMonths, 0); from.Before(limit) {
		from = limit
	}
	kinds := kindsIn(rb.docs)
	card := reportCard{ID: "growth", Chart: "stacked-area", Title: v.T("reports.growth"), Why: v.T("reports.growth_why"),
		Empty: len(rb.docs) == 0}
	data := chartSeries{}
	card.Head = []string{v.T("reports.month")}
	for _, k := range kinds {
		data.Keys = append(data.Keys, kindLabel(v, k))
		card.Head = append(card.Head, kindLabel(v, k))
	}
	for _, p := range reports.Growth(rb.docs, from, now) {
		row := chartRow{Label: p.Month.Format("2006-01")}
		cells := []reportCell{text(p.Month.Format("01/2006"))}
		for _, k := range kinds {
			row.Values = append(row.Values, p.Values[k])
			cells = append(cells, num(p.Values[k]))
		}
		data.Rows = append(data.Rows, row)
		card.Rows = append(card.Rows, cells)
	}
	rb.card(card, data)
}
