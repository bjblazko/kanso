// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/todos"
)

const gardenID = "gartenaaaaaa"

// annaGarden makes anna's list "Gartenarbeit" with one item.
func annaGarden(t *testing.T, ts *testServer) {
	t.Helper()
	ops := []todos.Op{
		{ID: "op1", Kind: "create", List: gardenID, Text: "Gartenarbeit", At: 1},
		{ID: "op2", Kind: "add", List: gardenID, Item: "itemaaaaaaaa", Text: "Laub wegfegen", At: 1},
	}
	if rejected := ts.deps.Todos.Apply("anna", access.Groups{}, ops); len(rejected) > 0 {
		t.Fatalf("rejected %v", rejected)
	}
}

type syncAnswer struct {
	Rejected []string  `json:"rejected"`
	Lists    []listOut `json:"lists"`
}

// syncTodos posts ops like todos.js does.
func syncTodos(t *testing.T, ts *testServer, c *http.Client, body string, withToken bool) (*http.Response, syncAnswer) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/todos")
	req, _ := http.NewRequest("POST", ts.URL+"/api/todos/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", ts.URL)
	if withToken {
		req.Header.Set("X-CSRF-Token", between(page, `<meta name="csrf" content="`, `"`))
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	var a syncAnswer
	_ = json.NewDecoder(resp.Body).Decode(&a)
	return resp, a
}

func TestTodosAPIListsOnlyVisible(t *testing.T) {
	ts := newTestServer(t)
	anna, ben := ts.client(t, "anna", "Anna", false), ts.client(t, "ben", "Ben", true)
	annaGarden(t, ts)
	_, body := get(t, anna, ts.URL+"/api/todos")
	if !strings.Contains(body, `"name":"Gartenarbeit"`) || !strings.Contains(body, `"mine":true`) {
		t.Fatalf("anna: %s", body)
	}
	if _, body := get(t, ben, ts.URL+"/api/todos"); strings.Contains(body, "Gartenarbeit") {
		t.Fatalf("ben (admin) sees anna's list: %s", body)
	}
}

func TestSyncAppliesAndReturnsState(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	annaGarden(t, ts)
	resp, a := syncTodos(t, ts, anna, `{"ops":[{"id":"op3","kind":"done","list":"`+gardenID+`","item":"itemaaaaaaaa","done":true,"at":2}]}`, true)
	if resp.StatusCode != http.StatusOK || len(a.Rejected) != 0 || len(a.Lists) != 1 || !a.Lists[0].Items[0].Done {
		t.Fatalf("%d %+v", resp.StatusCode, a)
	}
}

func TestSyncRejectsForeignList(t *testing.T) {
	ts := newTestServer(t)
	ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	annaGarden(t, ts)
	_, a := syncTodos(t, ts, ben, `{"ops":[{"id":"op3","kind":"done","list":"`+gardenID+`","item":"itemaaaaaaaa","done":true,"at":2}]}`, true)
	if len(a.Rejected) != 1 || a.Rejected[0] != "op3" || len(a.Lists) != 0 {
		t.Fatalf("%+v", a)
	}
}

func TestSyncNeedsCSRF(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if resp, _ := syncTodos(t, ts, anna, `{"ops":[]}`, false); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestSyncTooManyOps(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ops := make([]string, todos.MaxOps+1)
	for i := range ops {
		ops[i] = fmt.Sprintf(`{"id":"op%d","kind":"done","list":"%s","at":1}`, i, gardenID)
	}
	if resp, _ := syncTodos(t, ts, anna, `{"ops":[`+strings.Join(ops, ",")+`]}`, true); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

// postTodoForm sends a form to a todo page with the CSRF token of the page.
func postTodoForm(t *testing.T, ts *testServer, c *http.Client, path string, form url.Values) *http.Response {
	t.Helper()
	_, page := get(t, c, ts.URL+"/todos")
	form.Set("csrf", between(page, `<meta name="csrf" content="`, `"`))
	req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp
}

func TestShareOwnerOnly(t *testing.T) {
	ts := newTestServer(t)
	ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", true)
	annaGarden(t, ts)
	if resp, _ := get(t, ben, ts.URL+"/todos/"+gardenID+"/share"); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("GET status %d", resp.StatusCode)
	}
	form := url.Values{"action": {"add"}, "kind": {"user"}, "who": {"ben"}}
	if resp := postTodoForm(t, ts, ben, "/todos/"+gardenID+"/share", form); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("POST status %d", resp.StatusCode)
	}
}

func TestShareAddGroupAndPerson(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ts.client(t, "ben", "Ben", false)
	if err := access.SaveGroups(ts.deps.Cfg.DataDir, access.Groups{"Familie": {"anna", "ben"}}); err != nil {
		t.Fatal(err)
	}
	annaGarden(t, ts)
	for _, f := range []url.Values{
		{"action": {"add"}, "kind": {"user"}, "who": {"ben"}},
		{"action": {"add"}, "kind": {"group"}, "who": {"Familie"}},
		{"action": {"add"}, "kind": {"user"}, "who": {"nobody"}},
	} {
		postTodoForm(t, ts, anna, "/todos/"+gardenID+"/share", f)
	}
	l, _ := ts.deps.Todos.Get("anna", gardenID, access.Groups{})
	if len(l.Shared) != 2 || l.Shared[0].User != "ben" || l.Shared[1].Group != "Familie" {
		t.Fatalf("shared %+v", l.Shared)
	}
	_, page := get(t, anna, ts.URL+"/todos/"+gardenID+"/share")
	if !strings.Contains(page, "Ben") || !strings.Contains(page, "Familie") {
		t.Fatalf("share page: %s", page)
	}
}

func TestDeleteListOwnerOnly(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ben := ts.client(t, "ben", "Ben", false)
	annaGarden(t, ts)
	if err := ts.deps.Todos.SetShares("anna", gardenID, []todos.Share{{User: "ben"}}); err != nil {
		t.Fatal(err)
	}
	if resp := postTodoForm(t, ts, ben, "/todos/"+gardenID+"/delete", url.Values{}); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("ben: %d", resp.StatusCode)
	}
	_, page := get(t, anna, ts.URL+"/todos/"+gardenID+"/delete")
	if !strings.Contains(page, "Gartenarbeit") {
		t.Fatalf("confirm page: %s", page)
	}
	if resp := postTodoForm(t, ts, anna, "/todos/"+gardenID+"/delete", url.Values{}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("anna: %d", resp.StatusCode)
	}
	if _, err := ts.deps.Todos.Get("anna", gardenID, access.Groups{}); err == nil {
		t.Fatal("list still there")
	}
}

func TestRailHasTodos(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	for _, path := range []string{"/todos", "/todos/" + gardenID} {
		resp, page := get(t, anna, ts.URL+path)
		if resp.StatusCode != http.StatusOK || !strings.Contains(page, `<a href="/todos" aria-current="true">`) {
			t.Fatalf("%s: %d", path, resp.StatusCode)
		}
	}
	_, page := get(t, anna, ts.URL+"/todos")
	var msgs map[string]string
	raw := between(page, `<script type="application/json" id="todo-messages">`, `</script>`)
	if err := json.Unmarshal([]byte(raw), &msgs); err != nil || msgs["progress"] != "{done} von {total} erledigt" {
		t.Fatalf("messages %q: %v", raw, err)
	}
}

// todos.js reads items and shares as arrays, also for a new, empty list.
func TestNewListHasEmptyArrays(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	syncTodos(t, ts, anna, `{"ops":[{"id":"op1","kind":"create","list":"`+gardenID+`","text":"Leer","at":1}]}`, true)
	_, body := get(t, anna, ts.URL+"/api/todos")
	if !strings.Contains(body, `"items":[]`) || !strings.Contains(body, `"shared":[]`) {
		t.Fatalf("body %s", body)
	}
}
