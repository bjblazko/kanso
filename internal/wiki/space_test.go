// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// twoSpaces: "haus" with a shared page and a recipe with one attachment, "garten" with one page.
func twoSpaces(t *testing.T) (*Wiki, Page, Page, Page) {
	t.Helper()
	w := newWiki(t)
	if err := os.MkdirAll(filepath.Join(w.dataDir, "spaces", "garten"), 0o750); err != nil {
		t.Fatal(err)
	}
	shared := mustCreate(t, w, "", "Wunschliste")
	if err := w.SetMembers(anna, shared.Meta.ID, christina); err != nil {
		t.Fatal(err)
	}
	recipe := mustCreate(t, w, "", "Curry")
	if err := w.SaveWith(anna, recipe.Meta.ID, "Curry", "", func(m *PageMeta) { m.Type = TypeRecipe }); err != nil {
		t.Fatal(err)
	}
	if _, err := w.AddAttachment(anna, recipe.Meta.ID, "foto.jpg", strings.NewReader("x"), 1<<20); err != nil {
		t.Fatal(err)
	}
	other, err := w.Create(anna, "garten", "", "Rasen")
	if err != nil {
		t.Fatal(err)
	}
	return w, shared, recipe, other
}

func TestSpaceStats(t *testing.T) {
	w, shared, recipe, _ := twoSpaces(t)
	st := w.SpaceStats("haus")
	if st.Pages != 2 || st.Recipes != 1 || st.Attachments != 1 || st.Shared != 1 {
		t.Errorf("stats = %+v", st)
	}
	slices.Sort(st.IDs)
	want := []string{shared.Meta.ID, recipe.Meta.ID}
	slices.Sort(want)
	if !slices.Equal(st.IDs, want) {
		t.Errorf("ids = %v, want %v", st.IDs, want)
	}
}

func TestForgetSpace(t *testing.T) {
	w, shared, recipe, other := twoSpaces(t)
	var gone []string
	w.OnChange = func(p Page, deleted bool) {
		if deleted {
			gone = append(gone, p.Meta.ID)
		}
	}
	w.ForgetSpace("haus")
	for _, id := range []string{shared.Meta.ID, recipe.Meta.ID} {
		if _, ok := w.ByID(id); ok {
			t.Errorf("%s still indexed", id)
		}
	}
	if _, ok := w.ByID(other.Meta.ID); !ok {
		t.Error("other space's page forgotten")
	}
	slices.Sort(gone)
	want := []string{shared.Meta.ID, recipe.Meta.ID}
	slices.Sort(want)
	if !slices.Equal(gone, want) {
		t.Errorf("notified %v, want %v", gone, want)
	}
}

func TestPagePathAndAttachmentPath(t *testing.T) {
	w := newWiki(t)
	p := mustCreate(t, w, "", "Keller")
	if got := PagePath(p); got != "spaces/haus/pages/"+p.Dir+"/index.md" {
		t.Errorf("PagePath = %q", got)
	}
	if got := AttachmentPath(p, "a.png"); got != "spaces/haus/pages/"+p.Dir+"/attachments/a.png" {
		t.Errorf("AttachmentPath = %q", got)
	}
}
