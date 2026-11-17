// SPDX-License-Identifier: Apache-2.0

package web

import (
	"cmp"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/reports"
	"github.com/bjblazko/kanso/internal/visits"
)

// homeMaxItems is how many changed documents the start page lists.
const homeMaxItems = 20

// homeRanges are the windows the start page offers; "" is since the last visit.
var homeRanges = []struct {
	Key string
	D   time.Duration
}{{"", 0}, {"24h", 24 * time.Hour}, {"7d", 7 * 24 * time.Hour}, {"30d", 30 * 24 * time.Hour}}

// homeSummary is what changed in the window, for the start page.
type homeSummary struct {
	Ranges       []homeOption
	Own          bool
	OwnHref      string
	Range        string
	Sentence     string
	Empty        bool
	Tiles        []homeTile
	Cards        []reportCard // the band and who was active
	Groups       []homeGroup
	More         int
	HistoryError bool
	JSON         template.JS // chart data by id
	Today        homeToday
}

// homeOption is one entry of the window select.
type homeOption struct {
	Value, Label string
	Selected     bool
}

// homeTile is one kind's number with its spark line.
type homeTile struct{ Kind, Label, Value, Spark string }

// homeGroup is the changed documents of one space (or of lists, notes, tasks).
type homeGroup struct {
	Name  string
	Items []homeItem
}

// homeItem is one line of the start page: a document or a meal or a task.
type homeItem struct{ Title, Href, Person, When string }

// startPage is Kanso's start page: what happened, and what is on today.
type startPage struct{ Summary *homeSummary }

// home is the start page: what changed recently and what is on today.
func (s *Server) home(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("home.title")
	v.Data = startPage{Summary: s.homeSummary(r, v)}
	s.render(w, r, http.StatusOK, "start", v)
}

// wikiHome lists the visible spaces, each with who else sees it.
func (s *Server) wikiHome(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	data := homePage{Audiences: map[string]audience{}}
	for _, sp := range v.Nav {
		data.Audiences[sp.Slug] = s.spaceAudience(v, sp)
	}
	if name := r.URL.Query().Get("deleted"); name != "" && !s.spaceNamed(name) {
		data.Deleted = name
	}
	v.Data = data
	s.render(w, r, http.StatusOK, "wiki", v)
}

// homeSummary sums up the window the request asks for.
func (s *Server) homeSummary(r *http.Request, v view) *homeSummary {
	_, p := auth.UserFrom(r.Context())
	q := r.URL.Query()
	rng, own, now := pickRange(q.Get("range")), q.Get("own") == "1", time.Now()
	from := s.windowStart(p.Sub, rng, now)
	rb := s.newReport(v, reportCtx{v: v, p: p, g: s.groups(), scope: scopeSeen})
	sum := reports.Summarize(readableDocs(rb.docs), s.recentChanges(rb), from, now, p.Sub, own)
	h := &homeSummary{Range: rng, Own: own, HistoryError: rb.page.HistoryError, Empty: sum.Total == 0}
	h.Ranges, h.OwnHref = homeLinks(v, rng, own)
	h.Sentence = homeSentence(v, sum, from, now)
	h.Today = s.today(v, p.Sub, rb.rc.g, now)
	data := map[string]any{}
	if !h.Empty {
		h.Tiles = homeTiles(v, sum, data)
		h.Cards = []reportCard{bandCard(v, sum, data), s.peopleCard(v, sum, data)}
		h.Groups, h.More = s.homeGroups(v, sum.Docs, now)
	}
	js, _ := json.Marshal(data) // json escapes <, > and &: safe inside <script>
	h.JSON = template.JS(js)
	return h
}

// pickRange reads the window; anything unknown is the default.
func pickRange(q string) string {
	for _, r := range homeRanges {
		if r.Key == q {
			return q
		}
	}
	return ""
}

// windowStart is where the summary begins. Only the default window counts as
// a visit, so trying another window does not move it.
func (s *Server) windowStart(sub, rng string, now time.Time) time.Time {
	for _, r := range homeRanges {
		if r.Key == rng && r.D > 0 {
			return now.Add(-r.D)
		}
	}
	prev, err := s.Visits.Touch(sub, now)
	if err != nil {
		slog.Warn("home: remembering the visit", "err", err)
	}
	return visits.Since(prev, now)
}

// homeLinks are the select's options and the link that toggles own changes.
func homeLinks(v view, rng string, own bool) ([]homeOption, string) {
	var opts []homeOption
	for _, r := range homeRanges {
		opts = append(opts, homeOption{r.Key, v.T("home.range_" + cmp.Or(r.Key, "visit")), r.Key == rng})
	}
	q := url.Values{}
	if rng != "" {
		q.Set("range", rng)
	}
	if !own {
		q.Set("own", "1")
	}
	href := "/"
	if len(q) > 0 {
		href += "?" + q.Encode()
	}
	return opts, href
}

// readableDocs are the documents the viewer may read: the others have no title.
func readableDocs(docs []reports.Doc) []reports.Doc {
	var out []reports.Doc
	for _, d := range docs {
		if d.Title != "" {
			out = append(out, d)
		}
	}
	return out
}

// recentChanges are the changes to the readable documents: commits on their
// files with the author as person, and list items written or ticked, whose
// person nobody records.
func (s *Server) recentChanges(rb *reportBuild) []reports.Change {
	byPath, lists := map[string]reports.Doc{}, map[string]bool{}
	for _, d := range readableDocs(rb.docs) {
		if d.Kind == reports.List {
			lists[d.ID] = true
		} else {
			byPath[filepath.ToSlash(d.Path)] = d
		}
	}
	subs := s.subsByEmail()
	var out []reports.Change
	for _, c := range rb.rc.scan.Changes {
		if d, ok := byPath[c.Path]; ok && !c.Deleted {
			out = append(out, reports.Change{Kind: d.Kind, DocID: d.ID, When: c.When, Person: subs[strings.ToLower(c.Email)]})
		}
	}
	return append(out, s.listChanges(lists)...)
}

// listChanges are the items written and ticked on the lists named.
func (s *Server) listChanges(lists map[string]bool) []reports.Change {
	var out []reports.Change
	for _, l := range s.Todos.All() {
		if !lists[l.ID] {
			continue
		}
		for _, it := range l.Items {
			out = append(out, reports.Change{Kind: reports.List, DocID: l.ID, When: time.UnixMilli(it.At.Text)})
			if it.Done && it.At.Done > it.At.Text {
				out = append(out, reports.Change{Kind: reports.List, DocID: l.ID, When: time.UnixMilli(it.At.Done), Ticked: true})
			}
		}
	}
	return out
}

// subsByEmail maps the people's e-mail addresses, lower case, to their subs.
func (s *Server) subsByEmail() map[string]string {
	out := map[string]string{}
	for _, u := range s.Users.List() {
		if u.Email != "" {
			out[strings.ToLower(u.Email)] = u.Sub
		}
	}
	return out
}

// homeSentence says the window and what happened in it.
func homeSentence(v view, sum reports.Summary, from, now time.Time) string {
	when := homeWhen(v, from, now)
	var parts []string
	for _, k := range sum.Kinds {
		parts = append(parts, kindParts(v, k)...)
	}
	if len(parts) == 0 {
		return v.T("home.nothing", when)
	}
	return v.T("home.sentence", when, strings.Join(parts, ", "))
}

// kindParts are the phrases for one kind: "2 neue Seiten", "1 Seite geändert".
func kindParts(v view, k reports.KindSum) []string {
	counts := []struct {
		key string
		n   int
	}{{"new", k.New}, {"changed", k.Changed}}
	if k.Kind == reports.List {
		counts = []struct {
			key string
			n   int
		}{{"written", sparkSum(k.Spark) - k.Ticked}, {"ticked", k.Ticked}}
	}
	var out []string
	for _, c := range counts {
		key := "home.part_" + string(k.Kind) + "_" + c.key
		switch {
		case c.n == 1:
			out = append(out, v.T(key+"_one"))
		case c.n > 1:
			out = append(out, v.T(key, c.n))
		}
	}
	return out
}

func sparkSum(xs []int) int {
	n := 0
	for _, x := range xs {
		n += x
	}
	return n
}

// homeWhen writes a time near now the way people say it: "heute, 18:40".
func homeWhen(v view, t, now time.Time) string {
	t, now = t.Local(), now.Local()
	clock := t.Format("15:04")
	day := func(x time.Time) time.Time { return time.Date(x.Year(), x.Month(), x.Day(), 0, 0, 0, 0, x.Location()) }
	switch day(now).Sub(day(t)) {
	case 0:
		return v.T("home.today_at", clock)
	case 24 * time.Hour:
		return v.T("home.yesterday_at", clock)
	}
	wd := v.T(fmt.Sprintf("reports.wd_%d", (int(t.Weekday())+6)%7))
	return v.T("home.date_at", wd, t.Format(v.T("home.date_layout")), clock)
}
