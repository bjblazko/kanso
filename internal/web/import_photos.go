// SPDX-License-Identifier: Apache-2.0

package web

import (
	"bytes"
	"context"
	"encoding/base64"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/photo"
	"github.com/bjblazko/kanso/internal/recipeimport"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/wiki"
)

const (
	maxPreviewPhoto = 2 << 20 // larger photos are not offered: the preview carries them inline
	maxImportPhotos = 12
	maxPhotoBytes   = 8 << 20
)

// previewPhoto is a photo offered in the import preview, with where it would go.
type previewPhoto struct {
	URL   string
	Data  template.URL // data: address, so the browser loads nothing from elsewhere
	Place string
}

// places are the positions a photo can take in a recipe with n steps.
func places(n int) []string {
	out := []string{"top"}
	for i := 1; i <= n; i++ {
		out = append(out, "step-"+strconv.Itoa(i))
	}
	return append(out, "end")
}

// previewPhotos fetches the photos a page offers: the dish (on top), each
// step's photo (below its step), and the page's other photos (not taken).
func (s *Server) previewPhotos(ctx context.Context, d recipeimport.Draft, page recipeimport.Page) []previewPhoto {
	type cand struct{ url, place string }
	main := d.Image
	if main == "" {
		main = recipeimport.PageImage(page.HTML, page.URL)
	}
	var cands []cand
	seen := map[string]bool{}
	add := func(u, place string) {
		if u != "" && !seen[u] && len(cands) < maxImportPhotos {
			seen[u] = true
			cands = append(cands, cand{u, place})
		}
	}
	add(main, "top")
	for i, u := range d.StepImages {
		add(u, "step-"+strconv.Itoa(i+1))
	}
	for _, u := range recipeimport.PagePhotos(page.HTML, page.URL, main) {
		add(u, "skip")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	var out []previewPhoto
	for _, c := range cands {
		img, err := s.Fetch.FetchImage(ctx, c.url)
		if err != nil || len(img.Data) > maxPreviewPhoto {
			continue
		}
		data := "data:" + img.Type + ";base64," + base64.StdEncoding.EncodeToString(img.Data)
		out = append(out, previewPhoto{URL: c.url, Data: template.URL(data), Place: c.place}) // #nosec G203 -- type checked, base64
	}
	return out
}

// attachPhotos fetches the photos the person placed, stores them as
// attachments (JPEGs without GPS) and says where each goes; image is the
// photo for the kitchen: the one on top, else the first.
func (s *Server) attachPhotos(r *http.Request, a store.Author, page wiki.PageMeta) (placed []recipeimport.Placement, image string) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	onTop := false
	for i := 0; i < maxImportPhotos; i++ {
		place := r.PostFormValue("place_" + strconv.Itoa(i))
		if !validPlace(place) {
			continue
		}
		name := s.storePhoto(ctx, a, page.ID, r.PostFormValue("photo_url_"+strconv.Itoa(i)), len(placed)+1)
		if name == "" {
			continue
		}
		alt := strings.NewReplacer("[", "", "]", "", "\n", " ").Replace(page.Title)
		placed = append(placed, recipeimport.Placement{Place: place, Markdown: "![" + alt + "](attachments/" + name + ")"})
		if image == "" || place == "top" && !onTop {
			image, onTop = name, place == "top"
		}
	}
	return placed, image
}

func validPlace(p string) bool {
	if p == "top" || p == "end" {
		return true
	}
	n, err := strconv.Atoi(strings.TrimPrefix(p, "step-"))
	return strings.HasPrefix(p, "step-") && err == nil && n > 0
}

// storePhoto fetches one photo and attaches it as foto-<n>; "" when that fails.
func (s *Server) storePhoto(ctx context.Context, a store.Author, pageID, url string, n int) string {
	img, err := s.Fetch.FetchImage(ctx, url)
	if err != nil {
		return ""
	}
	file := "foto-" + strconv.Itoa(n) + img.Ext
	clean, err := photo.Clean(file, bytes.NewReader(img.Data), maxPhotoBytes)
	if err != nil {
		return ""
	}
	name, err := s.Wiki.AddAttachment(a, pageID, file, clean, maxPhotoBytes)
	if err != nil {
		return ""
	}
	return name
}
