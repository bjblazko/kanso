// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/notes"
)

type noteEdit struct {
	Note    notes.Note
	Title   string
	Version string // Updated when the editor opened, sent back on save
	Error   string
}

func (s *Server) editNote(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	s.renderNoteEditor(w, r, http.StatusOK, noteEdit{Note: n, Version: n.Meta.Updated.Format(time.RFC3339Nano)})
}

func (s *Server) renderNoteEditor(w http.ResponseWriter, r *http.Request, status int, data noteEdit) {
	v := s.newView(r, "", nil)
	data.Title = noteTitle(v, data.Note)
	v.Title, v.Data, v.Here = data.Title, data, notesHere
	s.render(w, r, status, "note_edit", v)
}

// saveNote stores the editor's text unless the note changed since the editor
// opened (another device); then the text stays in the editor.
func (s *Server) saveNote(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	body := strings.ReplaceAll(r.PostFormValue("body"), "\r\n", "\n")
	version, _ := time.Parse(time.RFC3339Nano, r.PostFormValue("version"))
	_, err := s.Notes.Save(n.Owner, n.Meta.ID, body, version)
	switch {
	case errors.Is(err, notes.ErrConflict):
		s.noteConflict(w, r, n, body)
	case err != nil:
		s.serverError(w, r, err)
	case wantsJSON(r):
		writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
	default:
		http.Redirect(w, r, "/notes/"+n.Meta.ID, http.StatusSeeOther)
	}
}

func (s *Server) noteConflict(w http.ResponseWriter, r *http.Request, n notes.Note, body string) {
	if wantsJSON(r) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "conflict"})
		return
	}
	v := s.newView(r, "", nil)
	edited := n
	edited.Body = body
	s.renderNoteEditor(w, r, http.StatusConflict, noteEdit{Note: edited,
		Version: n.Meta.Updated.Format(time.RFC3339Nano), Error: v.T("notes.conflict")})
}

func (s *Server) uploadNoteAttachment(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachment+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	name, err := s.Notes.AddAttachment(n.Owner, n.Meta.ID, header.Filename, file, maxAttachment)
	if errors.Is(err, notes.ErrTooLarge) {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"name": name, "markdown": attachmentMarkdown(header.Filename, name)})
}

// notePreview renders Markdown for the note editor of the note's owner.
func (s *Server) notePreview(w http.ResponseWriter, r *http.Request) {
	r.SetPathValue("id", r.PostFormValue("note"))
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	s.writePreview(w, r.PostFormValue("body"), markdown.PageCtx{Base: "/notes/" + n.Meta.ID})
}

// writePreview renders body as an HTML fragment.
func (s *Server) writePreview(w http.ResponseWriter, body string, pc markdown.PageCtx) {
	html, _, err := s.MD.Render([]byte(body), pc)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(html))
}
