// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/notes"
)

const (
	notesHere       = "/notes" // view.Here for the notes area; no space slug has a slash
	maxNoteRequest  = 200 << 20
	multipartMemory = 32 << 20
)

type noteItem struct {
	ID, Title string
	Created   time.Time
	Files     int
}

type noteView struct {
	// TaskID names the task made from the note while it still exists; "" otherwise.
	TaskID string
	Note   notes.Note
	Title  string
	HTML   template.HTML
	Info   markdown.Info
}

type noteNew struct {
	Saved bool
	Error string
	Body  string
}

// noteTitle is the note's first line, or a name made from its date.
func noteTitle(v view, n notes.Note) string {
	if t := notes.Title(n); t != "" {
		return t
	}
	if len(n.Attachments) > 0 {
		return v.T("notes.photo_title", n.Meta.Created.Local().Format("02.01.2006, 15:04"))
	}
	return v.T("notes.untitled")
}

func (s *Server) listNotes(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	v := s.newView(r, "", nil)
	var items []noteItem
	for _, n := range s.Notes.List(u.Sub) {
		items = append(items, noteItem{ID: n.Meta.ID, Title: noteTitle(v, n), Created: n.Meta.Created.Local(), Files: len(n.Attachments)})
	}
	v.Title, v.Data, v.Here = v.T("notes.title"), items, notesHere
	v.Audience = ptr(onlyYou(v))
	s.render(w, r, http.StatusOK, "notes", v)
}

func (s *Server) newNoteForm(w http.ResponseWriter, r *http.Request) {
	s.renderCapture(w, r, http.StatusOK, noteNew{Saved: r.URL.Query().Get("saved") == "1"})
}

func (s *Server) renderCapture(w http.ResponseWriter, r *http.Request, status int, data noteNew) {
	v := s.newView(r, "", data)
	v.Title, v.Here, v.Audience = v.T("notes.new"), notesHere, ptr(onlyYou(v))
	s.render(w, r, status, "note_new", v)
}

// noteFor loads the signed-in person's note {id}; anyone else gets 404.
func (s *Server) noteFor(w http.ResponseWriter, r *http.Request) (notes.Note, bool) {
	u, _ := auth.UserFrom(r.Context())
	n, err := s.Notes.Get(u.Sub, r.PathValue("id"))
	if err != nil {
		s.notFound(w, r)
		return notes.Note{}, false
	}
	return n, true
}

func (s *Server) showNote(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	html, info, err := s.MD.Render([]byte(n.Body), markdown.PageCtx{Base: "/notes/" + n.Meta.ID})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	v := s.newView(r, "", nil)
	v.Data = noteView{Note: n, Title: noteTitle(v, n), HTML: html, Info: info, TaskID: s.noteTaskID(r, n)}
	v.Title, v.Here, v.Audience = noteTitle(v, n), notesHere, ptr(onlyYou(v))
	s.render(w, r, http.StatusOK, "note_view", v)
}

func (s *Server) serveNoteAttachment(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	name := r.PathValue("name")
	f, err := s.Notes.OpenAttachment(u.Sub, r.PathValue("id"), name)
	if err != nil {
		s.notFound(w, r)
		return
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	setAttachmentHeaders(w.Header(), name)
	http.ServeContent(w, r, "", info.ModTime(), f)
}

func (s *Server) confirmDeleteNote(w http.ResponseWriter, r *http.Request) {
	n, ok := s.noteFor(w, r)
	if !ok {
		return
	}
	v := s.newView(r, "", nil)
	v.Data = noteView{Note: n, Title: noteTitle(v, n)}
	v.Title, v.Here = v.T("notes.delete"), notesHere
	s.render(w, r, http.StatusOK, "note_delete", v)
}

func (s *Server) deleteNote(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := s.Notes.Delete(u.Sub, r.PathValue("id")); err != nil {
		s.notFound(w, r)
		return
	}
	http.Redirect(w, r, "/notes", http.StatusSeeOther)
}

// createNote takes the capture form: text and photos, optionally with an id
// made on the device so a repeated upload is recognised.
func (s *Server) createNote(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxNoteRequest)
	if err := r.ParseMultipartForm(multipartMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		s.noteFailed(w, r, http.StatusRequestEntityTooLarge, "too_large", "")
		return
	}
	body := strings.ReplaceAll(r.FormValue("body"), "\r\n", "\n")
	files := uploadedFiles(r)
	if strings.TrimSpace(body) == "" && len(files) == 0 {
		s.noteFailed(w, r, http.StatusBadRequest, "empty", body)
		return
	}
	uploads, closeAll, err := openUploads(files)
	defer closeAll()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	n, created, err := s.Notes.Create(u.Sub, r.FormValue("id"), body, uploads, maxAttachment)
	switch {
	case errors.Is(err, notes.ErrBadID):
		s.noteFailed(w, r, http.StatusBadRequest, "invalid", body)
	case errors.Is(err, notes.ErrTooLarge):
		s.noteFailed(w, r, http.StatusRequestEntityTooLarge, "too_large", body)
	case err != nil:
		s.serverError(w, r, err)
	default:
		s.noteCreated(w, r, n, created)
	}
}

func (s *Server) noteCreated(w http.ResponseWriter, r *http.Request, n notes.Note, created bool) {
	if !wantsJSON(r) {
		http.Redirect(w, r, "/notes/new?saved=1", http.StatusSeeOther)
		return
	}
	if created {
		writeJSON(w, http.StatusCreated, map[string]string{"id": n.Meta.ID, "status": "created"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"id": n.Meta.ID, "status": "exists"})
}

func (s *Server) noteFailed(w http.ResponseWriter, r *http.Request, status int, code, body string) {
	if wantsJSON(r) {
		writeJSON(w, status, map[string]string{"error": code})
		return
	}
	v := s.newView(r, "", nil)
	s.renderCapture(w, r, status, noteNew{Error: v.T("notes.error_" + code), Body: body})
}

// uploadedFiles skips the empty part a browser sends for an unused file field.
func uploadedFiles(r *http.Request) []*multipart.FileHeader {
	if r.MultipartForm == nil {
		return nil
	}
	var out []*multipart.FileHeader
	for _, fh := range r.MultipartForm.File["files"] {
		if fh.Filename != "" || fh.Size > 0 {
			out = append(out, fh)
		}
	}
	return out
}

func openUploads(files []*multipart.FileHeader) ([]notes.Upload, func(), error) {
	var opened []io.Closer
	closeAll := func() {
		for _, c := range opened {
			_ = c.Close()
		}
	}
	var uploads []notes.Upload
	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			return nil, closeAll, err
		}
		opened = append(opened, f)
		uploads = append(uploads, notes.Upload{Name: fh.Filename, R: f})
	}
	return uploads, closeAll, nil
}
