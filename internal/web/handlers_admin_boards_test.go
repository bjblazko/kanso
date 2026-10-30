// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

func adminBoards(t *testing.T, ts *testServer, c *http.Client, f url.Values) (*http.Response, string) {
	t.Helper()
	return postTaskForm(t, ts, c, "/admin/boards", f, "")
}

func TestAdminBoardsOnlyAdmins(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if resp, _ := get(t, anna, ts.URL+"/admin/boards"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("GET %d", resp.StatusCode)
	}
	if resp, _ := adminBoards(t, ts, anna, url.Values{"action": {"create"}, "name": {"X"}, "group": {"familie"}}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST %d", resp.StatusCode)
	}
}

func TestAdminCreatesGroupBoard(t *testing.T) {
	ts := newTestServer(t)
	boss := ts.client(t, "boss", "Boss", true)
	anna := ts.client(t, "anna", "Anna", false)
	if err := access.SaveGroups(ts.deps.Cfg.DataDir, access.Groups{"familie": {"anna"}}); err != nil {
		t.Fatal(err)
	}
	if _, body := get(t, boss, ts.URL+"/admin/boards"); !strings.Contains(body, `<option value="familie">`) {
		t.Fatalf("no group choice:\n%s", body)
	}
	if resp, _ := adminBoards(t, ts, boss, url.Values{"action": {"create"}, "name": {"Familie"}, "group": {"nobody"}}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("unknown group: %d", resp.StatusCode)
	}
	resp, _ := adminBoards(t, ts, boss, url.Values{"action": {"create"}, "name": {"Familie"}, "group": {"familie"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create: %d", resp.StatusCode)
	}
	if _, body := get(t, anna, ts.URL+"/tasks"); !strings.Contains(body, ">Familie</a>") {
		t.Fatalf("anna does not see it:\n%s", body)
	}
}

func TestAdminRenamesAndDeletesGroupBoard(t *testing.T) {
	ts := newTestServer(t)
	boss := ts.client(t, "boss", "Boss", true)
	id := familyBoardID(t, ts)
	adminBoards(t, ts, boss, url.Values{"action": {"rename"}, "id": {id}, "name": {"Haushalt"}})
	if gb := ts.deps.Tasks.GroupBoards(); len(gb) != 1 || gb[0].Name != "Haushalt" {
		t.Fatalf("rename: %+v", gb)
	}
	resp, body := adminBoards(t, ts, boss, url.Values{"action": {"delete"}, "id": {id}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "„Haushalt“ löschen?") || len(ts.deps.Tasks.GroupBoards()) != 1 {
		t.Fatalf("confirm step: %d", resp.StatusCode)
	}
	adminBoards(t, ts, boss, url.Values{"action": {"delete_confirmed"}, "id": {id}})
	if gb := ts.deps.Tasks.GroupBoards(); len(gb) != 0 {
		t.Fatalf("not deleted: %+v", gb)
	}
}

func TestAdminBoardsListHidesPrivateBoards(t *testing.T) {
	ts := newTestServer(t)
	boss := ts.client(t, "boss", "Boss", true)
	ts.client(t, "anna", "Anna", false)
	own := ownBoardID(t, ts, "anna")
	_, body := get(t, boss, ts.URL+"/admin/boards")
	if strings.Contains(body, own) {
		t.Fatal("private board listed")
	}
	if resp, _ := adminBoards(t, ts, boss, url.Values{"action": {"delete_confirmed"}, "id": {own}}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("delete private: %d", resp.StatusCode)
	}
}
