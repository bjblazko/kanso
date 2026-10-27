// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

func TestHistoryOverHTTP(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	if err := ts.deps.Wiki.Save(store.Author{Name: "Anna", Email: "a@x"}, id, "Heizung", "eins\n"); err != nil {
		t.Fatal(err)
	}
	if err := ts.deps.Wiki.Save(store.Author{Name: "Ben", Email: "b@x"}, id, "Heizung", "zwei\n"); err != nil {
		t.Fatal(err)
	}
	revs, _ := ts.deps.Wiki.History(id, 10)

	_, body := get(t, anna, ts.URL+"/p/"+id+"/history")
	if !strings.Contains(body, "Anna") || !strings.Contains(body, "Ben") || !strings.Contains(body, revs[1].Hash[:7]) {
		t.Errorf("history page: %s", body)
	}
	resp, body := get(t, anna, ts.URL+"/p/"+id+"/history/"+revs[1].Hash)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `class="diff-del"`) || !strings.Contains(body, "zwei") {
		t.Errorf("revision page: %d", resp.StatusCode)
	}
	r := ts.postForm(t, anna, "/p/"+id+"/history/"+revs[1].Hash, "/p/"+id+"/history/"+revs[1].Hash+"/restore", url.Values{})
	if r.StatusCode != http.StatusSeeOther {
		t.Fatalf("restore: %d", r.StatusCode)
	}
	if p, _ := ts.deps.Wiki.ByID(id); p.Body != "eins\n" {
		t.Errorf("after restore: %q", p.Body)
	}
}

func TestHistoryRejectsBadHashes(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	if r, _ := get(t, admin, ts.URL+"/p/"+id+"/history/HEAD;rm"); r.StatusCode != http.StatusBadRequest {
		t.Errorf("bad hash: %d", r.StatusCode)
	}
	if r, _ := get(t, admin, ts.URL+"/p/"+id+"/history/0000000"); r.StatusCode != http.StatusNotFound {
		t.Errorf("unknown hash: %d", r.StatusCode)
	}
}

func TestRestoreNeedsWriteAndFreeLock(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Heizung")
	revs, _ := ts.deps.Wiki.History(id, 10)
	hash := revs[0].Hash
	oma := ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if r := ts.postForm(t, oma, "/p/"+id, "/p/"+id+"/history/"+hash+"/restore", url.Values{}); r.StatusCode != http.StatusForbidden {
		t.Errorf("viewer restore: %d", r.StatusCode)
	}
	openEditor(t, ts, admin, id)
	if r := ts.postForm(t, anna, "/p/"+id, "/p/"+id+"/history/"+hash+"/restore", url.Values{}); r.StatusCode != http.StatusConflict {
		t.Errorf("restore while locked: %d", r.StatusCode)
	}
}
