// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func readPageFile(t *testing.T, w *Wiki, p Page) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(w.dataDir, filepath.FromSlash(pagePath(p.Space, p.Dir))))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestSaveWithRecipeMeta(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Curry")
	err := w.SaveWith(anna, p.Meta.ID, "Curry", "## Zutaten\n", func(m *PageMeta) {
		m.Type, m.Servings, m.Tags, m.Source, m.Time = "recipe", 4, []string{"deftig"}, "Oma", 40
	})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	if !got.Meta.IsRecipe() || got.Meta.Servings != 4 || !slices.Equal(got.Meta.Tags, []string{"deftig"}) ||
		got.Meta.Source != "Oma" || got.Meta.Time != 40 || got.Body != "## Zutaten\n" {
		t.Fatalf("%+v", got)
	}
	file := readPageFile(t, w, got)
	if !strings.Contains(file, "type: recipe") || !strings.Contains(file, "servings: 4") {
		t.Fatalf("file:\n%s", file)
	}
}

func TestSavePlainPageKeepsFrontmatter(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Wunschliste")
	if err := w.SetMembers(anna, p.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	if err := w.Save(anna, p.Meta.ID, "Wunschliste", "Text"); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	file := readPageFile(t, w, got)
	for _, field := range []string{"type:", "servings:", "tags:", "source:", "time:"} {
		if strings.Contains(file, field) {
			t.Errorf("plain page got %s\n%s", field, file)
		}
	}
	if len(got.Meta.Members) != 1 || got.Meta.IsRecipe() {
		t.Errorf("%+v", got.Meta)
	}
}

func TestSaveKeepsRecipeMeta(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Curry")
	_ = w.SaveWith(anna, p.Meta.ID, "Curry", "", func(m *PageMeta) { m.Type, m.Servings = "recipe", 4 })
	if err := w.Save(anna, p.Meta.ID, "Curry", "neu"); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	if !got.Meta.IsRecipe() || got.Meta.Servings != 4 {
		t.Fatalf("%+v", got.Meta)
	}
}

func TestRestoreBringsBackRecipeFields(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Curry")
	_ = w.SaveWith(anna, p.Meta.ID, "Curry", "a", func(m *PageMeta) { m.Type, m.Tags = TypeRecipe, []string{"deftig"} })
	revs, err := w.History(p.Meta.ID, 1)
	if err != nil || len(revs) != 1 {
		t.Fatal(err, revs)
	}
	_ = w.SaveWith(anna, p.Meta.ID, "Curry", "b", func(m *PageMeta) { m.Type, m.Tags = "", nil })
	if err := w.Restore(anna, p.Meta.ID, revs[0].Hash); err != nil {
		t.Fatal(err)
	}
	got, _ := w.ByID(p.Meta.ID)
	if !got.Meta.IsRecipe() || !slices.Equal(got.Meta.Tags, []string{"deftig"}) || got.Body != "a" {
		t.Fatalf("%+v %q", got.Meta, got.Body)
	}
}
