// SPDX-License-Identifier: Apache-2.0

package web

import (
	"errors"
	"net/http"

	"github.com/bjblazko/kanso/internal/notes"
	"github.com/bjblazko/kanso/internal/tasks"
)

// uploadTaskAttachment stores a file next to a task and answers with the
// Markdown that shows it, like the wiki and notes editors expect.
func (s *Server) uploadTaskAttachment(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAttachment+1<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	defer func() { _ = file.Close() }()
	name, err := s.Tasks.AddAttachment(s.actor(r), r.PathValue("board"), r.PathValue("task"), header.Filename, file, maxAttachment)
	switch {
	case errors.Is(err, tasks.ErrNotFound):
		s.notFound(w, r)
	case errors.Is(err, tasks.ErrTooLarge):
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
	case err != nil:
		s.serverError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, map[string]string{"name": name, "markdown": attachmentMarkdown(header.Filename, name)})
	}
}

// serveTaskAttachment delivers a task's file to whoever sees the board.
func (s *Server) serveTaskAttachment(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	f, err := s.Tasks.OpenAttachment(s.actor(r), r.PathValue("board"), r.PathValue("task"), name)
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

// copyNotePhotos gives a task made from a note copies of the note's files,
// under the same names, so the photo lines in its text keep working.
func (s *Server) copyNotePhotos(r *http.Request, n notes.Note, t tasks.Task) error {
	var files []tasks.File
	for _, name := range n.Attachments {
		f, err := s.Notes.OpenAttachment(n.Owner, n.Meta.ID, name)
		if err != nil {
			return err
		}
		defer func() { _ = f.Close() }()
		files = append(files, tasks.File{Name: name, R: f})
	}
	if len(files) == 0 {
		return nil
	}
	_, err := s.Tasks.AddAttachments(s.actor(r), t.Board, t.Meta.ID, files, maxAttachment)
	return err
}
