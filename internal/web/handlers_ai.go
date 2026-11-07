// SPDX-License-Identifier: Apache-2.0

package web

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/bjblazko/kanso/internal/ai"
)

type aiAdminPage struct {
	Config   ai.Config
	HasToken bool
	Result   string // outcome of a connection test
	Failed   bool
	Target   *aiTarget
}

func (s *Server) aiAdmin(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) {
		s.renderAIAdmin(w, r, http.StatusOK, aiAdminPage{})
	}
}

func (s *Server) renderAIAdmin(w http.ResponseWriter, r *http.Request, status int, page aiAdminPage) {
	page.Config = s.AI.Load()
	page.HasToken, page.Config.Token = page.Config.Token != "", "" // the token never reaches a page
	page.Target = s.aiTarget(r.Context())
	v := s.newView(r, "", page)
	v.Title = v.T("ai.title")
	s.render(w, r, status, "admin_ai", v)
}

// updateAI saves the connection and, for "test", asks the model for a short answer.
func (s *Server) updateAI(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	cfg := s.AI.Load()
	oldHost := cfg.Host()
	cfg.Enabled = r.PostFormValue("enabled") != ""
	cfg.BaseURL, cfg.Model = strings.TrimSpace(r.PostFormValue("base_url")), r.PostFormValue("model")
	if cfg.Host() != oldHost {
		cfg.Token = "" // a token never goes along to a host it was not entered for
	}
	if t := strings.TrimSpace(r.PostFormValue("token")); t != "" {
		cfg.Token = t
	}
	if r.PostFormValue("remove_token") != "" {
		cfg.Token = ""
	}
	if err := s.AI.Save(cfg); err != nil {
		s.serverError(w, r, err)
		return
	}
	if r.PostFormValue("action") != "test" {
		http.Redirect(w, r, "/admin/ai", http.StatusSeeOther)
		return
	}
	v := s.newView(r, "", nil)
	msg, failed := v.T("ai.test_ok"), false
	if err := s.testAI(r.Context(), s.AI.Load()); err != nil {
		msg, failed = v.T(aiErrorKey(err)), true
	}
	s.renderAIAdmin(w, r, http.StatusOK, aiAdminPage{Result: msg, Failed: failed})
}

func (s *Server) testAI(ctx context.Context, cfg ai.Config) error {
	if !cfg.Ready() {
		return errAINotReady
	}
	ctx, cancel := context.WithTimeout(ctx, ai.Timeout)
	defer cancel()
	_, err := ai.Client{}.Complete(ctx, cfg, []ai.Message{{Role: "user", Content: "Antworte nur mit: ok"}})
	return err
}

var errAINotReady = errors.New("ai not ready")

// aiErrorKey names the text for an AI error.
func aiErrorKey(err error) string {
	switch {
	case errors.Is(err, errAINotReady):
		return "ai.not_ready"
	case errors.Is(err, ai.ErrAuth):
		return "ai.err_auth"
	case errors.Is(err, ai.ErrModel):
		return "ai.err_model"
	case errors.Is(err, ai.ErrTimeout):
		return "ai.err_timeout"
	case errors.Is(err, ai.ErrBadAnswer):
		return "ai.err_answer"
	}
	return "ai.err_unreachable"
}
