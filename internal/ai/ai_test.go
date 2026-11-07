// SPDX-License-Identifier: Apache-2.0

package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreKeepsTokenOutOfTheWay(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(dir)
	if c := s.Load(); c.Enabled || c.Ready() {
		t.Fatalf("empty store: %+v", c)
	}
	cfg := Config{Enabled: true, BaseURL: "https://openrouter.ai/api/v1/", Model: "m", Token: "secret"}
	if err := s.Save(cfg); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, "ai.json"))
	if err != nil || fi.Mode().Perm() != 0o600 {
		t.Fatalf("file mode %v %v", fi.Mode(), err)
	}
	got := s.Load()
	if got.Token != "secret" || got.BaseURL != "https://openrouter.ai/api/v1" || !got.Ready() || got.Host() != "openrouter.ai" {
		t.Fatalf("%+v host %q", got, got.Host())
	}
	if (Config{Enabled: true, BaseURL: "ftp://x", Model: "m"}).Ready() {
		t.Error("ftp is ready")
	}
}

// fake answers like an OpenAI-compatible server.
func fake(t *testing.T, status int, body string, check func(r *http.Request)) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if check != nil {
			check(r)
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestComplete(t *testing.T) {
	var seen struct {
		Model    string    `json:"model"`
		Messages []Message `json:"messages"`
		Stream   bool      `json:"stream"`
	}
	srv := fake(t, 200, `{"choices":[{"message":{"role":"assistant","content":"<think>hm</think>\nok"}}]}`, func(r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Header.Get("Authorization") != "Bearer tok" {
			t.Errorf("%s %q", r.URL.Path, r.Header.Get("Authorization"))
		}
		_ = json.NewDecoder(r.Body).Decode(&seen)
	})
	cfg := Config{Enabled: true, BaseURL: srv.URL + "/v1", Model: "qwen", Token: "tok"}
	got, err := Client{}.Complete(context.Background(), cfg, []Message{{Role: "user", Content: "hi"}})
	if err != nil || got != "ok" {
		t.Fatalf("%q %v", got, err)
	}
	if seen.Model != "qwen" || seen.Messages[0].Content != "hi" || seen.Stream {
		t.Errorf("%+v", seen)
	}
}

func TestCompleteNoTokenNoHeader(t *testing.T) {
	srv := fake(t, 200, `{"choices":[{"message":{"content":"ok"}}]}`, func(r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			t.Error("authorization without token")
		}
	})
	if _, err := (Client{}).Complete(context.Background(), Config{Enabled: true, BaseURL: srv.URL, Model: "m"}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCompleteErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   error
	}{
		{401, `{"error":"bad key"}`, ErrAuth},
		{404, `{"error":{"message":"model 'x' not found"}}`, ErrModel},
		{200, `not json`, ErrBadAnswer},
		{200, `{"choices":[]}`, ErrBadAnswer},
		{500, `boom`, ErrUnreachable},
	}
	for _, c := range cases {
		srv := fake(t, c.status, c.body, nil)
		_, err := Client{}.Complete(context.Background(), Config{Enabled: true, BaseURL: srv.URL, Model: "x"}, nil)
		if !errors.Is(err, c.want) {
			t.Errorf("%d %s: %v", c.status, c.body, err)
		}
	}
	_, err := Client{}.Complete(context.Background(), Config{Enabled: true, BaseURL: "http://127.0.0.1:1", Model: "x"}, nil)
	if !errors.Is(err, ErrUnreachable) {
		t.Errorf("closed port: %v", err)
	}
	slow := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { time.Sleep(300 * time.Millisecond) }))
	defer slow.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := (Client{}).Complete(ctx, Config{Enabled: true, BaseURL: slow.URL, Model: "x"}, nil); !errors.Is(err, ErrTimeout) {
		t.Errorf("timeout: %v", err)
	}
}
