// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

const curryBody = `Kurz und mild, die Kinder essen es.

## Zutaten

- 800 g Kartoffeln, festkochend
- 2 Zwiebeln
- 1 Dose Kokosmilch
- Salz

## Zubereitung

1. Kartoffeln schälen und würfeln.
2. Zwiebeln anschwitzen.
`

// makeRecipe creates a recipe page in space through c and gives it fields.
func (ts *testServer) makeRecipe(t *testing.T, c *http.Client, space, title, body string, servings int, tags ...string) string {
	t.Helper()
	id := ts.createPage(t, c, space, "", title)
	err := ts.deps.Wiki.SaveWith(store.Author{Name: "Test", Email: "test@example.org"}, id, title, body, func(m *wiki.PageMeta) {
		m.Type, m.Servings, m.Tags, m.Time = wiki.TypeRecipe, servings, tags, 40
	})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func mustDay(t *testing.T, day string) time.Time {
	t.Helper()
	d, err := time.Parse("2006-01-02", day)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
