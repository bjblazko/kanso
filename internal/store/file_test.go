// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path/filepath"
	"testing"
)

type meta struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
}

func TestDocRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "a", "index.md")
	body := "# Hallo\n\n---\nnot frontmatter\n"
	if err := WriteDoc(p, Doc[meta]{Meta: meta{ID: "abc", Title: "Über: uns"}, Body: body}); err != nil {
		t.Fatal(err)
	}
	d, err := ReadDoc[meta](p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta.ID != "abc" || d.Meta.Title != "Über: uns" || d.Body != body {
		t.Errorf("round trip = %+v", d)
	}
}

func TestReadDocWithoutFrontmatter(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.md")
	os.WriteFile(p, []byte("just text\n"), 0o640)
	d, err := ReadDoc[meta](p)
	if err != nil {
		t.Fatal(err)
	}
	if d.Meta.ID != "" || d.Body != "just text\n" {
		t.Errorf("got %+v", d)
	}
}

func TestWriteAtomicReplaces(t *testing.T) {
	p := filepath.Join(t.TempDir(), "f")
	WriteAtomic(p, []byte("one"))
	if err := WriteAtomic(p, []byte("two")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	if string(b) != "two" {
		t.Errorf("content %q", b)
	}
	entries, _ := os.ReadDir(filepath.Dir(p))
	if len(entries) != 1 {
		t.Errorf("leftover temp files: %d entries", len(entries))
	}
}

func TestYAMLRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "c.yaml")
	if err := WriteYAML(p, meta{ID: "1", Title: "t"}); err != nil {
		t.Fatal(err)
	}
	var m meta
	if err := ReadYAML(p, &m); err != nil || m.ID != "1" {
		t.Errorf("got %+v, %v", m, err)
	}
}

func TestParseDoc(t *testing.T) {
	d, err := ParseDoc[meta]([]byte("---\nid: x\ntitle: T\n---\n\nText\n"))
	if err != nil || d.Meta.ID != "x" || d.Body != "Text\n" {
		t.Errorf("got %+v %v", d, err)
	}
}
