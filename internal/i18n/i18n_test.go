// SPDX-License-Identifier: Apache-2.0

package i18n

import (
	"slices"
	"testing"
)

func TestBundlesHaveSameKeys(t *testing.T) {
	b, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	de, en := b.keys("de"), b.keys("en")
	if len(de) == 0 {
		t.Fatal("no keys")
	}
	for _, k := range de {
		if !slices.Contains(en, k) {
			t.Errorf("en lacks %q", k)
		}
	}
	for _, k := range en {
		if !slices.Contains(de, k) {
			t.Errorf("de lacks %q", k)
		}
	}
}

func TestTranslate(t *testing.T) {
	b, _ := Load()
	if got := b.T("en", "app.name"); got != "Hüpattl! Kanso" {
		t.Errorf("app.name = %q", got)
	}
	if got := b.T("en", "no.such.key"); got != "no.such.key" {
		t.Errorf("missing key = %q", got)
	}
	if got := b.T("fr", "nav.spaces"); got != b.T("de", "nav.spaces") {
		t.Errorf("fallback = %q", got)
	}
	if got := b.T("de", "lock.held_by", "Anna"); got != "Anna bearbeitet diese Seite gerade." {
		t.Errorf("args = %q", got)
	}
}

func TestNegotiate(t *testing.T) {
	cases := []struct{ accept, user, want string }{
		{"en-US,en;q=0.9", "", "en"},
		{"fr", "", "de"},
		{"", "", "de"},
		{"de-DE,de;q=0.9,en;q=0.8", "", "de"},
		{"fr-FR,en;q=0.5", "", "en"},
		{"en", "de", "de"},
	}
	for _, c := range cases {
		if got := Negotiate(c.accept, c.user); got != c.want {
			t.Errorf("Negotiate(%q,%q) = %q", c.accept, c.user, got)
		}
	}
}
