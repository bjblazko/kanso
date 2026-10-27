// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentName(t *testing.T) {
	cases := map[string]string{
		"Grundriss EG.PNG": "grundriss-eg.png",
		"../../x.png":      "x.png",
		`C:\evil.png`:      "evil.png",
		"....":             "datei",
		"":                 "datei",
		"Seite.pdf":        "seite.pdf",
		"Größe.JPEG":       "groesse.jpeg",
	}
	for in, want := range cases {
		if got := AttachmentName(in); got != want {
			t.Errorf("AttachmentName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestUniqueFile(t *testing.T) {
	dir := t.TempDir()
	if got := UniqueFile(dir, "a.png"); got != "a.png" {
		t.Errorf("free = %q", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.png"), nil, 0o640); err != nil {
		t.Fatal(err)
	}
	if got := UniqueFile(dir, "a.png"); got != "a-2.png" {
		t.Errorf("taken = %q", got)
	}
}
