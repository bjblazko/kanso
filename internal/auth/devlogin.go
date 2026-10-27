// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"html/template"
	"net/http"
	"net/url"

	"github.com/bjblazko/kanso/internal/devseed"
)

// The dev login exists only for local development (loopback address, see
// config). It signs anyone in without a password.

// The page is plain HTML without the app's style sheets: it is a developer
// tool, and static files are cached by fingerprinted URLs auth does not know.
var devPage = template.Must(template.New("dev").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Dev login · Kanso</title>
<h1>Dev login</h1>
<h2>Sign in as</h2>
<ul>
{{range .People}}<li><form method="post" action="/auth/dev">
<input type="hidden" name="next" value="{{$.Next}}">
<input type="hidden" name="sub" value="{{.Sub}}">
<input type="hidden" name="name" value="{{.Name}}">
{{if .Admin}}<input type="hidden" name="admin" value="on">{{end}}
<button type="submit">{{.Name}}</button> <small>{{.Note}}</small>
</form></li>
{{end}}</ul>
<p><small>Roles apply to the data from <code>just seed</code>.</small></p>
<h2>Someone else</h2>
<form method="post" action="/auth/dev">
<input type="hidden" name="next" value="{{.Next}}">
<p><label for="sub">User id</label><br><input id="sub" name="sub" required value="anna"></p>
<p><label for="name">Name</label><br><input id="name" name="name" required value="Anna"></p>
<p><label><input type="checkbox" name="admin"> Admin</label></p>
<p><button type="submit">Sign in</button></p>
</form></html>`))

type devView struct {
	Next   string
	People []devseed.Person
}

func (a *Auth) devRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/dev?next="+url.QueryEscape(safeNext(r.URL.Query().Get("next"))), http.StatusSeeOther)
}

func (a *Auth) devForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = devPage.Execute(w, devView{Next: safeNext(r.URL.Query().Get("next")), People: devseed.People})
}

func (a *Auth) devLogin(w http.ResponseWriter, r *http.Request) {
	if !a.sameOrigin(r) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	c := Claims{Sub: r.PostFormValue("sub"), Name: r.PostFormValue("name")}
	if r.PostFormValue("admin") != "" {
		c.Groups = []string{a.cfg.AdminGroup}
	}
	if ru, ok := a.current(r); ok {
		a.sessions.Delete(ru.sess.ID)
	}
	a.startSession(w, r, c, r.PostFormValue("next"))
}
