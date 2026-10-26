// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestAttachmentNamesStayInside(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Seite")
	safe := regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9]{1,8})?$`)
	for _, name := range []string{"../../x.png", `C:\evil.png`, "a\x00.png", ".htaccess", "Grundriss EG.PNG", "", "...."} {
		got, err := w.AddAttachment(anna, p.Meta.ID, name, strings.NewReader("data"), 1<<20)
		if err != nil {
			t.Fatalf("%q: %v", name, err)
		}
		if !safe.MatchString(got) {
			t.Errorf("%q → unsafe name %q", name, got)
		}
		dir := filepath.Join(w.dataDir, "spaces", "haus", "pages", p.Dir, "attachments")
		if _, err := os.Stat(filepath.Join(dir, got)); err != nil {
			t.Errorf("%q: not stored inside attachments: %v", name, err)
		}
	}
	if got := w.ListAttachments(p.Meta.ID); len(got) != 7 {
		t.Errorf("list = %v", got)
	}
}

func TestAttachmentKeepsReadableName(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Seite")
	got, _ := w.AddAttachment(anna, p.Meta.ID, "Grundriss EG.PNG", strings.NewReader("x"), 1<<20)
	again, _ := w.AddAttachment(anna, p.Meta.ID, "Grundriss EG.PNG", strings.NewReader("y"), 1<<20)
	if got != "grundriss-eg.png" || again != "grundriss-eg-2.png" {
		t.Errorf("names %q %q", got, again)
	}
}

func TestAttachmentTooLarge(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Seite")
	_, err := w.AddAttachment(anna, p.Meta.ID, "big.bin", strings.NewReader(strings.Repeat("x", 11)), 10)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if len(w.ListAttachments(p.Meta.ID)) != 0 {
		t.Error("partial file left behind")
	}
}

func TestOpenAndDeleteAttachment(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Seite")
	name, _ := w.AddAttachment(anna, p.Meta.ID, "a.txt", strings.NewReader("hallo"), 1<<20)
	f, err := w.OpenAttachment(p.Meta.ID, name)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if _, err := w.OpenAttachment(p.Meta.ID, "../index.md"); err == nil {
		t.Error("opened file outside attachments")
	}
	if err := w.DeleteAttachment(anna, p.Meta.ID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := w.OpenAttachment(p.Meta.ID, name); err == nil {
		t.Error("deleted attachment still opens")
	}
}
