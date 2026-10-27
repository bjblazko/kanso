// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/notes"
)

var pngBytes = []byte("\x89PNG\r\n\x1a\nnot really an image")

// postNote sends the capture form as multipart, like the browser does.
func postNote(t *testing.T, ts *testServer, c *http.Client, fields map[string]string, files map[string][]byte, asJSON bool) (*http.Response, string) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/notes/new")
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for k, v := range fields {
		if err := mw.WriteField(k, v); err != nil {
			t.Fatal(err)
		}
	}
	for name, content := range files {
		fw, err := mw.CreateFormFile("files", name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := fw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", ts.URL+"/notes", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", between(page, `<meta name="csrf" content="`, `"`))
	if asJSON {
		req.Header.Set("Accept", "application/json")
	}
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func annaNote(t *testing.T, ts *testServer, body string) notes.Note {
	t.Helper()
	n, _, err := ts.deps.Notes.Create("anna", "", body, []notes.Upload{{Name: "a.png", R: bytes.NewReader(pngBytes)}}, maxAttachment)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCaptureWithoutJS(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, _ := postNote(t, ts, anna, map[string]string{"body": "Einkaufen\nMilch"}, map[string][]byte{"a.png": pngBytes}, false)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/notes/new?saved=1" {
		t.Fatalf("capture: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	list := ts.deps.Notes.List("anna")
	if len(list) != 1 {
		t.Fatalf("notes = %d", len(list))
	}
	id := list[0].Meta.ID
	if _, body := get(t, anna, ts.URL+"/notes"); !strings.Contains(body, `href="/notes/`+id+`"`) || !strings.Contains(body, "Einkaufen") {
		t.Error("note not listed")
	}
	img, _ := get(t, anna, ts.URL+"/notes/"+id+"/attachments/a.png")
	if img.StatusCode != http.StatusOK || img.Header.Get("Content-Type") != "image/png" {
		t.Errorf("photo: %d %q", img.StatusCode, img.Header.Get("Content-Type"))
	}
	if _, body := get(t, anna, ts.URL+"/notes/"+id); !strings.Contains(body, `src="/notes/`+id+`/attachments/a.png"`) {
		t.Error("photo not shown in the note")
	}
}

func TestCaptureJSONIsIdempotent(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	fields := map[string]string{"id": "abcdefghijkl", "body": "Zettel"}
	resp, body := postNote(t, ts, anna, fields, nil, true)
	if resp.StatusCode != http.StatusCreated || !strings.Contains(body, `"status":"created"`) {
		t.Fatalf("first: %d %s", resp.StatusCode, body)
	}
	resp, body = postNote(t, ts, anna, fields, nil, true)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `"status":"exists"`) {
		t.Errorf("retry: %d %s", resp.StatusCode, body)
	}
	if n := len(ts.deps.Notes.List("anna")); n != 1 {
		t.Errorf("notes = %d", n)
	}
	resp, _ = postNote(t, ts, anna, map[string]string{"id": "../evil", "body": "x"}, nil, true)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("bad id: %d", resp.StatusCode)
	}
}

func TestEmptyNoteRefused(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	resp, body := postNote(t, ts, anna, map[string]string{"body": "  \n "}, nil, false)
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "Schreib etwas") {
		t.Errorf("empty note: %d", resp.StatusCode)
	}
	if n := len(ts.deps.Notes.List("anna")); n != 0 {
		t.Errorf("notes = %d", n)
	}
}

func TestNotesArePrivate(t *testing.T) {
	ts := newTestServer(t)
	n := annaNote(t, ts, "geheim")
	ben := ts.client(t, "ben", "Ben", false)
	admin := ts.client(t, "chef", "Chef", true)
	for name, c := range map[string]*http.Client{"ben": ben, "admin": admin} {
		for _, path := range []string{"/notes/" + n.Meta.ID, "/notes/" + n.Meta.ID + "/attachments/a.png", "/notes/" + n.Meta.ID + "/delete"} {
			if resp, _ := get(t, c, ts.URL+path); resp.StatusCode != http.StatusNotFound {
				t.Errorf("%s GET %s: %d", name, path, resp.StatusCode)
			}
		}
		if resp := ts.postForm(t, c, "/notes", "/notes/"+n.Meta.ID+"/delete", url.Values{}); resp.StatusCode != http.StatusNotFound {
			t.Errorf("%s delete: %d", name, resp.StatusCode)
		}
		if _, body := get(t, c, ts.URL+"/notes"); strings.Contains(body, n.Meta.ID) {
			t.Errorf("%s lists anna's note", name)
		}
	}
	if _, err := ts.deps.Notes.Get("anna", n.Meta.ID); err != nil {
		t.Errorf("note gone: %v", err)
	}
}

func TestDeleteNote(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n := annaNote(t, ts, "weg damit")
	if resp, body := get(t, anna, ts.URL+"/notes/"+n.Meta.ID+"/delete"); resp.StatusCode != http.StatusOK || !strings.Contains(body, "weg damit") {
		t.Fatalf("confirm page: %d", resp.StatusCode)
	}
	resp := ts.postForm(t, anna, "/notes/"+n.Meta.ID+"/delete", "/notes/"+n.Meta.ID+"/delete", url.Values{})
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/notes" {
		t.Fatalf("delete: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if _, err := ts.deps.Notes.Get("anna", n.Meta.ID); err == nil {
		t.Error("note still there")
	}
}

func TestNotesNavEntryAndEmptyState(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	if _, body := get(t, anna, ts.URL+"/"); !strings.Contains(body, `href="/notes"`) {
		t.Error("no notes entry in the navigation")
	}
	if _, body := get(t, anna, ts.URL+"/notes"); !strings.Contains(body, "Noch keine Notizen.") || !strings.Contains(body, `href="/notes/new"`) {
		t.Error("empty state missing")
	}
}
