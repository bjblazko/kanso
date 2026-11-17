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

// attachments gives page id the files a.png (linked from its text) and lost.png (not linked).
func (ts *testServer) attachments(t *testing.T, id string) {
	t.Helper()
	a := store.Author{Name: "Anna", Email: "anna@users.kanso"}
	for _, name := range []string{"a.png", "lost.png"} {
		if _, err := ts.deps.Wiki.AddAttachment(a, id, name, strings.NewReader("png-bytes"), 1<<20); err != nil {
			t.Fatal(err)
		}
	}
	if err := ts.deps.Wiki.Save(a, id, "Keller", "![](attachments/a.png)"); err != nil {
		t.Fatal(err)
	}
}

func TestReportsStorageTab(t *testing.T) {
	ts, _, anna := setupHaus(t)
	id := ts.createPage(t, anna, "haus", "", "Keller")
	ts.attachments(t, id)
	resp, body := get(t, anna, ts.URL+"/reports/storage")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%d", resp.StatusCode)
	}
	orphans := between(body, `id="card-orphans"`, "</section>")
	if !strings.Contains(orphans, `href="/p/`+id+`/attachments/lost.png"`) || strings.Contains(orphans, "a.png") {
		t.Errorf("orphans:\n%s", orphans)
	}
	if tree := between(body, `id="card-areas"`, "</section>"); !strings.Contains(tree, "<td>Haus</td>") {
		t.Errorf("areas:\n%s", tree)
	}
	if strings.Contains(body, `id="card-history"`) {
		t.Error("a non-admin gets the repository over time")
	}
}

func TestReportsStorageAdminSeesNoForeignNames(t *testing.T) {
	ts, _, anna := setupHaus(t)
	id := ts.createPage(t, anna, "haus", "", "Keller")
	ts.attachments(t, id)
	boss := otherAdmin(t, ts)
	_, body := get(t, boss, ts.URL+"/reports/storage?scope=all")
	main := htmlUnescape(between(body, "<main", "</main>"))
	if strings.Contains(main, "lost.png") || strings.Contains(main, "Keller") || !strings.Contains(main, "– (kein Zugriff)") {
		t.Errorf("admin sees names of things they cannot read:\n%s", between(main, `id="card-orphans"`, "</section>"))
	}
	if !strings.Contains(main, `id="card-history"`) {
		t.Error("admin lacks the repository over time")
	}
}

func TestReportsOverviewServerTileAdminOnly(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	if _, body := get(t, admin, ts.URL+"/reports/overview"); !strings.Contains(body, "Arbeitsspeicher") {
		t.Error("admin lacks the server tile")
	}
	if _, body := get(t, anna, ts.URL+"/reports/overview"); strings.Contains(body, "Arbeitsspeicher") {
		t.Error("anna gets the server tile")
	}
}

func TestReportsActivityTab(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.createPage(t, admin, "haus", "", "Vom Admin")
	_, body := get(t, anna, ts.URL+"/reports/activity")
	if cal := between(body, `id="card-calendar"`, "</section>"); !strings.Contains(cal, "Noch nichts da.") {
		t.Errorf("anna's own activity counts the admin's page:\n%s", cal)
	}
	ts.note(t, "anna", "Heute")
	_, body = get(t, anna, ts.URL+"/reports/activity")
	today := time.Now().Format("02.01.2006")
	if cal := between(body, `id="card-calendar"`, "</section>"); !strings.Contains(cal, "<td>"+today+"</td>") {
		t.Errorf("today's note missing:\n%s", cal)
	}
	if !strings.Contains(body, `id="card-weekhours"`) {
		t.Error("no weekday × hour card")
	}
}
