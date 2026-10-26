// SPDX-License-Identifier: Apache-2.0

package access

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestResolve(t *testing.T) {
	groups := Groups{"familie": {"anna", "ben"}, "oma": {"ghost"}}
	members := []Member{
		{User: "anna", Role: Viewer},
		{Group: "familie", Role: Editor},
		{User: "carl", Role: Viewer},
		{Group: "oma", Role: Viewer},
	}
	cases := []struct {
		name string
		p    Principal
		want Role
	}{
		{"no entry", Principal{Sub: "dora"}, None},
		{"highest of user and group", Principal{Sub: "anna"}, Editor},
		{"user only", Principal{Sub: "carl"}, Viewer},
		{"admin without entry", Principal{Sub: "erik", Admin: true}, Owner},
		{"group member unknown elsewhere", Principal{Sub: "ghost"}, Viewer},
	}
	for _, c := range cases {
		if got := Resolve(c.p, members, groups); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestResolveUnknownGroupGivesNothing(t *testing.T) {
	if got := Resolve(Principal{Sub: "x"}, []Member{{Group: "missing", Role: Owner}}, Groups{}); got != None {
		t.Errorf("got %v", got)
	}
}

func TestCan(t *testing.T) {
	want := map[Role][4]bool{ // Read, Write, ManageSpace, BreakLock
		None:   {false, false, false, false},
		Viewer: {true, false, false, false},
		Editor: {true, true, false, false},
		Owner:  {true, true, true, true},
	}
	actions := [4]Action{Read, Write, ManageSpace, BreakLock}
	for role, w := range want {
		for i, a := range actions {
			if got := Can(role, a); got != w[i] {
				t.Errorf("Can(%v, %d) = %v", role, a, got)
			}
		}
	}
}

func TestParseRole(t *testing.T) {
	for _, r := range []Role{Viewer, Editor, Owner} {
		got, err := ParseRole(r.String())
		if err != nil || got != r {
			t.Errorf("round trip %v: %v %v", r, got, err)
		}
	}
	if _, err := ParseRole("god"); err == nil {
		t.Error("unknown role accepted")
	}
}

func TestGroupsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	g, err := LoadGroups(dir)
	if err != nil || len(g) != 0 {
		t.Fatalf("missing file: %v %v", g, err)
	}
	if err := SaveGroups(dir, Groups{"familie": {"anna"}}); err != nil {
		t.Fatal(err)
	}
	g, err = LoadGroups(dir)
	if err != nil || len(g["familie"]) != 1 {
		t.Errorf("got %v %v", g, err)
	}
}

func TestMemberYAMLUsesRoleNames(t *testing.T) {
	b, err := yaml.Marshal([]Member{{Group: "familie", Role: Editor}})
	if err != nil || !strings.Contains(string(b), "role: editor") {
		t.Fatalf("yaml = %s, %v", b, err)
	}
	var back []Member
	if err := yaml.Unmarshal(b, &back); err != nil || back[0].Role != Editor {
		t.Errorf("back = %+v, %v", back, err)
	}
	if err := yaml.Unmarshal([]byte("- user: x\n  role: god\n"), &back); err == nil {
		t.Error("unknown role accepted")
	}
}
