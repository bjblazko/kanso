// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"regexp"
	"strconv"
	"strings"
)

const (
	maxPhotos = 8
	minSide   = 200 // pixels; smaller images are icons, avatars or thumbnails
)

var (
	imgTag   = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	imgAttr  = regexp.MustCompile(`(?is)\b(src|data-src|width|height)\s*=\s*("[^"]*"|'[^']*'|[^\s>]+)`)
	cdnSize  = regexp.MustCompile(`/w_(\d+),h_(\d+)`)
	stepHead = map[string]bool{"zubereitung": true, "schritte": true, "steps": true, "method": true}
	listItem = regexp.MustCompile(`^(\d+[.)]|[-*+])\s+`)
)

// PagePhotos lists the photos in a page's content, not counting main: no
// icons or thumbnails (under 200 pixels), no SVG, each picture once.
func PagePhotos(page, base, main string) []string {
	seen := map[string]bool{photoKey(main): true}
	var out []string
	for _, tag := range imgTag.FindAllString(page, -1) {
		u := photoOf(tag, base)
		if u == "" || seen[photoKey(u)] || len(out) >= maxPhotos {
			continue
		}
		seen[photoKey(u)] = true
		out = append(out, u)
	}
	return out
}

// photoOf is the address of an <img> if it shows a photo of a useful size.
func photoOf(tag, base string) string {
	attrs := map[string]string{}
	for _, m := range imgAttr.FindAllStringSubmatch(tag, -1) {
		attrs[strings.ToLower(m[1])] = strings.Trim(m[2], `"'`)
	}
	src := attrs["data-src"] // lazy loading puts the real address here
	if src == "" || strings.HasPrefix(src, "data:") {
		src = attrs["src"]
	}
	u := resolve(base, src)
	if u == "" || isIcon(u, attrs) {
		return ""
	}
	return u
}

// isIcon tells an SVG or a picture too small to be a photo of the dish.
func isIcon(u string, attrs map[string]string) bool {
	if strings.HasSuffix(strings.ToLower(strings.SplitN(u, "?", 2)[0]), ".svg") || small(attrs["width"]) || small(attrs["height"]) {
		return true
	}
	m := cdnSize.FindStringSubmatch(u)
	return m != nil && (small(m[1]) || small(m[2]))
}

func small(v string) bool {
	n, err := strconv.Atoi(strings.TrimSuffix(v, "px"))
	return err == nil && n < minSide
}

// photoKey names a picture regardless of the size a CDN serves it in.
func photoKey(u string) string {
	if i := strings.Index(u, "/v1/"); i > 0 {
		return u[:i]
	}
	return strings.SplitN(u, "?", 2)[0]
}

// Placement puts a photo's Markdown at a place: "top", "step-<n>" or "end".
type Placement struct{ Place, Markdown string }

// InsertPhotos puts each photo where the person chose; a step that does not
// exist means the end.
func InsertPhotos(body string, ps []Placement) string {
	lines := strings.Split(strings.TrimRight(body, "\n"), "\n")
	var top []string
	for _, p := range ps {
		switch {
		case p.Place == "top":
			top = append(top, p.Markdown, "")
		case strings.HasPrefix(p.Place, "step-"):
			n, _ := strconv.Atoi(strings.TrimPrefix(p.Place, "step-"))
			lines = afterStep(lines, n, p.Markdown)
		default:
			lines = atEnd(lines, p.Markdown)
		}
	}
	return strings.Join(append(top, lines...), "\n") + "\n"
}

// afterStep puts md, indented, below step n of the method section.
func afterStep(lines []string, n int, md string) []string {
	in, count := false, 0
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "## ") {
			in = stepHead[strings.ToLower(strings.Trim(strings.Fields(t[3:] + " x")[0], ":()."))]
			continue
		}
		if in && listItem.MatchString(t) {
			if count++; count == n {
				end := i + 1
				for end < len(lines) && strings.TrimSpace(lines[end]) != "" && !listItem.MatchString(strings.TrimSpace(lines[end])) && !strings.HasPrefix(strings.TrimSpace(lines[end]), "#") {
					end++
				}
				return insert(lines, end, "", "   "+md, "")
			}
		}
	}
	return atEnd(lines, md)
}

// atEnd puts md before the closing source block ("---"), or last.
func atEnd(lines []string, md string) []string {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "---" {
			return insert(lines, i, md, "")
		}
	}
	return append(lines, "", md)
}

func insert(lines []string, at int, add ...string) []string {
	out := append([]string{}, lines[:at]...)
	out = append(out, add...)
	return append(out, lines[at:]...)
}
