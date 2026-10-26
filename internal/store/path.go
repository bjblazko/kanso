// SPDX-License-Identifier: Apache-2.0

// Package store keeps Kanso's content as files in a Git repository.
package store

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrOutside means a path would leave its root directory.
var ErrOutside = errors.New("path outside root")

const maxSlugLen = 80

// Join joins parts below root and refuses anything that would leave it.
func Join(root string, parts ...string) (string, error) {
	for _, p := range parts {
		if strings.ContainsRune(p, 0) || filepath.IsAbs(p) {
			return "", ErrOutside
		}
	}
	full := filepath.Join(append([]string{root}, parts...)...)
	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", ErrOutside
	}
	return full, nil
}

var umlauts = strings.NewReplacer("ä", "ae", "ö", "oe", "ü", "ue", "ß", "ss")

// Slugify turns a title into a readable, file-system-safe name.
func Slugify(title string) string {
	s := umlauts.Replace(strings.ToLower(title))
	var b strings.Builder
	dash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			dash = false
		} else if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > maxSlugLen {
		slug = strings.Trim(slug[:maxSlugLen], "-")
	}
	if slug == "" {
		return "seite"
	}
	return slug
}

// UniqueSlug returns slug, or slug-2, slug-3 … if dir already holds it.
func UniqueSlug(dir, slug string) string {
	candidate := slug
	for n := 2; exists(filepath.Join(dir, candidate)); n++ {
		candidate = slug + "-" + strconv.Itoa(n)
	}
	return candidate
}

func exists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}

var idEncoding = base32.StdEncoding.WithPadding(base32.NoPadding)

// NewID returns a random 12-character lower-case id.
func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return strings.ToLower(idEncoding.EncodeToString(b))[:12]
}
