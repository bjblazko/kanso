// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/photo"
)

// ErrTooLarge means an upload exceeded the size limit.
var ErrTooLarge = errors.New("upload too large")

var (
	safeExt  = regexp.MustCompile(`^\.[a-z0-9]{1,8}$`)
	safeName = regexp.MustCompile(`^[a-z0-9-]+(\.[a-z0-9]{1,8})?$`)
)

// ValidAttachmentName reports whether name is one AttachmentName can produce.
func ValidAttachmentName(name string) bool { return safeName.MatchString(name) }

// SaveUpload stores r (at most max bytes) in dir under a safe, unused name
// derived from filename, with the location removed from JPEG photos.
func SaveUpload(dir, filename string, r io.Reader, max int64) (string, error) {
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", err
	}
	name := UniqueFile(dir, AttachmentName(filename))
	clean, err := photo.Clean(name, r, max)
	if errors.Is(err, photo.ErrTooLarge) {
		return "", ErrTooLarge
	} else if err != nil {
		return "", err
	}
	return name, writeLimited(filepath.Join(dir, name), clean, max)
}

// writeLimited writes at most max bytes of r to the new file dst.
func writeLimited(dst string, r io.Reader, max int64) error {
	f, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	n, err := io.Copy(f, io.LimitReader(r, max+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > max {
		err = ErrTooLarge
	}
	if err != nil {
		_ = os.Remove(dst)
	}
	return err
}

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
