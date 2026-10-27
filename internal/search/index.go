// SPDX-License-Identifier: Apache-2.0

// Package search finds pages by words in their title and text. The index
// lives in memory; a household's wiki is small enough to scan.
package search

import (
	"cmp"
	"slices"
	"strings"
	"sync"
	"unicode"
)

const (
	titleWeight = 3
	snippetLen  = 160
)

// Doc is one searchable page (Space set) or note (Owner set).
type Doc struct {
	ID, Space, Owner, Title, Body string
}

// Hit is one search result.
type Hit struct {
	ID, Space, Owner, Title, Snippet string
	Score                            int
}

type entry struct {
	doc   Doc
	title map[string]int
	body  map[string]int
	words []string // body words in their original spelling, for snippets
}

// Index holds all documents.
type Index struct {
	mu   sync.RWMutex
	docs map[string]entry
}

// NewIndex creates an empty index.
func NewIndex() *Index { return &Index{docs: map[string]entry{}} }

// Put adds or replaces a document.
func (ix *Index) Put(d Doc) {
	words := strings.Fields(plain(d.Body))
	e := entry{doc: d, title: counts(fold(d.Title)), body: counts(fold(strings.Join(words, " "))), words: words}
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.docs[d.ID] = e
}

// Remove drops a document.
func (ix *Index) Remove(id string) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	delete(ix.docs, id)
}

// Search returns documents containing every word of q (the last one as a
// prefix) that allowed reports true for, best first.
func (ix *Index) Search(q string, allowed func(d Doc) bool, limit int) []Hit {
	terms := fold(q)
	if len(terms) == 0 {
		return nil
	}
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var hits []Hit
	for _, e := range ix.docs {
		if score := e.score(terms); score > 0 && allowed(e.doc) {
			hits = append(hits, Hit{ID: e.doc.ID, Space: e.doc.Space, Owner: e.doc.Owner, Title: e.doc.Title,
				Snippet: snippet(e.words, terms), Score: score})
		}
	}
	slices.SortFunc(hits, func(a, b Hit) int { return cmp.Or(b.Score-a.Score, strings.Compare(a.Title, b.Title)) })
	return hits[:min(limit, len(hits))]
}

// score is 0 unless every term matches.
func (e entry) score(terms []string) int {
	total := 0
	for i, t := range terms {
		prefix := i == len(terms)-1
		s := titleWeight*matches(e.title, t, prefix) + matches(e.body, t, prefix)
		if s == 0 {
			return 0
		}
		total += s
	}
	return total
}

func matches(words map[string]int, term string, prefix bool) int {
	if !prefix {
		return words[term]
	}
	n := 0
	for w, c := range words {
		if strings.HasPrefix(w, term) {
			n += c
		}
	}
	return n
}

var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")

// fold lower-cases and spells umlauts out, so "Größe" and "Groesse" meet.
func fold(s string) []string {
	return strings.FieldsFunc(umlauts.Replace(strings.ToLower(s)), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

func counts(words []string) map[string]int {
	m := map[string]int{}
	for _, w := range words {
		m[w]++
	}
	return m
}

var markup = strings.NewReplacer("*", " ", "_", " ", "#", " ", ">", " ", "`", " ", "[", " ", "]", " ",
	"(", " ", ")", " ", "|", " ", "~", " ")

// plain removes the most common Markdown markup characters.
func plain(md string) string { return markup.Replace(md) }

// snippet shows the words around the first match, about snippetLen runes.
func snippet(words []string, terms []string) string {
	at := 0
	for i, w := range words {
		if f := fold(w); len(f) > 0 && strings.HasPrefix(f[0], terms[0]) {
			at = i
			break
		}
	}
	start := max(0, at-8)
	var b strings.Builder
	for _, w := range words[start:] {
		if b.Len() > 0 && len([]rune(b.String()))+len([]rune(w))+1 > snippetLen {
			break
		}
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(w)
	}
	return b.String()
}
