// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSessionSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	s, err := NewSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	created := s.Create("anna", true)
	again, err := NewSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := again.Get(created.ID)
	if !ok || got.Sub != "anna" || !got.Admin || got.CSRF == "" || got.CSRF != created.CSRF {
		t.Errorf("got %+v %v", got, ok)
	}
	info, _ := os.Stat(filepath.Join(dir, "sessions.json"))
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", info.Mode().Perm())
	}
}

func TestExpiredSessionIsGone(t *testing.T) {
	s, _ := NewSessions(t.TempDir())
	now := time.Now()
	s.now = func() time.Time { return now }
	created := s.Create("anna", false)
	s.now = func() time.Time { return now.Add(sessionTTL + time.Minute) }
	if _, ok := s.Get(created.ID); ok {
		t.Error("expired session returned")
	}
}

func TestDeleteSession(t *testing.T) {
	s, _ := NewSessions(t.TempDir())
	c := s.Create("anna", false)
	s.Delete(c.ID)
	if _, ok := s.Get(c.ID); ok {
		t.Error("deleted session returned")
	}
}

func TestSignerRejectsTampering(t *testing.T) {
	sg, err := loadSigner(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	v := sg.sign("hello")
	if got, ok := sg.verify(v); !ok || got != "hello" {
		t.Errorf("verify = %q %v", got, ok)
	}
	if _, ok := sg.verify("hellO" + v[5:]); ok {
		t.Error("tampered value accepted")
	}
}
