// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"html/template"
	"net/http"
	"net/url"
)

// The dev login exists only for local development (loopback address, see
// config). It signs anyone in without a password.

var devPage = template.Must(template.New("dev").Parse(`<!doctype html>
<html lang="en"><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Dev login · Kanso</title>
<form method="post" action="/auth/dev">
<input type="hidden" name="next" value="{{.}}">
<p><label for="sub">User id</label><br><input id="sub" name="sub" required value="anna"></p>
<p><label for="name">Name</label><br><input id="name" name="name" required value="Anna"></p>
<p><label><input type="checkbox" name="admin"> Admin</label></p>
<p><button type="submit">Sign in</button></p>
</form></html>`))

func (a *Auth) devRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/dev?next="+url.QueryEscape(safeNext(r.URL.Query().Get("next"))), http.StatusSeeOther)
}

func (a *Auth) devForm(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = devPage.Execute(w, safeNext(r.URL.Query().Get("next")))
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
	a.startSession(w, r, c, r.PostFormValue("next"))
}
