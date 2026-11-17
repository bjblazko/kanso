// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/ai"
)

const genAnswer = "# Paprika-Reis-Pfanne\nPortionen: 4\nZeit: 30\n\nDeftig.\n\n## Zutaten\n\n- 500 g Reis\n- 2 Paprika\n\n## Zubereitung\n\n1. Kochen."

// seeingAI answers like fakeAI and keeps the last request body.
func seeingAI(t *testing.T, answer string, seen *string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		*seen = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + jsonString(answer) + `}}]}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL + "/v1"
}

func wishForm(kv ...string) url.Values {
	v := url.Values{"ingredients": {"Reis, Paprika"}, "style": {"homestyle"}, "servings": {"4"}}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}

func TestGenerateButtonOnlyWithAI(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	if _, body := get(t, anna, ts.URL+"/recipes"); strings.Contains(body, `href="/recipes/generate"`) {
		t.Fatal("button without AI")
	}
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: fakeAI(t, genAnswer), Model: "m"})
	if _, body := get(t, anna, ts.URL+"/recipes"); !strings.Contains(body, `href="/recipes/generate"`) {
		t.Fatal("no button with AI")
	}
}

func TestGenerateThenAccept(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: fakeAI(t, genAnswer), Model: "m"})
	if _, form := get(t, anna, ts.URL+"/recipes/generate"); !strings.Contains(form, "127.0.0.1") || !strings.Contains(form, `name="ingredients"`) {
		t.Fatal("form")
	}
	resp, body := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm())
	if resp.StatusCode != http.StatusOK || !strings.Contains(body, "Paprika-Reis-Pfanne") || !strings.Contains(body, "500 · g · Reis") ||
		!strings.Contains(body, `name="style" value="homestyle"`) || !strings.Contains(body, `name="generated" value="1"`) {
		t.Fatalf("%d\n%s", resp.StatusCode, body)
	}
	create := url.Values{"title": {"Paprika-Reis-Pfanne"}, "servings": {"4"}, "time": {"30"}, "generated": {"1"},
		"body": {htmlUnescape(between(body, `name="body" rows="16">`, `</textarea>`))}}
	resp = ts.postForm(t, anna, "/recipes/generate", "/recipes/import/create", create)
	loc := resp.Header.Get("Location")
	if resp.StatusCode != http.StatusSeeOther || !strings.HasSuffix(loc, "/edit") {
		t.Fatalf("%d %q", resp.StatusCode, loc)
	}
	p, _ := ts.deps.Wiki.ByID(strings.TrimSuffix(strings.TrimPrefix(loc, "/p/"), "/edit"))
	if !p.Meta.IsRecipe() || p.Meta.Source != "" || !slices.Contains(p.Meta.Tags, "ki-generiert") || !strings.Contains(p.Body, "- 500 g Reis") {
		t.Fatalf("%+v\n%s", p.Meta, p.Body)
	}
}

func TestImportCreateHasNoTagWithoutGenerated(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	resp := ts.postForm(t, anna, "/recipes/import", "/recipes/import/create", url.Values{"title": {"X"}, "body": {"## Zutaten\n- 1 Ei\n"}, "generated": {"yes"}})
	p, _ := ts.deps.Wiki.ByID(strings.TrimSuffix(strings.TrimPrefix(resp.Header.Get("Location"), "/p/"), "/edit"))
	if len(p.Meta.Tags) != 0 {
		t.Fatalf("%q", p.Meta.Tags)
	}
}

func TestGenerateRerollSendsRejected(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	var seen string
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: seeingAI(t, genAnswer, &seen), Model: "m"})
	resp, body := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm("action", "reroll", "rejected_now", "Omas Apfelkuchen"))
	var req struct{ Messages []struct{ Content string } }
	_ = json.Unmarshal([]byte(seen), &req)
	if resp.StatusCode != http.StatusOK || len(req.Messages) != 2 || !strings.Contains(req.Messages[1].Content, "<schon_verworfen>\nOmas Apfelkuchen\n</schon_verworfen>") {
		t.Fatalf("%d %+v", resp.StatusCode, req)
	}
	if !strings.Contains(body, `name="rejected" value="Omas Apfelkuchen"`) {
		t.Fatal("rejected title not carried")
	}
}

func TestGenerateRefusesBadChoice(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: fakeAI(t, genAnswer), Model: "m"})
	if resp, _ := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm("style", "evil")); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("%d", resp.StatusCode)
	}
}

func TestGenerateNonRecipeShowsError(t *testing.T) {
	ts, _, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: fakeAI(t, "Paris ist die Hauptstadt."), Model: "m"})
	resp, body := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm())
	if resp.StatusCode != http.StatusBadRequest || strings.Contains(body, "Paris") || !strings.Contains(body, "kein Rezept") || !strings.Contains(body, "Reis, Paprika") {
		t.Fatalf("%d\n%s", resp.StatusCode, body)
	}
}

func TestGenerateNeedsWriteAndAI(t *testing.T) {
	ts, admin, anna := setupHaus(t)
	ts.cookbook(t, "haus")
	resp, body := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm())
	if resp.StatusCode != http.StatusBadRequest || !strings.Contains(body, "KI-Anbindung") {
		t.Fatalf("no AI: %d", resp.StatusCode)
	}
	_ = ts.deps.AI.Save(ai.Config{Enabled: true, BaseURL: fakeAI(t, genAnswer), Model: "m"})
	ts.postForm(t, admin, "/", "/admin/spaces/new", url.Values{"name": {"Geheim"}})
	ts.cookbook(t, "geheim")
	if resp, _ := get(t, anna, ts.URL+"/recipes/generate"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("GET %d", resp.StatusCode)
	}
	if resp, _ := ts.postFormBody(t, anna, "/recipes/generate", "/recipes/generate", wishForm()); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("POST %d", resp.StatusCode)
	}
}
