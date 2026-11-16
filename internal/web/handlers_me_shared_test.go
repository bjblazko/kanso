// SPDX-License-Identifier: Apache-2.0

package web

import (
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
		`href="/s/haus"`, "Sichtbar für dich, Admin",
		`href="/todos/` + shared.ID + `"`, "Baumarkt",
		"Putzplan", `href="/mealplan/` + plan + `"`, "Familienküche", "Sichtbar für dich, Oma</span>",
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
