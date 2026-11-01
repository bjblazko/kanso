// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

func toolbarOf(body string) string { return between(body, `<div class="page-toolbar"`, `</div>`) }

func TestPageToolbarForWriters(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.admin, w.ts.URL+"/p/"+w.list)
	bar := toolbarOf(body)
	for _, want := range []string{
		`href="/p/` + w.list + `/edit"`, `href="/s/haus/new?parent=` + w.list + `"`, `href="/p/` + w.list + `/move"`,
		`href="/p/` + w.list + `/history"`, `href="/p/` + w.list + `/delete"`, `href="/p/` + w.list + `/share"`,
		"Sichtbar für Anna, Christina und Admins",
	} {
		if !strings.Contains(bar, want) {
			t.Errorf("toolbar lacks %s:\n%s", want, bar)
		}
	}
	if strings.Contains(body, `<details class="more">`) {
		t.Error("the old actions are still below the page")
	}
}

func TestPageToolbarForReaders(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.christina, w.ts.URL+"/p/"+w.list)
	bar := toolbarOf(body)
	if !strings.Contains(bar, `href="/p/`+w.list+`/history"`) || !strings.Contains(bar, "Sichtbar für") {
		t.Fatalf("reader toolbar:\n%s", bar)
	}
	for _, not := range []string{"/edit", "/move", "/delete", "/share", "/new?parent"} {
		if strings.Contains(bar, not) {
			t.Errorf("reader gets %s", not)
		}
	}
}

func TestMovePageForm(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	resp, body := get(t, w.admin, w.ts.URL+"/p/"+w.books+"/move")
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `value="haus/"`) || !strings.Contains(body, `action="/p/`+w.books+`/move"`) {
		t.Fatalf("%d\n%s", resp.StatusCode, body)
	}
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.books+"/move"); code != http.StatusForbidden {
		t.Fatalf("reader: %d", code)
	}
}
