// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

func (ts *testServer) note(t *testing.T, owner, body string) string {
	t.Helper()
	n, _, err := ts.deps.Notes.Create(owner, store.NewID(), body, nil, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	return n.Meta.ID
}

func TestReportsMenuEntry(t *testing.T) {
	ts, _, anna := setupHaus(t)
	if _, body := get(t, anna, ts.URL+"/"); !strings.Contains(body, `href="/reports"`) {
		t.Error("user menu lacks Berichte")
	}
	resp, _ := get(t, anna, ts.URL+"/reports")
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/reports/overview" {
		t.Errorf("/reports: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestReportsOverviewOwn(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.note(t, "anna", "Einkauf")
	ts.note(t, "anna", "Ideen")
	ts.note(t, "admin", "Geheim")
	ts.createPage(t, anna, "haus", "", "Keller")
	resp, body := get(t, anna, ts.URL+"/reports/overview")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	main := htmlUnescape(between(body, "<main", "</main>"))
	for _, want := range []string{
		`id="report-data"`, "Admins sehen von dir nur diese Zahlen",
		`<td>Notizen</td><td class="n">2</td>`, `<td>Seiten</td><td class="n">1</td>`,
		`href="/reports/overview?scope=own" aria-current="page"`,
	} {
		if !strings.Contains(main, want) {
			t.Errorf("overview lacks %q", want)
		}
	}
	if strings.Contains(main, "Geheim") || strings.Contains(main, "scope=all") {
		t.Error("anna's report shows someone else's note or the admin scope")
	}
}

func TestReportsScopeAllAdminOnly(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.note(t, "anna", "Einkauf")
	if code := status(t, anna, ts.URL+"/reports/overview?scope=all"); code != http.StatusForbidden {
		t.Errorf("anna: %d", code)
	}
	resp, body := get(t, admin, ts.URL+"/reports/overview")
	main := htmlUnescape(between(body, "<main", "</main>"))
	if resp.StatusCode != http.StatusOK || !strings.Contains(main, `scope=all" aria-current="page"`) || !strings.Contains(main, "<td>Anna</td>") {
		t.Errorf("admin default is not everything by person: %d", resp.StatusCode)
	}
	if strings.Contains(main, "Admins sehen von dir") {
		t.Error("admin gets the note meant for others")
	}
}

func TestReportsAllHidesUnreadableTitles(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.createPage(t, anna, "haus", "", "Tagebuch")
	boss := otherAdmin(t, ts)
	_, body := get(t, boss, ts.URL+"/reports/overview?scope=all")
	if strings.Contains(body, "Tagebuch") {
		t.Error("admin sees the title of a page they cannot read")
	}
}

func TestReportsUnknownTabAndScope(t *testing.T) {
	ts, _, anna := setupHaus(t)
	if code := status(t, anna, ts.URL+"/reports/nope"); code != http.StatusNotFound {
		t.Errorf("unknown tab: %d", code)
	}
	resp, body := get(t, anna, ts.URL+"/reports/overview?scope=xyz")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `scope=own" aria-current="page"`) {
		t.Errorf("unknown scope: %d", resp.StatusCode)
	}
}

func TestReportsCacheScansOncePerHead(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	var scans atomic.Int32
	real := ts.srv.scanRepo
	ts.srv.scanRepo = func() (store.Scan, error) {
		scans.Add(1)
		time.Sleep(50 * time.Millisecond)
		return real()
	}
	var wg sync.WaitGroup
	for range 3 {
		wg.Go(func() { get(t, admin, ts.URL+"/reports/overview") })
	}
	wg.Wait()
	if n := scans.Load(); n != 1 {
		t.Errorf("%d scans for one HEAD", n)
	}
	ts.createPage(t, admin, "haus", "", "Neu")
	get(t, admin, ts.URL+"/reports/overview")
	if n := scans.Load(); n != 2 {
		t.Errorf("%d scans after a commit", n)
	}
}

func TestReportsGitErrorStillRenders(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.srv.scanRepo = func() (store.Scan, error) { return store.Scan{}, errors.New("broken") }
	resp, body := get(t, anna, ts.URL+"/reports/overview")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Historie gerade nicht lesbar") {
		t.Errorf("git error: %d", resp.StatusCode)
	}
}
