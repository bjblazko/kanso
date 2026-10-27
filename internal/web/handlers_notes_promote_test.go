// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/notes"
)

func TestPromoteNoteToWikiPage(t *testing.T) {
	ts, _, anna := setupHaus(t)
	n := annaNote(t, ts, "Zähler ablesen\n")
	resp, form := get(t, anna, ts.URL+"/notes/"+n.Meta.ID+"/promote")
	if resp.StatusCode != http.StatusOK || !strings.Contains(form, `value="haus/"`) || !strings.Contains(form, `value="Zähler ablesen"`) {
		t.Fatalf("promote form: %d", resp.StatusCode)
	}
	resp = ts.postForm(t, anna, "/notes/"+n.Meta.ID+"/promote", "/notes/"+n.Meta.ID+"/promote",
		url.Values{"where": {"haus/"}, "title": {"Zähler"}})
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasPrefix(loc, "/p/") {
		t.Fatalf("promote: %d %q", resp.StatusCode, loc)
	}
	if img, _ := get(t, anna, ts.URL+loc+"/attachments/a.png"); img.StatusCode != http.StatusOK {
		t.Errorf("photo on the page: %d", img.StatusCode)
	}
	if _, page := get(t, anna, ts.URL+loc); !strings.Contains(page, `src="`+loc+`/attachments/a.png"`) {
		t.Error("page does not show the photo")
	}
	if _, err := ts.deps.Notes.Get("anna", n.Meta.ID); err == nil {
		t.Error("note still there")
	}
	if _, body := get(t, anna, ts.URL+"/search?q=ablesen"); strings.Contains(body, "/notes/"+n.Meta.ID) || !strings.Contains(body, loc) {
		t.Error("search still points at the note")
	}
}

func TestPromoteNeedsEditor(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	ben := ts.client(t, "ben", "Ben", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"ben"}, "role": {"viewer"}})
	n, _, err := ts.deps.Notes.Create("ben", "", "Idee", []notes.Upload{{Name: "a.png", R: bytes.NewReader(pngBytes)}}, maxAttachment)
	if err != nil {
		t.Fatal(err)
	}
	if _, form := get(t, ben, ts.URL+"/notes/"+n.Meta.ID+"/promote"); strings.Contains(form, `value="haus/"`) {
		t.Error("viewer is offered a space to write into")
	}
	for _, where := range []string{"haus/", "gibtsnicht/", "haus"} {
		resp := ts.postForm(t, ben, "/notes/"+n.Meta.ID+"/promote", "/notes/"+n.Meta.ID+"/promote",
			url.Values{"where": {where}, "title": {"Idee"}})
		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("where %q: %d", where, resp.StatusCode)
		}
	}
	if _, err := ts.deps.Notes.Get("ben", n.Meta.ID); err != nil {
		t.Errorf("note lost: %v", err)
	}
}
