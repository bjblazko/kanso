// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/store"
)

// reportTab is one tab of the Berichte page and what builds it.
type reportTab struct {
	ID    string
	build func(s *Server, rb *reportBuild)
}

// reportTabs are the tabs in order.
var reportTabs = []reportTab{
	{"overview", (*Server).overviewTab},
	{"storage", (*Server).storageTab},
	{"activity", (*Server).activityTab},
	{"wiki", (*Server).wikiTab},
	{"tasks", (*Server).tasksTab},
	{"kitchen", (*Server).kitchenTab},
}

// reportPage is what the reports template shows.
type reportPage struct {
	Tab          string
	Scope        reportScope
	Tabs, Scopes []sharedTab
	Tiles        []reportTile
	Cards        []reportCard
	Stand        time.Time
	ForAdmin     bool // the viewer is an admin (no "Admins sehen" note)
	HistoryError bool
	JSON         template.JS // chart data by card id
}

// reportTile is one headline number.
type reportTile struct{ Label, Value, Detail string }

// reportCard is one chart with its numbers as a table.
type reportCard struct {
	ID, Chart, Title, Why string
	Head                  []string
	Rows                  [][]reportCell
	Empty                 bool
}

// reportCell is one table cell; Num aligns right.
type reportCell struct {
	Text, Href string
	Num        bool
}

// reportBuild is one request's report in the making.
type reportBuild struct {
	rc   reportCtx
	docs []reports.Doc
	page *reportPage
	data map[string]any
}

// card adds a chart card with its data.
func (rb *reportBuild) card(c reportCard, data any) {
	rb.page.Cards = append(rb.page.Cards, c)
	if !c.Empty {
		rb.data[c.ID] = data
	}
}

func (s *Server) reportsIndex(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/reports/overview", http.StatusSeeOther)
}

func (s *Server) reportsPage(w http.ResponseWriter, r *http.Request) {
	i := slices.IndexFunc(reportTabs, func(t reportTab) bool { return t.ID == r.PathValue("tab") })
	if i < 0 {
		s.notFound(w, r)
		return
	}
	_, p := auth.UserFrom(r.Context())
	scope, ok := pickScope(r.URL.Query().Get("scope"), p.Admin)
	if !ok {
		s.forbidden(w, r)
		return
	}
	v := s.newView(r, "", nil)
	v.Title = v.T("reports.title")
	rb := s.newReport(v, reportCtx{v: v, p: p, g: s.groups(), scope: scope})
	rb.page.Tab = reportTabs[i].ID
	reportTabs[i].build(s, rb)
	rb.page.Tabs, rb.page.Scopes = reportLinks(v, rb.page.Tab, scope, p.Admin)
	js, _ := json.Marshal(rb.data) // json escapes <, > and &: safe inside <script>
	rb.page.JSON = template.JS(js)
	v.Data = *rb.page
	s.render(w, r, http.StatusOK, "reports", v)
}

// newReport reads the history (cached per HEAD) and collects the scope's documents.
func (s *Server) newReport(v view, rc reportCtx) *reportBuild {
	page := &reportPage{Scope: rc.scope, ForAdmin: rc.p.Admin}
	head, err := s.Repo.Head()
	if err == nil {
		rc.scan, page.Stand, err = s.reports.get(head, s.scanRepo)
	}
	if err != nil {
		slog.Warn("reports: reading the history", "err", err)
		page.HistoryError, rc.scan = true, store.Scan{Live: map[string]int64{}}
	}
	rc.added, rc.bodies = s.addedBy(rc.scan), map[string]string{}
	return &reportBuild{rc: rc, docs: s.reportDocs(rc), page: page, data: map[string]any{}}
}

// pickScope reads the scope; false when a non-admin asks for everything.
func pickScope(q string, admin bool) (reportScope, bool) {
	sc := reportScope(q)
	if sc == scopeAll && !admin {
		return "", false
	}
	if slices.Contains(reportScopes, sc) {
		return sc, true
	}
	if admin {
		return scopeAll, true
	}
	return scopeOwn, true
}

// reportLinks are the tab links (keeping the scope) and the scope links (keeping the tab).
func reportLinks(v view, tab string, scope reportScope, admin bool) (tabs, scopes []sharedTab) {
	for _, t := range reportTabs {
		tabs = append(tabs, sharedTab{v.T("reports.tab_" + t.ID), "/reports/" + t.ID + "?scope=" + string(scope), t.ID == tab})
	}
	for _, sc := range reportScopes {
		if sc == scopeAll && !admin {
			continue
		}
		scopes = append(scopes, sharedTab{v.T("reports.scope_" + string(sc)), "/reports/" + tab + "?scope=" + string(sc), sc == scope})
	}
	return tabs, scopes
}

// personName names a document's person for a chart.
func (s *Server) personName(v view, person string) string {
	switch {
	case person == "":
		return v.T("reports.unknown")
	case strings.HasPrefix(person, "group:"):
		return v.T("reports.group", strings.TrimPrefix(person, "group:"))
	}
	return s.userName(person)
}

// kindLabel is the plural of a kind, for charts.
func kindLabel(v view, k reports.Kind) string { return v.T("reports.kind_" + string(k)) }

// formatBytes writes n bytes the way people read them: "3,8 GB".
func formatBytes(v view, n int64) string {
	units := []string{"B", "KB", "MB", "GB", "TB"}
	f, u := float64(n), 0
	for f >= 1000 && u < len(units)-1 {
		f, u = f/1000, u+1
	}
	s := fmt.Sprintf("%.0f", f)
	if u > 0 && f < 100 {
		s = fmt.Sprintf("%.1f", f)
	}
	if v.Lang == "de" {
		s = strings.Replace(s, ".", ",", 1)
	}
	return s + " " + units[u]
}

func num(n int) reportCell     { return reportCell{Text: fmt.Sprint(n), Num: true} }
func text(s string) reportCell { return reportCell{Text: s} }

func bytesCell(v view, n int64) reportCell { return reportCell{Text: formatBytes(v, n), Num: true} }

// docCell is a document's name, linked when the viewer may read it.
func docCell(v view, title, href string) reportCell {
	if title == "" {
		return text(v.T("reports.no_access"))
	}
	return reportCell{Text: title, Href: href}
}
