// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// setupKeller adds a second space "Keller" that only admins see.
func setupKeller(t *testing.T) (*testServer, *http.Client, *http.Client) {
	t.Helper()
	ts, admin, anna := setupHaus(t)
	if resp := ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Keller"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("create Keller: %d", resp.StatusCode)
	}
	return ts, admin, anna
}

func movePost(t *testing.T, ts *testServer, c *http.Client, id string, f url.Values) *http.Response {
	t.Helper()
	return ts.postForm(t, c, "/p/"+id, "/p/"+id+"/move", f)
}

func TestMoveTargetsListOtherSpaces(t *testing.T) {
	ts, admin, anna := setupKeller(t)
	id := ts.createPage(t, anna, "haus", "", "Heizung")
	if _, body := get(t, admin, ts.URL+"/p/"+id); !strings.Contains(body, `value="keller/"`) {
		t.Fatalf("admin misses Keller as a target:\n%s", body)
	}
	if _, body := get(t, anna, ts.URL+"/p/"+id); strings.Contains(body, `value="keller/`) {
		t.Fatal("anna gets a space she cannot write")
	}
}

func TestMovePageToOtherSpaceAsksWhenAudienceChanges(t *testing.T) {
	ts, admin, anna := setupKeller(t)
	id := ts.createPage(t, anna, "haus", "", "Heizung")
	child := ts.createPage(t, anna, "haus", id, "Wartung")
	resp := movePost(t, ts, admin, id, url.Values{"parent": {"keller/"}, "order": {"0"}})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected the confirmation page, got %d", resp.StatusCode)
	}
	if p, _ := ts.deps.Wiki.ByID(id); p.Space != "haus" {
		t.Fatal("moved without confirmation")
	}
	_, page := ts.postFormBody(t, admin, "/p/"+id, "/p/"+id+"/move", url.Values{"parent": {"keller/"}, "order": {"0"}})
	if !strings.Contains(page, "Bisher") || !strings.Contains(page, "Danach") || !strings.Contains(page, "Sichtbar nur für Admins") {
		t.Fatalf("confirmation lacks the audiences:\n%s", page)
	}
	resp = movePost(t, ts, admin, id, url.Values{"parent": {"keller/"}, "order": {"0"}, "confirm": {"1"}})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("confirmed move: %d", resp.StatusCode)
	}
	for _, pid := range []string{id, child} {
		if p, _ := ts.deps.Wiki.ByID(pid); p.Space != "keller" {
			t.Fatalf("%s still in %s", pid, p.Space)
		}
	}
	if code := status(t, anna, ts.URL+"/p/"+id); code != http.StatusNotFound {
		t.Fatalf("anna still reads it: %d", code)
	}
}

func TestMoveAcrossNeedsWriteAtTarget(t *testing.T) {
	ts, _, anna := setupKeller(t)
	id := ts.createPage(t, anna, "haus", "", "Heizung")
	if resp := movePost(t, ts, anna, id, url.Values{"parent": {"keller/"}, "confirm": {"1"}}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestMoveAcrossRefusedWhileEdited(t *testing.T) {
	ts, admin, anna := setupKeller(t)
	id := ts.createPage(t, anna, "haus", "", "Heizung")
	child := ts.createPage(t, anna, "haus", id, "Wartung")
	if _, err := ts.deps.Locks.Acquire(child, "anna", "Anna"); err != nil {
		t.Fatal(err)
	}
	resp, body := ts.postFormBody(t, admin, "/p/"+id, "/p/"+id+"/move", url.Values{"parent": {"keller/"}, "confirm": {"1"}})
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, "Anna") {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if p, _ := ts.deps.Wiki.ByID(id); p.Space != "haus" {
		t.Fatal("moved while being edited")
	}
}

func TestMoveWithinSpaceNeedsNoConfirmation(t *testing.T) {
	ts, _, anna := setupKeller(t)
	a := ts.createPage(t, anna, "haus", "", "Garten")
	b := ts.createPage(t, anna, "haus", "", "Rasen")
	if resp := movePost(t, ts, anna, b, url.Values{"parent": {"haus/" + a}, "order": {"0"}}); resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if c := ts.deps.Wiki.Chain(b); len(c) != 2 || c[1].ID != a {
		t.Fatalf("chain %+v", c)
	}
}
