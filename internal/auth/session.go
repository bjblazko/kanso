// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	sessionTTL     = 30 * 24 * time.Hour
	sessionRefresh = 24 * time.Hour // extend at most once a day
	sessionsFile   = "sessions.json"
	keyFile        = "cookie.key"
)

// Session is a signed-in browser.
type Session struct {
	ID      string    `json:"id"`
	Sub     string    `json:"sub"`
	CSRF    string    `json:"csrf"`
	Admin   bool      `json:"admin"`
	Expires time.Time `json:"expires"`
}

// Sessions keeps sessions in memory and in the state directory.
type Sessions struct {
	path string
	mu   sync.Mutex
	m    map[string]Session
	now  func() time.Time
}

// NewSessions loads the sessions stored in stateDir.
func NewSessions(stateDir string) (*Sessions, error) {
	s := &Sessions{path: filepath.Join(stateDir, sessionsFile), m: map[string]Session{}, now: time.Now}
	b, err := os.ReadFile(s.path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, os.MkdirAll(stateDir, 0o700)
	}
	if err != nil {
		return nil, err
	}
	return s, json.Unmarshal(b, &s.m)
}

// Create starts a session for sub.
func (s *Sessions) Create(sub string, admin bool) Session {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess := Session{ID: randomString(), Sub: sub, CSRF: randomString(), Admin: admin, Expires: s.now().Add(sessionTTL)}
	s.m[sess.ID] = sess
	s.persist()
	return sess
}

// Get returns a live session and extends it while it is in use.
func (s *Sessions) Get(id string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.m[id]
	now := s.now()
	if !ok || now.After(sess.Expires) {
		return Session{}, false
	}
	if sess.Expires.Sub(now) < sessionTTL-sessionRefresh {
		sess.Expires = now.Add(sessionTTL)
		s.m[id] = sess
		s.persist()
	}
	return sess, true
}

// Delete ends a session.
func (s *Sessions) Delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
	s.persist()
}

// persist drops expired sessions and writes the rest; the caller holds s.mu.
func (s *Sessions) persist() {
	now := s.now()
	for id, sess := range s.m {
		if now.After(sess.Expires) {
			delete(s.m, id)
		}
	}
	b, _ := json.Marshal(s.m)
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, s.path)
	}
}

// signer protects short-lived cookies against tampering.
type signer struct{ key []byte }

func loadSigner(stateDir string) (*signer, error) {
	path := filepath.Join(stateDir, keyFile)
	key, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		key = make([]byte, 32)
		_, _ = rand.Read(key)
		err = os.WriteFile(path, key, 0o600)
	}
	if err != nil {
		return nil, err
	}
	return &signer{key: key}, nil
}

func (sg *signer) mac(v string) string {
	m := hmac.New(sha256.New, sg.key)
	m.Write([]byte(v))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (sg *signer) sign(v string) string { return v + "." + sg.mac(v) }

func (sg *signer) verify(signed string) (string, bool) {
	i := strings.LastIndexByte(signed, '.')
	if i < 0 {
		return "", false
	}
	v := signed[:i]
	return v, hmac.Equal([]byte(signed[i+1:]), []byte(sg.mac(v)))
}
