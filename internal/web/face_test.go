// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/tasks"
)

func TestInitials(t *testing.T) {
	for in, want := range map[string]string{
		"Anna Becker":              "AB",
		"Ben":                      "B",
		"  carla  von  Weide ":     "CW",
		"Élise Müller-Lüdenscheid": "ÉM",
		"":                         "",
	} {
		if got := initials(in); got != want {
			t.Errorf("initials(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCardShowsClaimantAvatar(t *testing.T) {
	ts := newTestServer(t)
	anna := ts.client(t, "anna", "Anna", false)
	ts.client(t, "ben", "Ben Becker", false)
	fam := familyBoardID(t, ts)
	c := addTask(t, ts, "anna", tasks.New{Board: fam, Title: "Hof fegen"})
	if _, err := ts.deps.Tasks.Claim(taskActor(ts, "ben"), fam, c.Meta.ID, true); err != nil {
		t.Fatal(err)
	}
	_, body := get(t, anna, ts.URL+"/tasks/b/"+fam)
	for _, want := range []string{`class="face" title="Ben Becker"`, `class="face-initials" aria-hidden="true">BB<`} {
		if !strings.Contains(body, want) {
			t.Errorf("board: missing %q", want)
		}
	}
	if _, body := get(t, anna, ts.URL+taskURL(c)); !strings.Contains(body, `title="Ben Becker"`) {
		t.Error("task page: no avatar with full name")
	}
}
