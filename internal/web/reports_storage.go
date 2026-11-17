// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io/fs"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/reports"
)

// largestFiles is how many files "Größte Dateien" lists.
const largestFiles = 20

// storageTab: where the space sits, how the repository grew, what lies around.
func (s *Server) storageTab(rb *reportBuild) {
	s.areasCard(rb)
	if rb.rc.scope == scopeAll && !rb.page.HistoryError {
		historyCard(rb)
	}
	s.orphansCard(rb)
	s.largestCard(rb)
}

// areaName names an area for people.
func (s *Server) areaName(v view, area string) string {
	switch area {
	case "notes", "lists", "boards", "mealplans", "other":
		return v.T("reports.area_" + area)
	}
	if sp, ok := s.Spaces.Get(area); ok {
		return sp.Name
	}
	return area
}

type areaData struct {
	Name        string `json:"name"`
	Text        int64  `json:"text"`
	Attachments int64  `json:"attachments"`
	Deleted     int64  `json:"deleted"`
}

func (s *Server) areasCard(rb *reportBuild) {
	v := rb.rc.v
	var sc = &rb.rc.scan
	if rb.rc.scope != scopeAll || rb.page.HistoryError {
		sc = nil // what Git keeps is the installation's, not a person's
	}
	areas := reports.Storage(rb.docs, sc)
	card := reportCard{ID: "areas", Chart: "treemap", Title: v.T("reports.areas"), Why: v.T("reports.areas_why"),
		Empty: len(areas) == 0, Head: []string{v.T("reports.area"), v.T("reports.text"), v.T("reports.kind_attachment"), v.T("reports.deleted"), v.T("reports.sum")}}
	data := struct {
		Labels map[string]string `json:"labels"`
		Areas  []areaData        `json:"areas"`
	}{Labels: map[string]string{"text": v.T("reports.text"), "attachments": v.T("reports.kind_attachment"), "deleted": v.T("reports.deleted")}}
	for _, a := range areas {
		name := s.areaName(v, a.Name)
		data.Areas = append(data.Areas, areaData{name, a.Text, a.Attachments, a.Deleted})
		card.Rows = append(card.Rows, []reportCell{text(name), bytesCell(v, a.Text), bytesCell(v, a.Attachments),
			bytesCell(v, a.Deleted), bytesCell(v, a.Total())})
	}
	rb.card(card, data)
}

type sizeData struct {
	Month string `json:"t"`
	Repo  int64  `json:"repo"`
	Tree  int64  `json:"tree"`
}

func historyCard(rb *reportBuild) {
	v := rb.rc.v
	pts := reports.StorageOverTime(&rb.rc.scan, time.Now())
	card := reportCard{ID: "history", Chart: "size-lines", Title: v.T("reports.history"),
		Why: v.T("reports.history_why", formatBytes(v, rb.rc.scan.PackBytes)), Empty: len(pts) == 0,
		Head: []string{v.T("reports.month"), v.T("reports.repo"), v.T("reports.tree")}}
	data := struct {
		Labels map[string]string `json:"labels"`
		Points []sizeData        `json:"points"`
	}{Labels: map[string]string{"repo": v.T("reports.repo"), "tree": v.T("reports.tree")}}
	for _, p := range pts {
		data.Points = append(data.Points, sizeData{p.Month.Format("2006-01"), p.Repo, p.Tree})
		card.Rows = append(card.Rows, []reportCell{text(p.Month.Format("01/2006")), bytesCell(v, p.Repo), bytesCell(v, p.Tree)})
	}
	rb.card(card, data)
}

func (s *Server) orphansCard(rb *reportBuild) {
	v := rb.rc.v
	var refs []reports.AttachmentRef
	for _, d := range rb.docs {
		if d.Kind != reports.Attachment {
			continue
		}
		parent, name, _ := strings.Cut(d.ID, "/")
		refs = append(refs, reports.AttachmentRef{Doc: d, Name: name, Body: rb.rc.bodies[parent]})
	}
	files := reports.Orphans(refs)
	if rb.rc.scope == scopeAll {
		files = append(files, looseAttachments(rb)...)
	}
	card := reportCard{ID: "orphans", Title: v.T("reports.orphans"), Why: v.T("reports.orphans_why"), Empty: len(files) == 0,
		Head: []string{v.T("reports.file"), v.T("reports.area"), v.T("reports.size")}}
	for _, f := range files {
		card.Rows = append(card.Rows, []reportCell{docCell(v, f.Doc.Title, f.Doc.Href), text(s.areaName(v, f.Doc.Space)), bytesCell(v, f.Bytes)})
	}
	rb.card(card, nil)
}

// looseAttachments are attachment files whose page is gone. Nobody can read
// them, so they carry no name.
func looseAttachments(rb *reportBuild) []reports.File {
	var out []reports.File
	for p, n := range rb.rc.scan.Live {
		dir := path.Dir(p)
		if !strings.HasPrefix(p, "spaces/") || path.Base(dir) != "attachments" {
			continue
		}
		if _, ok := rb.rc.scan.Live[path.Join(path.Dir(dir), "index.md")]; !ok {
			out = append(out, reports.File{Doc: reports.Doc{Kind: reports.Attachment, Space: reports.AreaOf(p), Bytes: n}, Bytes: n})
		}
	}
	return out
}

func (s *Server) largestCard(rb *reportBuild) {
	v := rb.rc.v
	docs := reports.Largest(rb.docs, largestFiles)
	card := reportCard{ID: "largest", Title: v.T("reports.largest"), Why: v.T("reports.largest_why"), Empty: len(docs) == 0,
		Head: []string{v.T("reports.file"), v.T("reports.kind"), v.T("reports.area"), v.T("reports.size")}}
	for _, d := range docs {
		card.Rows = append(card.Rows, []reportCell{docCell(v, d.Title, d.Href), text(v.T("reports.one_" + string(d.Kind))),
			text(s.areaName(v, d.Space)), bytesCell(v, d.Bytes)})
	}
	rb.card(card, nil)
}

// serverTile is what the Kanso process and its data take; admins only.
func (s *Server) serverTile(v view, packBytes int64) reportTile {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	var data int64
	_ = filepath.WalkDir(s.Cfg.DataDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.Name() == ".git" && d.IsDir() {
			return fs.SkipDir
		}
		if info, err := d.Info(); err == nil && !d.IsDir() {
			data += info.Size()
		}
		return nil
	})
	up := time.Since(s.started).Round(time.Minute)
	return reportTile{v.T("reports.tile_server"), formatBytes(v, int64(mem.Sys)),
		v.T("reports.tile_server_detail", up.String(), formatBytes(v, data), formatBytes(v, packBytes))}
}
