// SPDX-License-Identifier: Apache-2.0

package reports

import (
	"testing"

	"github.com/bjblazko/kanso/internal/store"
)

func TestStorageSplitsAttachmentsTextDeleted(t *testing.T) {
	docs := []Doc{
		{Kind: Page, Space: "haus", Bytes: 10},
		{Kind: Attachment, Space: "haus", Bytes: 100},
		{Kind: Note, Space: "notes", Bytes: 5},
	}
	sc := store.Scan{
		Live:      map[string]int64{"spaces/haus/pages/a/index.md": 10},
		LiveBlobs: map[string]bool{"new": true},
		Changes: []store.FileChange{
			{Path: "spaces/haus/pages/a/index.md", Blob: "old", Size: 7},
			{Path: "spaces/haus/pages/a/index.md", Blob: "new", Size: 10},
			{Path: "boards/b/tasks/t.md", Blob: "gone", Size: 3},
			{Path: "boards/b/tasks/t.md", Deleted: true},
		},
	}
	areas := Storage(docs, &sc)
	got := map[string]Area{}
	for _, a := range areas {
		got[a.Name] = a
	}
	if h := got["haus"]; h.Text != 10 || h.Attachments != 100 || h.Deleted != 7 {
		t.Errorf("haus = %+v", h)
	}
	if n := got["notes"]; n.Text != 5 || n.Deleted != 0 {
		t.Errorf("notes = %+v", n)
	}
	if b := got["boards"]; b.Deleted != 3 {
		t.Errorf("boards = %+v", b)
	}
	if areas[0].Name != "haus" {
		t.Errorf("largest area not first: %+v", areas)
	}
	if without := Storage(docs, nil); len(without) != 2 {
		t.Errorf("without history: %+v", without)
	}
}

func TestStorageOverTimeTracksLiveAndRepo(t *testing.T) {
	jan, feb := day("2026-01-10"), day("2026-02-10")
	sc := store.Scan{Changes: []store.FileChange{
		{When: jan, Path: "a", Blob: "a1", Size: 10},
		{When: jan, Path: "b", Blob: "b1", Size: 5},
		{When: feb, Path: "a", Deleted: true},
		{When: feb, Path: "c", From: "b", Blob: "b1", Size: 5},
	}}
	pts := StorageOverTime(&sc, day("2026-02-20"))
	if len(pts) != 2 {
		t.Fatalf("points %+v", pts)
	}
	if pts[0].Repo != 15 || pts[0].Tree != 15 || pts[1].Repo != 15 || pts[1].Tree != 5 {
		t.Errorf("points %+v", pts)
	}
}

func TestOrphansFindsUnreferenced(t *testing.T) {
	page := Doc{Kind: Page, ID: "p"}
	atts := []AttachmentRef{
		{Doc: Doc{Kind: Attachment, Title: "a b.png", Bytes: 3}, Name: "a b.png", Body: "![](attachments/a%20b.png)"},
		{Doc: Doc{Kind: Attachment, Title: "plain.pdf", Bytes: 4}, Name: "plain.pdf", Body: "[x](attachments/plain.pdf)"},
		{Doc: Doc{Kind: Attachment, Title: "lost.jpg", Bytes: 9}, Name: "lost.jpg", Body: "nothing", Parent: page},
	}
	got := Orphans(atts)
	if len(got) != 1 || got[0].Name != "lost.jpg" || got[0].Bytes != 9 {
		t.Errorf("orphans = %+v", got)
	}
}

func TestLargestTopN(t *testing.T) {
	docs := []Doc{{Title: "s", Bytes: 1}, {Title: "l", Bytes: 30}, {Title: "m", Bytes: 20}}
	got := Largest(docs, 2)
	if len(got) != 2 || got[0].Title != "l" || got[1].Title != "m" {
		t.Errorf("largest = %+v", got)
	}
}
