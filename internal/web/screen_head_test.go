// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/tasks"
)

// headOf is the fixed head of a screen: toolbar, path and title.
func headOf(body string) string { return between(body, `<header class="screen-head">`, `</header>`) }

// expectHead checks that the head holds every want and the title, and that the
// screen has no action row below its content.
func expectHead(t *testing.T, body, title string, want ...string) {
	t.Helper()
	head := headOf(body)
	if !strings.Contains(head, `role="toolbar"`) || !strings.Contains(head, "<h1") || !strings.Contains(head, title) {
		t.Fatalf("head lacks toolbar or title %q:\n%s", title, head)
	}
	for _, w := range want {
		if !strings.Contains(head, w) {
			t.Errorf("head lacks %s", w)
		}
	}
	if strings.Contains(body, `<div class="actions">`) {
		t.Error("actions still below the content")
	}
}

func TestScreenHeadWiki(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.admin, w.ts.URL+"/s/haus")
	expectHead(t, body, "Haus", `href="/s/haus/new"`, `href="/s/haus/settings"`, "Sichtbar für")
	_, body = get(t, w.admin, w.ts.URL+"/p/"+w.list)
	expectHead(t, body, "Wunschliste", `href="/p/`+w.list+`/edit"`, `class="breadcrumb"`)
	_, body = get(t, w.admin, w.ts.URL+"/")
	expectHead(t, body, "Wiki-Bereiche", `href="/admin/spaces/new"`)
}

func TestScreenHeadNotes(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n, _, _ := ts.deps.Notes.Create("anna", "", "Hof fegen\n", nil, 1<<20)
	_, body := get(t, anna, ts.URL+"/notes")
	expectHead(t, body, "Notizen", `href="/notes/new"`, "Nur für dich sichtbar")
	_, body = get(t, anna, ts.URL+"/notes/"+n.Meta.ID)
	expectHead(t, body, "Hof fegen", `href="/notes/`+n.Meta.ID+`/edit"`, `/promote"`, `/task"`, `/delete"`)
}

func TestScreenHeadTasks(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fam := familyBoardID(t, ts)
	tk := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Hof fegen"})
	_, body := get(t, anna, ts.URL+"/tasks/b/"+fam)
	expectHead(t, body, "Familie", `href="/tasks/new?board=`+fam+`"`, "?view=matrix", "Sichtbar für")
	_, body = get(t, anna, ts.URL+taskURL(tk))
	expectHead(t, body, "Hof fegen", `href="`+taskURL(tk)+`/edit"`, `/claim"`, `/delete"`)
}
