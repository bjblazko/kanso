// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/todos"
)

// shoppingList makes a shopping list of owner with the given items.
func (ts *testServer) shoppingList(t *testing.T, owner, name string, texts ...string) string {
	t.Helper()
	l, err := ts.deps.Todos.Create(owner, name, todos.KindShopping)
	if err != nil {
		t.Fatal(err)
	}
	_ = ts.deps.Todos.Change(owner, nil, l.ID, func(l *todos.List) error {
		for _, tx := range texts {
			l.Items = append(l.Items, todos.NewItem(tx, "", time.Now()))
		}
		return nil
	})
	return l.ID
}

func TestShopPreview(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	list := ts.shoppingList(t, "admin", "Wocheneinkauf")
	plain, _ := ts.deps.Todos.Create("admin", "Gartenarbeit", "")
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/shop?servings=6")
	for _, want := range []string{"1200 g Kartoffeln", `name="take" value="0" checked`, `value="` + list + `"`, "Wocheneinkauf", `value="new"`} {
		if !strings.Contains(body, want) {
			t.Errorf("preview lacks %q", want)
		}
	}
	if strings.Contains(body, plain.ID) {
		t.Error("a plain todo list is offered")
	}
}

func TestShopMergePreview(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	list := ts.shoppingList(t, "admin", "Wocheneinkauf", "200 g Kartoffeln")
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/shop?list="+list)
	if !strings.Contains(body, "200 g Kartoffeln → 1 kg Kartoffeln") {
		t.Fatalf("no merge shown:\n%s", between(body, "<main", "</main>"))
	}
}

func postShop(t *testing.T, ts *testServer, c *http.Client, id string, form url.Values) *http.Response {
	t.Helper()
	return ts.postForm(t, c, "/recipes/"+id+"/shop", "/recipes/"+id+"/shop", form)
}

func TestShopApplyNewList(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	resp := postShop(t, ts, admin, id, url.Values{"servings": {"4"}, "list": {"new"}, "name": {"Samstag"}, "take": {"0", "1"}})
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/todos/") {
		t.Fatalf("%d %q", resp.StatusCode, loc)
	}
	l, err := ts.deps.Todos.Get("admin", strings.TrimPrefix(loc, "/todos/"), nil)
	if err != nil || l.Kind != todos.KindShopping || l.Name != "Samstag" || len(l.Items) != 2 {
		t.Fatalf("%+v %v", l, err)
	}
	if it := l.Items[0]; it.Text != "800 g Kartoffeln" || it.Note != "festkochend" || it.For[0].Title != "Curry" || it.For[0].Recipe != id {
		t.Fatalf("%+v", it)
	}
}

func TestShopApplyExisting(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	list := ts.shoppingList(t, "admin", "Wocheneinkauf", "200 g Kartoffeln")
	resp := postShop(t, ts, admin, id, url.Values{"servings": {"4"}, "list": {list}, "take": {"0"}})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/todos/"+list {
		t.Fatalf("%d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	l, _ := ts.deps.Todos.Get("admin", list, nil)
	if len(l.Items) != 1 || l.Items[0].Text != "1 kg Kartoffeln" || l.Items[0].For[0].Amount != "800 g" {
		t.Fatalf("%+v", l.Items)
	}
}

func TestShopAccess(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	foreign := ts.shoppingList(t, "admin", "Admins Liste")
	if code := status(t, anna, ts.URL+"/recipes/"+id+"/shop?list="+foreign); code != http.StatusNotFound {
		t.Errorf("foreign list in preview: %d", code)
	}
	if resp := postShop(t, ts, anna, id, url.Values{"list": {foreign}, "take": {"0"}}); resp.StatusCode != http.StatusNotFound {
		t.Errorf("foreign list: %d", resp.StatusCode)
	}
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	secret := ts.makeRecipe(t, admin, "geheim", "Tresorsuppe", curryBody, 2)
	if code := status(t, anna, ts.URL+"/recipes/"+secret+"/shop"); code != http.StatusNotFound {
		t.Errorf("unreadable recipe: %d", code)
	}
	plain := ts.createPage(t, admin, "haus", "", "Heizung")
	if code := status(t, admin, ts.URL+"/recipes/"+plain+"/shop"); code != http.StatusNotFound {
		t.Errorf("plain page: %d", code)
	}
}

func TestRecipeToolbarHasShop(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	if _, body := get(t, admin, ts.URL+"/p/"+id); !strings.Contains(toolbarOf(body), `href="/recipes/`+id+`/shop"`) {
		t.Error("page toolbar lacks Einkaufen")
	}
	if _, body := get(t, admin, ts.URL+"/recipes/"+id+"/cook?servings=6"); !strings.Contains(toolbarOf(body), `href="/recipes/`+id+`/shop?servings=6"`) {
		t.Error("cook toolbar lacks Einkaufen with servings")
	}
}

func TestShopOneFormCarriesWhatIsTyped(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/shop")
	post := between(body, `<form method="post"`, `</form>`)
	for _, want := range []string{`id="servings" name="servings"`, `id="list" name="list"`, `name="fill" value="`, `name="refresh"`, `Neue Liste „Einkauf: Curry“`} {
		if !strings.Contains(post, want) {
			t.Errorf("post form lacks %s", want)
		}
	}
	if strings.Contains(body, `<form method="get"`) {
		t.Error("a second form holds the choices")
	}
}

func TestShopRefreshKeepsUnticks(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	resp, body := ts.postFormBody(t, admin, "/recipes/"+id+"/shop", "/recipes/"+id+"/shop",
		url.Values{"servings": {"8"}, "list": {"new"}, "name": {"Samstag"}, "refresh": {"1"}, "take": {"1"}})
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "1600 g Kartoffeln") || !strings.Contains(body, `value="Samstag"`) {
		t.Fatalf("%d", resp.StatusCode)
	}
	if strings.Contains(body, `name="take" value="0" checked`) || !strings.Contains(body, `name="take" value="1" checked`) {
		t.Error("refresh lost the unticks")
	}
	if len(ts.shoppingLists(t, "admin")) != 0 {
		t.Error("refresh created a list")
	}
}

func TestShopTwiceAddsOnce(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	form := url.Values{"servings": {"4"}, "list": {"new"}, "name": {"Samstag"}, "take": {"0"}, "fill": {"fillaaaaaaaa"}}
	first := postShop(t, ts, admin, id, form)
	second := postShop(t, ts, admin, id, form)
	if lists := ts.shoppingLists(t, "admin"); len(lists) != 1 || len(lists[0].Items) != 1 || lists[0].Items[0].Text != "800 g Kartoffeln" {
		t.Fatalf("%+v", lists)
	}
	if first.Header.Get("Location") != second.Header.Get("Location") {
		t.Error("second post does not lead to the same list")
	}
}

func (ts *testServer) shoppingLists(t *testing.T, sub string) []todos.List {
	t.Helper()
	var out []todos.List
	for _, l := range ts.deps.Todos.Visible(sub, nil) {
		if l.Kind == todos.KindShopping {
			out = append(out, l)
		}
	}
	return out
}

func TestShopNoNameToType(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.makeRecipe(t, admin, "haus", "Curry", curryBody, 4)
	_, body := get(t, admin, ts.URL+"/recipes/"+id+"/shop")
	if strings.Contains(body, `type="text" id="name"`) || strings.Contains(body, `<input id="name"`) || strings.Contains(body, "Name der neuen Liste") {
		t.Fatal("still asks for a name")
	}
	postShop(t, ts, admin, id, url.Values{"list": {"new"}, "take": {"0"}})
	if l := ts.shoppingLists(t, "admin"); len(l) != 1 || l[0].Name != "Einkauf: Curry" {
		t.Fatalf("%+v", l)
	}
}
