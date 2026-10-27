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
)

// postWith sends form to path with csrf; asJSON asks for a JSON answer.
func postWith(t *testing.T, ts *testServer, c *http.Client, csrf, path string, form url.Values, asJSON bool) (*http.Response, string) {
	t.Helper()
	req, _ := http.NewRequest("POST", ts.URL+path, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", csrf)
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

// openNoteEditor returns the CSRF token and version of the note editor.
func openNoteEditor(t *testing.T, ts *testServer, c *http.Client, id string) (string, string) {
	t.Helper()
	resp, body := get(t, c, ts.URL+"/notes/"+id+"/edit")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("edit: %d", resp.StatusCode)
	}
	return between(body, `<meta name="csrf" content="`, `"`), between(body, `name="version" value="`, `"`)
}

func TestEditNoteSaves(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n := annaNote(t, ts, "alt")
	csrf, version := openNoteEditor(t, ts, anna, n.Meta.ID)
	resp, _ := postWith(t, ts, anna, csrf, "/notes/"+n.Meta.ID, url.Values{"body": {"neu\r\n"}, "version": {version}}, false)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/notes/"+n.Meta.ID {
		t.Fatalf("save: %d %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	if got, _ := ts.deps.Notes.Get("anna", n.Meta.ID); got.Body != "neu\n" {
		t.Errorf("body = %q", got.Body)
	}
}

func TestEditNoteConflict(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n := annaNote(t, ts, "alt")
	csrf, version := openNoteEditor(t, ts, anna, n.Meta.ID)
	form := url.Values{"body": {"Handy"}, "version": {version}}
	if resp, body := postWith(t, ts, anna, csrf, "/notes/"+n.Meta.ID, form, true); resp.StatusCode != http.StatusOK {
		t.Fatalf("first save: %d %s", resp.StatusCode, body)
	}
	form.Set("body", "Laptop")
	resp, body := postWith(t, ts, anna, csrf, "/notes/"+n.Meta.ID, form, true)
	if resp.StatusCode != http.StatusConflict || !strings.Contains(body, `"error":"conflict"`) {
		t.Errorf("stale save: %d %s", resp.StatusCode, body)
	}
	if got, _ := ts.deps.Notes.Get("anna", n.Meta.ID); got.Body != "Handy" {
		t.Errorf("body = %q", got.Body)
	}
}

func TestNoteEditorUpload(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n := annaNote(t, ts, "x")
	csrf, _ := openNoteEditor(t, ts, anna, n.Meta.ID)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "Bild.png")
	if _, err := fw.Write(pngBytes); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", ts.URL+"/notes/"+n.Meta.ID+"/attachments", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", csrf)
	resp, err := anna.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(b), `![Bild](attachments/bild.png)`) {
		t.Errorf("upload: %d %s", resp.StatusCode, b)
	}
}

func TestPreviewForNote(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	n := annaNote(t, ts, "x")
	csrf, _ := openNoteEditor(t, ts, anna, n.Meta.ID)
	form := url.Values{"note": {n.Meta.ID}, "body": {"![](attachments/a.png)"}}
	resp, body := postWith(t, ts, anna, csrf, "/api/preview", form, false)
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, `src="/notes/`+n.Meta.ID+`/attachments/a.png"`) {
		t.Errorf("preview: %d %s", resp.StatusCode, body)
	}
	ben := ts.client(t, "ben", "Ben", false)
	_, page := get(t, ben, ts.URL+"/notes")
	benCSRF := between(page, `<meta name="csrf" content="`, `"`)
	if resp, _ := postWith(t, ts, ben, benCSRF, "/api/preview", form, false); resp.StatusCode != http.StatusNotFound {
		t.Errorf("ben previews anna's note: %d", resp.StatusCode)
	}
	for _, path := range []string{"/notes/" + n.Meta.ID, "/notes/" + n.Meta.ID + "/attachments"} {
		if resp, _ := postWith(t, ts, ben, benCSRF, path, url.Values{"body": {"x"}}, true); resp.StatusCode != http.StatusNotFound {
			t.Errorf("ben POST %s: %d", path, resp.StatusCode)
		}
	}
	if resp, _ := get(t, ben, ts.URL+"/notes/"+n.Meta.ID+"/edit"); resp.StatusCode != http.StatusNotFound {
		t.Errorf("ben opens the editor: %d", resp.StatusCode)
	}
}
