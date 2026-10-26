// SPDX-License-Identifier: Apache-2.0

package search

import (
	"strings"
	"testing"
)

func everything(string) bool { return true }

func ids(hits []Hit) []string {
	var out []string
	for _, h := range hits {
		out = append(out, h.ID)
	}
	return out
}

func TestUmlautsMatchBothSpellings(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "a", Space: "s", Title: "Größe", Body: "Kleidung"})
	ix.Put(Doc{ID: "b", Space: "s", Title: "Notiz", Body: "Groesse 98"})
	if got := ids(ix.Search("größe", everything, 10)); len(got) != 2 {
		t.Errorf("größe → %v", got)
	}
	if got := ids(ix.Search("GROESSE", everything, 10)); len(got) != 2 {
		t.Errorf("GROESSE → %v", got)
	}
}

func TestPrefixOnLastWord(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "a", Space: "s", Title: "Kindergarten", Body: "Abholen um 15 Uhr"})
	ix.Put(Doc{ID: "b", Space: "s", Title: "Garten", Body: "Rasen"})
	if got := ids(ix.Search("kind", everything, 10)); len(got) != 1 || got[0] != "a" {
		t.Errorf("kind → %v", got)
	}
	if got := ids(ix.Search("abholen 15", everything, 10)); len(got) != 1 {
		t.Errorf("all words must match: %v", got)
	}
}

func TestPermissionFilter(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "a", Space: "geheim", Title: "Tresor", Body: "Code"})
	only := func(space string) bool { return space != "geheim" }
	if got := ix.Search("tresor", only, 10); len(got) != 0 {
		t.Errorf("hidden space leaked: %v", ids(got))
	}
}

func TestTitleRanksAboveBody(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "body", Space: "s", Title: "Allgemein", Body: "Die Heizung ist im Keller."})
	ix.Put(Doc{ID: "title", Space: "s", Title: "Heizung", Body: "Wartung"})
	got := ids(ix.Search("heizung", everything, 10))
	if len(got) != 2 || got[0] != "title" {
		t.Errorf("ranking %v", got)
	}
}

func TestRemoveAndReplace(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "a", Space: "s", Title: "Alt", Body: "x"})
	ix.Put(Doc{ID: "a", Space: "s", Title: "Neu", Body: "x"})
	if len(ix.Search("alt", everything, 10)) != 0 {
		t.Error("old title still found")
	}
	ix.Remove("a")
	if len(ix.Search("neu", everything, 10)) != 0 {
		t.Error("removed doc still found")
	}
}

func TestSnippetIsPlainAndShort(t *testing.T) {
	ix := NewIndex()
	body := strings.Repeat("Füllwort ", 50) + "**Wichtig:** Zähler ablesen. " + strings.Repeat("Ende ", 50)
	ix.Put(Doc{ID: "a", Space: "s", Title: "Strom", Body: body})
	hits := ix.Search("zähler", everything, 10)
	if len(hits) != 1 || !strings.Contains(hits[0].Snippet, "Zähler ablesen") || len([]rune(hits[0].Snippet)) > 170 || strings.Contains(hits[0].Snippet, "**") {
		t.Errorf("snippet %q", hits[0].Snippet)
	}
}

func TestEmptyQuery(t *testing.T) {
	ix := NewIndex()
	ix.Put(Doc{ID: "a", Space: "s", Title: "x", Body: "y"})
	if len(ix.Search("  ", everything, 10)) != 0 {
		t.Error("empty query returned hits")
	}
}
