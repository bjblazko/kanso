// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/access"
)

func TestPageShareOnlyManagers(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	if code := status(t, w.anna, w.ts.URL+"/p/"+w.list+"/share"); code != http.StatusForbidden {
		t.Errorf("space editor: %d", code)
	}
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.list+"/share"); code != http.StatusForbidden {
		t.Errorf("page viewer: %d", code)
	}
	if code := status(t, w.admin, w.ts.URL+"/p/"+w.list+"/share"); code != http.StatusOK {
		t.Errorf("admin: %d", code)
	}
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.secret+"/share"); code != http.StatusNotFound {
		t.Errorf("outside the share: %d", code)
	}
	m := setupWishlist(t, access.Owner)
	if code := status(t, m.christina, m.ts.URL+"/p/"+m.books+"/share"); code != http.StatusOK {
		t.Errorf("page manager: %d", code)
	}
}

func TestPageShareAddRemove(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	page := "/p/" + w.list + "/share"
	resp := w.ts.postForm(t, w.admin, page, page, url.Values{"action": {"add"}, "kind": {"group"}, "who": {"familie"}, "role": {"editor"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("add: %d", resp.StatusCode)
	}
	p, _ := w.ts.deps.Wiki.ByID(w.list)
	if len(p.Meta.Members) != 2 || p.Meta.Members[1].Group != "familie" || p.Meta.Members[1].Role != access.Editor {
		t.Fatalf("members %+v", p.Meta.Members)
	}
	w.ts.postForm(t, w.admin, page, page, url.Values{"action": {"remove"}, "index": {"0"}})
	if p, _ = w.ts.deps.Wiki.ByID(w.list); len(p.Meta.Members) != 1 || p.Meta.Members[0].Group != "familie" {
		t.Fatalf("after remove %+v", p.Meta.Members)
	}
	if code := status(t, w.christina, w.ts.URL+"/p/"+w.list); code != http.StatusNotFound {
		t.Fatalf("removed person still reads: %d", code)
	}
}

func TestPageShareHintAndInherited(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	if _, body := get(t, w.admin, w.ts.URL+"/p/"+w.list+"/share"); !strings.Contains(body, "Gilt auch für 1 Unterseite.") {
		t.Fatalf("hint missing:\n%s", body)
	}
	_, body := get(t, w.admin, w.ts.URL+"/p/"+w.books+"/share")
	if !strings.Contains(body, "Christina") || !strings.Contains(body, `href="/p/`+w.list+`/share"`) {
		t.Fatalf("inherited share missing:\n%s", body)
	}
}

func TestPageAudienceNamesSharedPeople(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	for _, id := range []string{w.list, w.books} {
		if _, body := get(t, w.anna, w.ts.URL+"/p/"+id); !strings.Contains(body, "Sichtbar für dich, Christina und Admins") {
			t.Errorf("%s: audience lacks Christina", id)
		}
	}
	if _, body := get(t, w.anna, w.ts.URL+"/p/"+w.secret); strings.Contains(body, "Christina") {
		t.Error("sibling names Christina")
	}
}

func TestPageAudienceLinksToPageShare(t *testing.T) {
	w := setupWishlist(t, access.Viewer)
	if _, body := get(t, w.admin, w.ts.URL+"/p/"+w.books); !strings.Contains(body, `href="/p/`+w.books+`/share">Freigabe ändern`) {
		t.Fatal("page link missing")
	}
	if _, body := get(t, w.christina, w.ts.URL+"/p/"+w.list); strings.Contains(body, "Freigabe ändern") {
		t.Fatal("a reader gets the link")
	}
}
