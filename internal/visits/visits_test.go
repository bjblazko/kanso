// SPDX-License-Identifier: Apache-2.0

package visits

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var t0 = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func touch(t *testing.T, s *Store, at time.Time) time.Time {
	t.Helper()
	prev, err := s.Touch("anna", at)
	if err != nil {
		t.Fatal(err)
	}
	return prev
}

func TestTouchFirstVisit(t *testing.T) {
	if prev := touch(t, NewStore(t.TempDir()), t0); !prev.IsZero() {
		t.Errorf("first visit: %v", prev)
	}
}

func TestTouchWithinHourKeepsWindow(t *testing.T) {
	s := NewStore(t.TempDir())
	touch(t, s, t0)
	if prev := touch(t, s, t0.Add(30*time.Minute)); !prev.IsZero() {
		t.Errorf("within the hour moved: %v", prev)
	}
	if prev := touch(t, s, t0.Add(2*time.Hour)); !prev.Equal(t0) {
		t.Errorf("after two hours: %v", prev)
	}
	if prev := touch(t, s, t0.Add(150*time.Minute)); !prev.Equal(t0) {
		t.Errorf("within the hour after the move: %v", prev)
	}
}

func TestTouchPersists(t *testing.T) {
	dir := t.TempDir()
	touch(t, NewStore(dir), t0)
	if prev := touch(t, NewStore(dir), t0.Add(2*time.Hour)); !prev.Equal(t0) {
		t.Errorf("not kept: %v", prev)
	}
	fi, err := os.Stat(filepath.Join(dir, "visits.json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Errorf("file: %v %v", fi, err)
	}
}

func TestTouchSurvivesBrokenFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "visits.json")
	if err := os.WriteFile(path, []byte("{nope"), 0o600); err != nil {
		t.Fatal(err)
	}
	if prev := touch(t, NewStore(dir), t0); !prev.IsZero() {
		t.Errorf("broken file gave %v", prev)
	}
	b, _ := os.ReadFile(path)
	if !json.Valid(b) {
		t.Errorf("not rewritten: %s", b)
	}
}

func TestTouchRefusesBadSub(t *testing.T) {
	if _, err := NewStore(t.TempDir()).Touch("", t0); err == nil {
		t.Error("empty sub accepted")
	}
}

func TestSince(t *testing.T) {
	now := t0
	day := now.Add(-Floor)
	for _, c := range []struct {
		prev, want time.Time
	}{
		{time.Time{}, day},
		{now.Add(-3 * time.Hour), day},
		{now.Add(-72 * time.Hour), now.Add(-72 * time.Hour)},
	} {
		if got := Since(c.prev, now); !got.Equal(c.want) {
			t.Errorf("Since(%v) = %v, want %v", c.prev, got, c.want)
		}
	}
}
