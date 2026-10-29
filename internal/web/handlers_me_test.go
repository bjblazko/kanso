// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"image"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMenuShowsPlaceholderWithoutPicture(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/")
	menu := between(body, `<details class="usermenu">`, `</summary>`)
	if !strings.Contains(menu, `class="avatar avatar-none"`) || !strings.Contains(menu, "Anna") || strings.Contains(menu, "<img") {
		t.Errorf("menu: %s", menu)
	}
}

func TestPictureIsServedAndShownRound(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	var img bytes.Buffer
	_ = png.Encode(&img, image.NewGray(image.Rect(0, 0, 1, 1)))
	dir := filepath.Join(ts.deps.Cfg.DataDir, "avatars")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "anna"), img.Bytes(), 0o640); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/")
	if !strings.Contains(body, `<img class="avatar" src="/avatar/anna?v=`) {
		t.Error("menu lacks the picture")
	}
	resp, err := anna.Get(ts.URL + "/avatar/anna")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "image/png" || !bytes.Equal(got, img.Bytes()) {
		t.Errorf("avatar: %d %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
	if r, _ := get(t, anna, ts.URL+"/avatar/nobody"); r.StatusCode != http.StatusNotFound {
		t.Errorf("missing avatar: %d", r.StatusCode)
	}
}

// The menu: who you are on top, then language, then the rest, each with an icon.
func TestMenuLanguageWithoutApplyButton(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	_, body := get(t, anna, ts.URL+"/")
	menu := between(body, `<div class="menu">`, `</details>`)
	for _, want := range []string{`name="lang" value="en"`, `aria-pressed="true"`, "Abmelden"} {
		if !strings.Contains(menu, want) {
			t.Errorf("menu lacks %q", want)
		}
	}
	if strings.Contains(menu, "Übernehmen") {
		t.Error("menu still has an apply button")
	}
}
