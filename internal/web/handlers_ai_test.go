// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

const tokenValue = "sk-secret-123"

// fakeAI answers chat completions with answer.
func fakeAI(t *testing.T, answer string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + jsonString(answer) + `}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/v1"
}

func jsonString(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}

func TestAdminAIKeepsTokenHidden(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	if code := status(t, anna, ts.URL+"/admin/ai"); code != http.StatusForbidden {
		t.Errorf("anna: %d", code)
	}
	base := fakeAI(t, "ok")
	resp := ts.postForm(t, admin, "/admin/ai", "/admin/ai", url.Values{"action": {"save"}, "enabled": {"1"}, "base_url": {base}, "model": {"qwen"}, "token": {tokenValue}})
	if resp.StatusCode != http.StatusSeeOther || ts.deps.AI.Load().Token != tokenValue {
		t.Fatalf("%d %+v", resp.StatusCode, ts.deps.AI.Load())
	}
	_, body := get(t, admin, ts.URL+"/admin/ai")
	if strings.Contains(body, tokenValue) || !strings.Contains(body, "Token ist gesetzt") || !strings.Contains(body, "Auftragsverarbeitung") {
		t.Fatal("token shown or no note")
	}
	ts.postForm(t, admin, "/admin/ai", "/admin/ai", url.Values{"action": {"save"}, "enabled": {"1"}, "base_url": {base}, "model": {"qwen"}, "token": {""}})
	if ts.deps.AI.Load().Token != tokenValue {
		t.Fatal("empty field dropped the token")
	}
	_, body = ts.postFormBody(t, admin, "/admin/ai", "/admin/ai", url.Values{"action": {"test"}, "enabled": {"1"}, "base_url": {base}, "model": {"qwen"}})
	if !strings.Contains(body, "Verbindung steht") {
		t.Fatalf("test: %s", between(body, "<main", "</main>"))
	}
	ts.postForm(t, admin, "/admin/ai", "/admin/ai", url.Values{"action": {"save"}, "enabled": {"1"}, "base_url": {base}, "model": {"qwen"}, "remove_token": {"1"}})
	if ts.deps.AI.Load().Token != "" {
		t.Fatal("token not removed")
	}
	_, body = ts.postFormBody(t, admin, "/admin/ai", "/admin/ai", url.Values{"action": {"test"}, "enabled": {"1"}, "base_url": {"http://127.0.0.1:1/v1"}, "model": {"qwen"}})
	if !strings.Contains(body, "nicht erreichbar") {
		t.Fatal("no error for an unreachable AI")
	}
}
