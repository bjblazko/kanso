// SPDX-License-Identifier: Apache-2.0

package web

import "net/http"

// shellRoutes registers the start page, search, licenses and the person's own settings.
func (s *Server) shellRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.home)
	mux.HandleFunc("GET /about/licenses", s.showLicenses)
	mux.HandleFunc("POST /me/lang", s.setLang)
	mux.HandleFunc("GET /avatar/{sub}", s.avatar)
	mux.HandleFunc("GET /search", s.search)
}

// wikiRoutes registers spaces and their pages.
func (s *Server) wikiRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /s/{space}", s.showSpace)
	mux.HandleFunc("GET /s/{space}/settings", s.spaceSettings)
	mux.HandleFunc("POST /s/{space}/settings", s.updateSpaceSettings)
	mux.HandleFunc("GET /s/{space}/new", s.newPageForm)
	mux.HandleFunc("POST /s/{space}/new", s.createPage)
	mux.HandleFunc("GET /p/{id}", s.showPage)
	mux.HandleFunc("POST /api/preview", s.preview)
	mux.HandleFunc("GET /p/{id}/edit", s.editPage)
	mux.HandleFunc("POST /p/{id}", s.savePage)
	mux.HandleFunc("POST /p/{id}/lock/heartbeat", s.heartbeat)
	mux.HandleFunc("POST /p/{id}/lock/break", s.breakLock)
	mux.HandleFunc("GET /p/{id}/history", s.showHistory)
	mux.HandleFunc("GET /p/{id}/history/{hash}", s.showRevision)
	mux.HandleFunc("POST /p/{id}/history/{hash}/restore", s.restoreRevision)
	mux.HandleFunc("POST /p/{id}/attachments", s.uploadAttachment)
	mux.HandleFunc("GET /p/{id}/attachments/{name}", s.serveAttachment)
	mux.HandleFunc("POST /p/{id}/attachments/{name}/delete", s.deleteAttachment)
	mux.HandleFunc("POST /p/{id}/move", s.movePage)
	mux.HandleFunc("GET /p/{id}/share", s.pageShare)
	mux.HandleFunc("POST /p/{id}/share", s.updatePageShare)
	mux.HandleFunc("GET /p/{id}/delete", s.confirmDeletePage)
	mux.HandleFunc("POST /p/{id}/delete", s.deletePage)
}

// notesRoutes registers private notes.
func (s *Server) notesRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /notes", s.listNotes)
	mux.HandleFunc("GET /notes/new", s.newNoteForm)
	mux.HandleFunc("GET /notes/token", s.noteToken)
	mux.HandleFunc("POST /notes", s.createNote)
	mux.HandleFunc("GET /notes/{id}", s.showNote)
	mux.HandleFunc("GET /notes/{id}/edit", s.editNote)
	mux.HandleFunc("POST /notes/{id}", s.saveNote)
	mux.HandleFunc("POST /notes/{id}/attachments", s.uploadNoteAttachment)
	mux.HandleFunc("GET /notes/{id}/attachments/{name}", s.serveNoteAttachment)
	mux.HandleFunc("GET /notes/{id}/promote", s.promoteNoteForm)
	mux.HandleFunc("POST /notes/{id}/promote", s.promoteNote)
	mux.HandleFunc("GET /notes/{id}/task", s.noteTaskForm)
	mux.HandleFunc("POST /notes/{id}/task", s.noteToTask)
	mux.HandleFunc("GET /notes/{id}/delete", s.confirmDeleteNote)
	mux.HandleFunc("POST /notes/{id}/delete", s.deleteNote)
}

// todoRoutes registers to-do lists.
func (s *Server) todoRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /todos", s.todosPage)
	mux.HandleFunc("GET /todos/{id}", s.todosPage)
	mux.HandleFunc("GET /todos/{id}/share", s.shareTodoForm)
	mux.HandleFunc("POST /todos/{id}/share", s.shareTodo)
	mux.HandleFunc("GET /todos/{id}/delete", s.confirmDeleteTodo)
	mux.HandleFunc("POST /todos/{id}/delete", s.deleteTodo)
	mux.HandleFunc("GET /api/todos", s.apiTodos)
	mux.HandleFunc("POST /api/todos/sync", s.syncTodos)
}

// taskRoutes registers boards and tasks.
func (s *Server) taskRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /tasks", s.tasksHome)
	mux.HandleFunc("GET /tasks/new", s.newTaskForm)
	mux.HandleFunc("POST /tasks", s.createTask)
	mux.HandleFunc("GET /tasks/b/{board}", s.showBoard)
	mux.HandleFunc("GET /tasks/t/{task}", s.findTaskByID)
	mux.HandleFunc("GET /tasks/b/{board}/{task}", s.showTask)
	mux.HandleFunc("GET /tasks/b/{board}/{task}/edit", s.editTask)
	mux.HandleFunc("POST /tasks/b/{board}/{task}", s.saveTask)
	mux.HandleFunc("POST /tasks/b/{board}/{task}/claim", s.claimTask)
	mux.HandleFunc("POST /tasks/b/{board}/{task}/move", s.moveTask)
	mux.HandleFunc("GET /tasks/b/{board}/{task}/delete", s.confirmDeleteTask)
	mux.HandleFunc("POST /tasks/b/{board}/{task}/delete", s.deleteTask)
}

// adminRoutes registers administration (admins only, checked in the handlers).
func (s *Server) adminRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /admin/spaces/new", s.newSpaceForm)
	mux.HandleFunc("POST /admin/spaces/new", s.createSpace)
	mux.HandleFunc("GET /admin/groups", s.groupsPage)
	mux.HandleFunc("POST /admin/groups", s.updateGroups)
	mux.HandleFunc("GET /admin/boards", s.boardsPage)
	mux.HandleFunc("POST /admin/boards", s.updateBoards)
}
