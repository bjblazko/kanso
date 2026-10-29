// SPDX-License-Identifier: Apache-2.0

package users

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Black)
	var b bytes.Buffer
	if err := png.Encode(&b, img); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

// serve answers every request with body.
func serve(t *testing.T, body []byte) *httptest.Server {
	t.Helper()
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) }))
	t.Cleanup(ts.Close)
	return ts
}

func TestAvatarIsFetchedAndKeptOutOfGit(t *testing.T) {
	s, repo := newStore(t)
	img := pngBytes(t)
	ts := serve(t, img)
	if err := s.SyncAvatar(context.Background(), ts.Client(), "anna", ts.URL+"/anna.png"); err != nil {
		t.Fatal(err)
	}
	got, typ, ok := s.Avatar("anna")
	if !ok || typ != "image/png" || !bytes.Equal(got, img) || s.AvatarVersion("anna") == "" {
		t.Fatalf("avatar %v %q", ok, typ)
	}
	ignore, _ := os.ReadFile(filepath.Join(repo.Dir(), ".gitignore"))
	if !strings.Contains(string(ignore), "/avatars/\n") {
		t.Fatalf(".gitignore: %q", ignore)
	}
}

func TestAvatarRefusesWhatIsNoImage(t *testing.T) {
	s, _ := newStore(t)
	for name, body := range map[string][]byte{
		"html":  []byte("<html><script>x</script></html>"),
		"large": append(pngBytes(t), make([]byte, maxAvatar)...),
	} {
		ts := serve(t, body)
		if err := s.SyncAvatar(context.Background(), ts.Client(), "anna", ts.URL); err == nil {
			t.Errorf("%s accepted", name)
		}
		if _, _, ok := s.Avatar("anna"); ok {
			t.Errorf("%s stored", name)
		}
	}
}

func TestAvatarOnlyOverHTTPS(t *testing.T) {
	s, _ := newStore(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(pngBytes(t)) }))
	defer ts.Close()
	if err := s.SyncAvatar(context.Background(), ts.Client(), "anna", ts.URL); err == nil {
		t.Error("plain http accepted")
	}
}

// No picture any more: the stored one goes too.
func TestAvatarRemovedWithoutPicture(t *testing.T) {
	s, _ := newStore(t)
	ts := serve(t, pngBytes(t))
	if err := s.SyncAvatar(context.Background(), ts.Client(), "anna", ts.URL); err != nil {
		t.Fatal(err)
	}
	if err := s.SyncAvatar(context.Background(), ts.Client(), "anna", ""); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := s.Avatar("anna"); ok || s.AvatarVersion("anna") != "" {
		t.Error("avatar kept")
	}
}
