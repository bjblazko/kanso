// SPDX-License-Identifier: Apache-2.0

// Package i18n holds the interface texts in German and English.
package i18n

import (
	"embed"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Fallback is the language used when nothing else matches.
const Fallback = "de"

// Supported lists the interface languages.
var Supported = []string{"de", "en"}

//go:embed de.yaml en.yaml
var files embed.FS

// Bundle maps language → key → text. Texts with arguments use fmt verbs.
type Bundle struct {
	texts map[string]map[string]string
}

// Load reads the embedded language files.
func Load() (*Bundle, error) {
	b := &Bundle{texts: map[string]map[string]string{}}
	for _, lang := range Supported {
		raw, err := files.ReadFile(lang + ".yaml")
		if err != nil {
			return nil, err
		}
		m := map[string]string{}
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("i18n %s: %w", lang, err)
		}
		b.texts[lang] = m
	}
	return b, nil
}

// T returns the text for key, falling back to German, then to the key.
func (b *Bundle) T(lang, key string, args ...any) string {
	text, ok := b.texts[lang][key]
	if !ok {
		text, ok = b.texts[Fallback][key]
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(text, args...)
	}
	return text
}

func (b *Bundle) keys(lang string) []string {
	var ks []string
	for k := range b.texts[lang] {
		ks = append(ks, k)
	}
	return ks
}

// Negotiate picks the user's choice, else the first supported language in
// the Accept-Language header, else the fallback.
func Negotiate(acceptLanguage, userLang string) string {
	if slices.Contains(Supported, userLang) {
		return userLang
	}
	for _, part := range strings.Split(acceptLanguage, ",") {
		tag, _, _ := strings.Cut(strings.TrimSpace(part), ";")
		primary, _, _ := strings.Cut(strings.ToLower(tag), "-")
		if slices.Contains(Supported, primary) {
			return primary
		}
	}
	return Fallback
}
