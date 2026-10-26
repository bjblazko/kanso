// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/wiki"
)

const maxAttachment = 50 << 20 // 50 MiB

// Only these types are served as themselves; everything else downloads as bytes.
var attachmentTypes = map[string]string{
	".png": "image/png", ".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".gif": "image/gif",
	".webp": "image/webp", ".pdf": "application/pdf", ".txt": "text/plain; charset=utf-8",
	".csv": "text/csv; charset=utf-8", ".zip": "application/zip",
}

func isImage(ext string) bool {
	return strings.HasPrefix(attachmentTypes[ext], "image/")
}

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachment+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer file.Close()
	u, _ := auth.UserFrom(r.Context())
	name, err := s.Wiki.AddAttachment(u.Author(), pc.Page.Meta.ID, header.Filename, file, maxAttachment)
	if errors.Is(err, wiki.ErrTooLarge) {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"name": name, "markdown": attachmentMarkdown(header.Filename, name)})
}

// attachmentMarkdown is what the editor inserts for an uploaded file.
func attachmentMarkdown(original, name string) string {
	ext := path.Ext(name)
	target := "attachments/" + name
	if isImage(ext) {
		alt := strings.TrimSuffix(path.Base(strings.ReplaceAll(original, `\`, "/")), path.Ext(original))
		alt = strings.NewReplacer("[", "", "]", "", "\n", " ").Replace(alt)
		return "![" + alt + "](" + target + ")"
	}
	return "[" + name + "](" + target + ")"
}

func (s *Server) serveAttachment(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok {
		return
	}
	name := r.PathValue("name")
	f, err := s.Wiki.OpenAttachment(pc.Page.Meta.ID, name)
	if err != nil {
		s.notFound(w, r)
		return
	}
	defer f.Close()
	ext := path.Ext(name)
	ctype, known := attachmentTypes[ext]
	disposition := "attachment"
	if known && (isImage(ext) || ext == ".pdf") {
		disposition = "inline"
	}
	if !known {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Disposition", disposition+`; filename="`+name+`"`)
	w.Header().Set("Cache-Control", "private, no-cache")
	info, err := f.Stat()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.ServeContent(w, r, "", info.ModTime(), f)
}

func (s *Server) deleteAttachment(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	if err := s.Wiki.DeleteAttachment(u.Author(), pc.Page.Meta.ID, r.PathValue("name")); err != nil {
		s.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/p/"+pc.Page.Meta.ID, http.StatusSeeOther)
}
