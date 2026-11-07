// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/ai"
)

func fakeAI(t *testing.T, answer string, seen *string) ai.Config {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		if seen != nil {
			*seen = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":` + quote(answer) + `}}]}`))
	}))
	t.Cleanup(srv.Close)
	return ai.Config{Enabled: true, BaseURL: srv.URL, Model: "m"}
}

func quote(s string) string {
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)
	return `"` + r.Replace(s) + `"`
}

const answer = "<think>Erst lesen.</think>\n```markdown\n# Omas Apfelkuchen\nPortionen: 12\nZeit: 75\n\nSaftig.\n\n## Zutaten\n\n- 500 g Äpfel\n- 200 g Mehl\n\n## Zubereitung\n\n1. Äpfel schneiden.\n2. Backen.\n```"

func TestFromAI(t *testing.T) {
	var seen string
	cfg := fakeAI(t, answer, &seen)
	page := Page{URL: "https://blog.example/apfel", HTML: "<html><body><h1>Omas Apfelkuchen</h1><p>500 g Äpfel</p><script>x</script></body></html>"}
	d, err := FromAI(context.Background(), ai.Client{}, cfg, page)
	if err != nil {
		t.Fatal(err)
	}
	if d.Title != "Omas Apfelkuchen" || d.Servings != 12 || d.Time != 75 || d.Description != "Saftig." || d.Source != page.URL ||
		!slices.Equal(d.Ingredients, []string{"500 g Äpfel", "200 g Mehl"}) || !slices.Equal(d.Steps, []string{"Äpfel schneiden.", "Backen."}) {
		t.Fatalf("%+v", d)
	}
	if !strings.Contains(seen, "500 g Äpfel") || strings.Contains(seen, "<script>") || !strings.Contains(seen, "## Zutaten") {
		t.Errorf("request: %s", seen)
	}
}

func TestFromAIWithoutRecipe(t *testing.T) {
	cfg := fakeAI(t, "Auf dieser Seite steht kein Rezept.", nil)
	if _, err := FromAI(context.Background(), ai.Client{}, cfg, Page{URL: "https://x.example", HTML: "<p>Hallo</p>"}); !errors.Is(err, ErrNoRecipe) {
		t.Fatalf("%v", err)
	}
}

func TestFromAICutsLongPages(t *testing.T) {
	var seen string
	cfg := fakeAI(t, answer, &seen)
	long := "<p>" + strings.Repeat("Wort ", 20000) + "</p>"
	if _, err := FromAI(context.Background(), ai.Client{}, cfg, Page{URL: "https://x.example", HTML: long}); err != nil {
		t.Fatal(err)
	}
	if len(seen) > maxText+8000 {
		t.Errorf("sent %d bytes", len(seen))
	}
}
