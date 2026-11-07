// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// words are the parts of a draft that depend on the language.
var words = map[string]struct{ ingredients, steps, source, private string }{
	"de": {"## Zutaten", "## Zubereitung", "Quelle", "Für den privaten Gebrauch übernommen."},
	"en": {"## Ingredients", "## Method", "Source", "Copied for private use."},
}

// Markdown writes the draft in Kanso's recipe format (ADR-010).
func (d Draft) Markdown(lang string) string {
	w, ok := words[lang]
	if !ok {
		w = words["de"]
	}
	var b strings.Builder
	if d.Description != "" {
		b.WriteString(d.Description + "\n\n")
	}
	b.WriteString(w.ingredients + "\n\n")
	for _, ing := range d.Ingredients {
		b.WriteString("- " + ing + "\n")
	}
	b.WriteString("\n" + w.steps + "\n\n")
	for i, s := range d.Steps {
		b.WriteString(strconv.Itoa(i+1) + ". " + s + "\n")
	}
	if d.Source != "" {
		b.WriteString("\n---\n\n" + w.source + ": [" + hostOf(d.Source) + "](" + d.Source + "). " + w.private + "\n")
	}
	return b.String()
}

func hostOf(raw string) string {
	if u, err := url.Parse(raw); err == nil && u.Hostname() != "" {
		return u.Hostname()
	}
	return raw
}

var (
	dropBlocks []*regexp.Regexp
	breaks     = regexp.MustCompile(`(?i)<br\s*/?>|</(p|li|h[1-6]|div|tr|section|article|ul|ol)>`)
)

func init() {
	for _, tag := range []string{"script", "style", "nav", "header", "footer", "noscript", "svg", "form", "aside", "template"} {
		dropBlocks = append(dropBlocks, regexp.MustCompile(`(?is)<`+tag+`\b.*?</`+tag+`>`))
	}
}

// Text is the readable text of a page, without scripts, styles and site
// furniture (navigation, header, footer), one block per line.
func Text(page string) string {
	for _, re := range dropBlocks {
		page = re.ReplaceAllString(page, " ")
	}
	page = html.UnescapeString(tags.ReplaceAllString(breaks.ReplaceAllString(page, "\n"), " "))
	var lines []string
	for _, line := range strings.Split(page, "\n") {
		if l := strings.Join(strings.Fields(line), " "); l != "" {
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}
