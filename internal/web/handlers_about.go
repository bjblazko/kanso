// SPDX-License-Identifier: Apache-2.0

package web

import (
	"io/fs"
	"net/http"

	"github.com/bjblazko/kanso/internal/auth"
	"go.yaml.in/yaml/v3"
)

type license struct {
	Name    string `yaml:"name"`
	Module  string `yaml:"module"`
	Version string `yaml:"version"`
	License string `yaml:"license"`
	URL     string `yaml:"url"`
	File    string `yaml:"file"`
}

func loadLicenses(static fs.FS) ([]license, error) {
	b, err := fs.ReadFile(static, "licenses/index.yaml")
	if err != nil {
		return nil, err
	}
	var list []license
	return list, yaml.Unmarshal(b, &list)
}

func (s *Server) showLicenses(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", s.licenses)
	v.Title = v.T("licenses.title")
	s.render(w, r, http.StatusOK, "licenses", v)
}

func (s *Server) setLang(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := s.Users.SetLang(u.Sub, r.PostFormValue("lang")); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, safeReferer(r), http.StatusSeeOther)
}

// safeReferer returns the path of the referring page on this site, or "/".
func safeReferer(r *http.Request) string {
	ref, err := r.URL.Parse(r.Header.Get("Referer"))
	if err != nil || ref.Host != r.Host || ref.Path == "" {
		return "/"
	}
	return ref.RequestURI()
}
