// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func upload(t *testing.T, ts *testServer, c *http.Client, id, filename, content string) (*http.Response, map[string]string) {
	t.Helper()
	_, page := get(t, c, ts.URL+"/p/"+id)
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", filename)
	if _, err := fw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	req, _ := http.NewRequest("POST", ts.URL+"/p/"+id+"/attachments", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Origin", ts.URL)
	req.Header.Set("X-CSRF-Token", between(page, `<meta name="csrf" content="`, `"`))
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	out := map[string]string{}
	_ = json.NewDecoder(resp.Body).Decode(&out) // refusals answer with an HTML page
	return resp, out
}

func TestUploadAndServeAttachments(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Bilder")
	resp, out := upload(t, ts, anna, id, "Grundriss EG.png", "\x89PNG fake")
	if resp.StatusCode != http.StatusOK || out["name"] != "grundriss-eg.png" || out["markdown"] != "![Grundriss EG](attachments/grundriss-eg.png)" {
		t.Fatalf("upload: %d %v", resp.StatusCode, out)
	}
	r, _ := get(t, anna, ts.URL+"/p/"+id+"/attachments/grundriss-eg.png")
	if r.StatusCode != http.StatusOK || r.Header.Get("Content-Type") != "image/png" || !strings.HasPrefix(r.Header.Get("Content-Disposition"), "inline") {
		t.Errorf("image: %d %q %q", r.StatusCode, r.Header.Get("Content-Type"), r.Header.Get("Content-Disposition"))
	}
	_, out = upload(t, ts, anna, id, "bild.svg", "<svg onload=alert(1)>")
	r, _ = get(t, anna, ts.URL+"/p/"+id+"/attachments/"+out["name"])
	if !strings.HasPrefix(r.Header.Get("Content-Disposition"), "attachment") || r.Header.Get("Content-Type") != "application/octet-stream" {
		t.Errorf("svg served inline: %q %q", r.Header.Get("Content-Disposition"), r.Header.Get("Content-Type"))
	}
	if out["markdown"] != "[bild.svg](attachments/bild.svg)" {
		t.Errorf("svg markdown = %q", out["markdown"])
	}
}

func TestViewerCannotUpload(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Bilder")
	oma := ts.client(t, "oma", "Oma", false)
	ts.postForm(t, admin, "/s/haus/settings", "/s/haus/settings",
		url.Values{"action": {"add"}, "kind": {"user"}, "who": {"oma"}, "role": {"viewer"}})
	if resp, _ := upload(t, ts, oma, id, "a.png", "x"); resp.StatusCode != http.StatusForbidden {
		t.Errorf("viewer upload: %d", resp.StatusCode)
	}
	outsider := ts.client(t, "fremd", "Fremd", false)
	if r, _ := get(t, outsider, ts.URL+"/p/"+id+"/attachments/a.png"); r.StatusCode != http.StatusNotFound {
		t.Errorf("outsider download: %d", r.StatusCode)
	}
}

func TestPageListsAttachments(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	id := ts.createPage(t, admin, "haus", "", "Rechnungen")
	upload(t, ts, admin, id, "Rechnung 2026.pdf", "%PDF")
	_, body := get(t, admin, ts.URL+"/p/"+id)
	if !strings.Contains(body, `href="/p/`+id+`/attachments/rechnung-2026.pdf"`) {
		t.Error("attachment not listed on page")
	}
}
