// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

func TestSharedSectionInNav(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.christina, w.ts.URL+"/p/"+w.books)
	nav := between(body, `<nav class="sidenav"`, `</nav>`)
	if !strings.Contains(nav, "Für dich freigegeben") || !strings.Contains(nav, `href="/p/`+w.list+`"`) ||
		!strings.Contains(nav, `href="/p/`+w.books+`" aria-current="page"`) {
		t.Fatalf("nav:\n%s", nav)
	}
	if strings.Contains(nav, "Geheim") || strings.Contains(nav, `href="/s/haus"`) {
		t.Fatalf("nav shows more than the share:\n%s", nav)
	}
}

func TestSharedOnHome(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.christina, w.ts.URL+"/")
	main := between(body, `<main`, `</main>`)
	if !strings.Contains(main, "Für dich freigegeben") || !strings.Contains(main, `href="/p/`+w.list+`"`) || !strings.Contains(main, "Haus") {
		t.Fatalf("home:\n%s", main)
	}
	if strings.Contains(main, "Noch keine") {
		t.Fatal("home says it is empty")
	}
}

func TestSharedSectionHiddenWhenEmpty(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	// anna reads the whole space: the page is not "shared with her"
	if _, body := get(t, w.anna, w.ts.URL+"/"); strings.Contains(body, "Für dich freigegeben") {
		t.Fatal("member of the space gets the section")
	}
}

func TestBreadcrumbsStartAtSharedRoot(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	_, body := get(t, w.christina, w.ts.URL+"/p/"+w.books)
	crumbs := between(body, `<nav class="breadcrumb"`, `</nav>`)
	if strings.Contains(crumbs, `/s/haus`) || !strings.Contains(crumbs, `href="/p/`+w.list+`"`) {
		t.Fatalf("crumbs:\n%s", crumbs)
	}
}

func TestRemovedShareDisappears(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	page := "/p/" + w.list + "/share"
	w.ts.postForm(t, w.admin, page, page, url.Values{"action": {"remove"}, "index": {"0"}})
	if _, body := get(t, w.christina, w.ts.URL+"/"); strings.Contains(body, "Für dich freigegeben") {
		t.Fatal("section still there")
	}
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.list); code != http.StatusNotFound {
		t.Fatalf("page still opens: %d", code)
	}
}
