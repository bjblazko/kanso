// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestJoinRejectsEscapes(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"../x", "a/../../x", "/etc/passwd", "a\x00b", ".."} {
		if _, err := Join(root, p); !errors.Is(err, ErrOutside) {
			t.Errorf("Join(%q) err = %v, want ErrOutside", p, err)
		}
	}
}

func TestJoinAcceptsInside(t *testing.T) {
	root := t.TempDir()
	got, err := Join(root, "a", "b")
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Join(root, "a", "b") {
		t.Errorf("got %q", got)
	}
}

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Über uns":         "ueber-uns",
		"Größe & Gewicht":  "groesse-gewicht",
		"🙂":                "seite",
		"  Hallo  Welt!! ": "hallo-welt",
		"Gesundheit Timo":  "gesundheit-timo",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
	if got := Slugify(strings.Repeat("a", 300)); len(got) != 80 {
		t.Errorf("long slug length = %d", len(got))
	}
}

func TestUniqueSlug(t *testing.T) {
	dir := t.TempDir()
	if got := UniqueSlug(dir, "x"); got != "x" {
		t.Errorf("free slug = %q", got)
	}
	if err := os.Mkdir(filepath.Join(dir, "x"), 0o750); err != nil {
		t.Fatal(err)
	}
	if got := UniqueSlug(dir, "x"); got != "x-2" {
		t.Errorf("taken slug = %q", got)
	}
}

func TestNewID(t *testing.T) {
	re := regexp.MustCompile(`^[a-z2-7]{12}$`)
	a, b := NewID(), NewID()
	if !re.MatchString(a) || a == b {
		t.Errorf("ids %q %q", a, b)
	}
}
