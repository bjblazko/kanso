// SPDX-License-Identifier: Apache-2.0

// Package ai talks to one large language model through the OpenAI-compatible
// chat completions API (ADR-013): Ollama, OpenRouter, OpenAI and others.
// It never logs what it sends or receives.
package ai

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config is the connection an admin registered. The token stays in the
// state directory, never in the data repository or on a page.
type Config struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"baseURL"`
	Model   string `json:"model"`
	Token   string `json:"token,omitempty"`
}

// Ready tells whether the connection is switched on and complete.
func (c Config) Ready() bool {
	u, err := url.Parse(c.BaseURL)
	return c.Enabled && err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" && c.Model != ""
}

// Host names where text goes, for the note before sending.
func (c Config) Host() string {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return ""
	}
	return u.Hostname()
}

// Store keeps the configuration in <stateDir>/ai.json.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore opens the configuration in stateDir.
func NewStore(stateDir string) *Store { return &Store{path: filepath.Join(stateDir, "ai.json")} }

// Load returns the configuration; none means switched off.
func (s *Store) Load() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	var c Config
	b, err := os.ReadFile(s.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Config{}
	}
	_ = json.Unmarshal(b, &c)
	return c
}

// Save stores c readable by this process only.
func (s *Store) Save(c Config) error {
	c.BaseURL = strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	c.Model, c.Token = strings.TrimSpace(c.Model), strings.TrimSpace(c.Token)
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeSecret(s.path, append(b, '\n'))
}

// writeSecret replaces path atomically; the file is 0600 from its first byte.
func writeSecret(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".ai-*") // CreateTemp makes 0600
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
