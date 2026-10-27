// SPDX-License-Identifier: Apache-2.0

package store

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var safeExt = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)

// AttachmentName turns an uploaded file name into a safe, readable one.
func AttachmentName(upload string) string {
	base := upload[strings.LastIndexAny(upload, `/\`)+1:]
	base = strings.ReplaceAll(base, "\x00", "")
	ext := strings.ToLower(filepath.Ext(base))
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	if stem == "" || !safeExt.MatchString(ext) {
		stem, ext = base, ""
	}
	slug := Slugify(stem)
	if slug == "seite" && !strings.Contains(strings.ToLower(stem), "seite") {
		slug = "datei" // Slugify's fallback for names without letters or digits
	}
	return slug + ext
}

// UniqueFile returns name, or name-2, name-3 … if dir already has it.
func UniqueFile(dir, name string) string {
	ext := path.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	candidate := name
	for n := 2; fileExists(filepath.Join(dir, candidate)); n++ {
		candidate = stem + "-" + strconv.Itoa(n) + ext
	}
	return candidate
}

// fileExists reports whether p exists, without following links.
func fileExists(p string) bool {
	_, err := os.Lstat(p)
	return err == nil
}
