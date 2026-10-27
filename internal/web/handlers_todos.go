// SPDX-License-Identifier: Apache-2.0

package web

import (
	"encoding/json"
	"net/http"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/todos"
)

const maxSyncRequest = 1 << 20

// listOut is a list as todos.js gets it, with the names it shows.
type listOut struct {
	todos.List
	Mine        bool     `json:"mine"`
	OwnerName   string   `json:"ownerName"`
	SharedNames []string `json:"sharedNames"`
}

// todoMessages are the texts todos.js shows; the page carries them as JSON.
var todoMessages = []string{"title", "new_list", "create", "empty", "offline_empty", "progress", "from",
	"shared_with", "new_item", "add", "indent", "outdent", "add_sub", "delete_item", "rename", "share",
	"delete_list", "all", "waiting_one", "waiting_many", "sync_login", "not_found", "name", "save", "cancel",
	"item", "edit_item"}

// todosPage serves the page for /todos and /todos/{id}; todos.js draws it
// from the copy on the device, so it also opens offline.
func (s *Server) todosPage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	msgs := map[string]string{}
	for _, k := range todoMessages {
		msgs[k] = v.T("todos." + k)
	}
	v.Title, v.Data = v.T("todos.title"), msgs
	s.render(w, r, http.StatusOK, "todos", v)
}

// todoLists returns what sub sees, with names resolved.
func (s *Server) todoLists(sub string, g access.Groups) []listOut {
	out := []listOut{}
	for _, l := range s.Todos.Visible(sub, g) {
		o := listOut{List: l, Mine: l.Owner == sub, OwnerName: s.userName(l.Owner), SharedNames: []string{}}
		for _, sh := range l.Shared {
			o.SharedNames = append(o.SharedNames, s.shareName(sh))
		}
		out = append(out, o)
	}
	return out
}

func (s *Server) userName(sub string) string {
	if u, ok := s.Users.Get(sub); ok && u.Name != "" {
		return u.Name
	}
	return sub
}

func (s *Server) shareName(sh todos.Share) string {
	if sh.Group != "" {
		return sh.Group
	}
	return s.userName(sh.User)
}

func (s *Server) apiTodos(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"lists": s.todoLists(u.Sub, s.groups())})
}

// syncTodos applies the operations a device collected and answers with the
// current state of every list the person sees.
func (s *Server) syncTodos(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSyncRequest)
	var req struct {
		Ops []todos.Op `json:"ops"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Ops) > todos.MaxOps {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid"})
		return
	}
	u, _ := auth.UserFrom(r.Context())
	g := s.groups()
	rejected := s.Todos.Apply(u.Sub, g, req.Ops)
	if rejected == nil {
		rejected = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"rejected": rejected, "lists": s.todoLists(u.Sub, g)})
}
