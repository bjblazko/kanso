// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/todos"
)

func TestSharedOverviewListsWhatOthersSee(t *testing.T) {
	ts, admin, anna := setupHaus(t) // Haus: admin owns it, group familie (anna) writes
	ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"add"}, "group": {"familie"}, "sub": {"oma"}})
	shared, _ := ts.deps.Todos.Create("anna", "Baumarkt", "")
	if err := ts.deps.Todos.SetShares("anna", shared.ID, []todos.Share{{User: "admin"}}); err != nil {
		t.Fatal(err)
	}
	ts.deps.Todos.Create("anna", "Geheimliste", "")
	if _, err := ts.deps.Tasks.CreateGroupBoard(store.Author{Name: "A", Email: "a@example.org"}, "Putzplan", "familie"); err != nil {
		t.Fatal(err)
	}
	plan := ts.familyPlan(t)
	_, body := get(t, anna, ts.URL+"/me/shared")
	main := between(body, "<main", "</main>")
	for _, want := range []string{
		`href="/s/haus"`, "Verwalten: Admin · Schreiben: Oma",
		`href="/todos/` + shared.ID + `"`, "Baumarkt",
		"Putzplan", `href="/mealplan/` + plan + `"`, "Familienküche", "Schreiben: Oma</span>",
	} {
		if !strings.Contains(main, want) {
			t.Errorf("overview lacks %q", want)
		}
	}
	if strings.Contains(main, "Geheimliste") || strings.Contains(main, "Nur für dich sichtbar") {
		t.Error("overview lists something only anna sees")
	}
}

func TestSharedOverviewEmpty(t *testing.T) {
	ts := newTestServer(t)
	ben := ts.client(t, "ben", "Ben", false)
	_, body := get(t, ben, ts.URL+"/me/shared")
	if !strings.Contains(body, "Nichts von dir ist für andere sichtbar.") {
		t.Error("empty overview does not say so")
	}
	if !strings.Contains(body, `href="/me/shared"`) {
		t.Error("user menu lacks the overview")
	}
}

func TestBoardOfAGroupOfOneIsOnlyYours(t *testing.T) {
	ts, _, anna := setupHaus(t) // familie has only anna
	b, err := ts.deps.Tasks.CreateGroupBoard(store.Author{Name: "A", Email: "a@example.org"}, "Putzplan", "familie")
	if err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, anna, ts.URL+"/tasks/b/"+b.ID); !strings.Contains(body, "Nur für dich sichtbar") {
		t.Error("a board only anna sees does not say so")
	}
}

// sharedFixture: anna's Haus (admin, familie = anna+oma), list Baumarkt shared
// with admin, board Putzplan and the family plan for familie.
func sharedFixture(t *testing.T) (*testServer, *http.Client) {
	t.Helper()
	ts, admin, anna := setupHaus(t)
	ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/admin/groups", "/admin/groups", url.Values{"action": {"add"}, "group": {"familie"}, "sub": {"oma"}})
	shared, _ := ts.deps.Todos.Create("anna", "Baumarkt", "")
	if err := ts.deps.Todos.SetShares("anna", shared.ID, []todos.Share{{User: "admin"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := ts.deps.Tasks.CreateGroupBoard(store.Author{Name: "A", Email: "a@example.org"}, "Putzplan", "familie"); err != nil {
		t.Fatal(err)
	}
	ts.familyPlan(t)
	ts.client(t, "ute", "Ute", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"ute"}, "role": {"viewer"}})
	return ts, anna
}

func TestSharedOverviewByPerson(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=person")
	main := between(body, "<main", "</main>")
	if !strings.Contains(main, `href="/me/shared?by=person" aria-current="page"`) {
		t.Error("person view not marked current")
	}
	admin := between(main, "<h2>Admin</h2>", "</section>")
	oma := between(main, "<h2>Oma</h2>", "</section>")
	for _, want := range []string{"Haus", "Baumarkt", "Liste"} {
		if !strings.Contains(admin, want) {
			t.Errorf("Admin lacks %q", want)
		}
	}
	if strings.Contains(admin, "Putzplan") || strings.Contains(admin, "Familienküche") {
		t.Error("Admin lists what only the family sees")
	}
	for _, want := range []string{"Haus", "Putzplan", "Familienküche"} {
		if !strings.Contains(oma, want) {
			t.Errorf("Oma lacks %q", want)
		}
	}
	if strings.Contains(main, "<h2>Anna</h2>") {
		t.Error("the viewer is listed as a person")
	}
}

func TestSharedOverviewByDocument(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=doc")
	main := between(body, "<main", "</main>")
	order := []string{"Baumarkt", "Familienküche", "Haus", "Putzplan"}
	last := -1
	for _, name := range order {
		i := strings.Index(main, ">"+name+"</a>")
		if i < last {
			t.Errorf("%q out of order", name)
		}
		last = i
	}
	if strings.Contains(main, "<h2>") || !strings.Contains(main, "Kochplan") || !strings.Contains(main, "Schreiben: Admin") {
		t.Error("document view should be one list with kind and audience")
	}
}

func TestSharedOverviewDefaultsToKind(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=nonsense")
	if !strings.Contains(body, `href="/me/shared" aria-current="page"`) || !strings.Contains(body, "<h2>Listen</h2>") {
		t.Error("unknown grouping does not fall back to kind")
	}
}

func TestSharedOverviewByRole(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=role")
	main := between(body, "<main", "</main>")
	manage := between(main, "<h2>Verwalten</h2>", "</section>")
	write := between(main, "<h2>Schreiben</h2>", "</section>")
	read := between(main, "<h2>Lesen</h2>", "</section>")
	if !strings.Contains(manage, "Haus") || !strings.Contains(manage, "Admin") || strings.Contains(manage, "Baumarkt") {
		t.Errorf("Verwalten: %s", manage)
	}
	for _, want := range []string{"Baumarkt", "Putzplan", "Familienküche", "Haus", "Oma"} {
		if !strings.Contains(write, want) {
			t.Errorf("Schreiben lacks %q", want)
		}
	}
	if !strings.Contains(read, "Haus") || !strings.Contains(read, "Ute") || strings.Contains(read, "Putzplan") {
		t.Errorf("Lesen: %s", read)
	}
	if strings.Index(main, "<h2>Verwalten</h2>") > strings.Index(main, "<h2>Lesen</h2>") {
		t.Error("roles not ordered from most to least")
	}
}

func TestSharedOverviewPersonShowsRole(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=person")
	ute := between(body, "<h2>Ute</h2>", "</section>")
	if !strings.Contains(ute, "Haus") || !strings.Contains(ute, "Lesen") {
		t.Errorf("Ute: %s", ute)
	}
}

func TestSharedOverviewFilterByRole(t *testing.T) {
	ts, anna := sharedFixture(t)
	_, body := get(t, anna, ts.URL+"/me/shared?by=person&role=viewer")
	main := between(body, "<main", "</main>")
	if !strings.Contains(main, `href="/me/shared?by=person&amp;role=viewer" aria-current="page"`) {
		t.Error("filter not marked current, or grouping lost")
	}
	if !strings.Contains(main, "<h2>Ute</h2>") || strings.Contains(main, "<h2>Oma</h2>") || strings.Contains(main, "Baumarkt") {
		t.Errorf("filter keeps the wrong people or things:\n%s", main)
	}
	_, body = get(t, anna, ts.URL+"/me/shared?role=editor")
	main = between(body, "<main", "</main>")
	if !strings.Contains(main, "Schreiben: Oma") || strings.Contains(main, "Verwalten: Admin") || strings.Contains(main, "Lesen: Ute") {
		t.Errorf("kind view does not narrow the lines to the role:\n%s", main)
	}
	if !strings.Contains(main, `href="/me/shared?by=person&amp;role=editor"`) {
		t.Error("grouping tabs drop the filter")
	}
}
