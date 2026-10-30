// SPDX-License-Identifier: Apache-2.0

package tasks

import (
	"strings"
	"testing"
	"time"
)

func task(title string, m Meta) Task {
	m.Title = title
	if m.Status == "" {
		m.Status = Open
	}
	return Task{Meta: m}
}

func titles(ts []Task) string {
	var out []string
	for _, t := range ts {
		out = append(out, t.Meta.Title)
	}
	return strings.Join(out, ",")
}

func TestSorted(t *testing.T) {
	ts := []Task{
		task("nov2", Meta{Due: "2026-11-02"}),
		task("older", Meta{Created: at("2026-10-30T08:00:00Z")}),
		task("nov1", Meta{Due: "2026-11-01"}),
		task("newer", Meta{Created: at("2026-10-30T09:00:00Z")}),
	}
	if got := titles(Sorted(ts)); got != "nov1,nov2,newer,older" {
		t.Fatalf("Sorted = %s", got)
	}
}

func TestKanbanHidesOldDone(t *testing.T) {
	now := at("2026-10-30T12:00:00Z")
	recent, old := now.AddDate(0, 0, -13), now.AddDate(0, 0, -15)
	ts := []Task{
		task("recent", Meta{Status: Done, DoneAt: &recent}),
		task("old", Meta{Status: Done, DoneAt: &old}),
		task("open", Meta{}),
	}
	k := Kanban(ts, now, false)
	if titles(k[Done]) != "recent" || titles(k[Open]) != "open" {
		t.Fatalf("Kanban = %v", k)
	}
	if got := titles(Kanban(ts, now, true)[Done]); got != "recent,old" && got != "old,recent" {
		t.Fatalf("all done = %s", got)
	}
}

func TestMatrixSkipsDone(t *testing.T) {
	ts := []Task{
		task("now", Meta{Urgent: true, Important: true}),
		task("done", Meta{Urgent: true, Important: true, Status: Done}),
		task("later", Meta{}),
	}
	m := Matrix(ts)
	if titles(m[DoNow]) != "now" || titles(m[Later]) != "later" {
		t.Fatalf("Matrix = %v", m)
	}
}

func TestQuadrantOf(t *testing.T) {
	cases := map[Quadrant]Meta{
		DoNow: {Urgent: true, Important: true},
		Plan:  {Important: true},
		Quick: {Urgent: true},
		Later: {},
	}
	for want, m := range cases {
		if got := QuadrantOf(Task{Meta: m}); got != want {
			t.Errorf("QuadrantOf(%+v) = %v, want %v", m, got, want)
		}
	}
}

func TestDueOf(t *testing.T) {
	now := time.Date(2026, 10, 30, 23, 30, 0, 0, time.Local)
	cases := map[string]DueState{
		"":           DueNone,
		"2026-10-29": DueOverdue,
		"2026-10-30": DueToday,
		"2026-10-31": DueTomorrow,
		"2026-11-01": DueLater,
		"garbage":    DueNone,
	}
	for due, want := range cases {
		if got := DueOf(due, now); got != want {
			t.Errorf("DueOf(%q) = %v, want %v", due, got, want)
		}
	}
}

func TestFromNote(t *testing.T) {
	title, rest := FromNote("\n\n# Hof fegen\n\nmit Besen\n")
	if title != "Hof fegen" || rest != "mit Besen\n" {
		t.Fatalf("FromNote = %q, %q", title, rest)
	}
	long := strings.Repeat("ö", 250)
	if title, _ := FromNote(long); len([]rune(title)) != maxTitle {
		t.Fatalf("title has %d runes", len([]rune(title)))
	}
	if title, rest := FromNote(""); title != "" || rest != "" {
		t.Fatalf("empty = %q, %q", title, rest)
	}
}

func TestFromNoteLikeNoteTitles(t *testing.T) {
	title, rest := FromNote("- Milch kaufen\n* Brot\n")
	if title != "Milch kaufen" || rest != "* Brot\n" {
		t.Fatalf("list mark: %q, %q", title, rest)
	}
	body := "![](attachments/a.jpg)\n\nKind anmelden\nMail an den Hort\n\n![](attachments/b.jpg)\n"
	title, rest = FromNote(body)
	if title != "Kind anmelden" || strings.Contains(rest, "attachments/") || !strings.Contains(rest, "Mail an den Hort") {
		t.Fatalf("photos: %q, %q", title, rest)
	}
	if title, rest := FromNote("![](attachments/a.jpg)\n"); title != "" || rest != "" {
		t.Fatalf("photo only: %q, %q", title, rest)
	}
	if title, _ := FromNote("#\nEinkauf\n"); title != "Einkauf" {
		t.Fatalf("empty heading: %q", title)
	}
}
