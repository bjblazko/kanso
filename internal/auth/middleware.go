// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/config"
	"github.com/bjblazko/kanso/internal/users"
)

const (
	sessionCookie = "kanso_session"
	authCookie    = "kanso_auth"
	authCookieTTL = 10 * time.Minute
)

// Auth wires sign-in, sessions and request protection together.
type Auth struct {
	cfg      config.Config
	provider *Provider // nil when only the dev login is used
	sessions *Sessions
	users    *users.Store
	signer   *signer
	secure   bool
	host     string
}

// NewAuth creates the authentication layer.
func NewAuth(cfg config.Config, p *Provider, s *Sessions, u *users.Store) (*Auth, error) {
	sg, err := loadSigner(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	local := base.Hostname() == "localhost" || base.Hostname() == "127.0.0.1"
	return &Auth{cfg: cfg, provider: p, sessions: s, users: u, signer: sg,
		secure: base.Scheme == "https" || !local, host: base.Host}, nil
}

// Routes registers the sign-in endpoints.
func (a *Auth) Routes(mux *http.ServeMux) {
	switch {
	case a.cfg.DevLogin:
		mux.HandleFunc("GET /auth/login", a.devRedirect)
		mux.HandleFunc("GET /auth/dev", a.devForm)
		mux.HandleFunc("POST /auth/dev", a.devLogin)
	case a.provider != nil:
		mux.HandleFunc("GET /auth/login", a.login)
		mux.HandleFunc("GET /auth/callback", a.callback)
	}
	mux.Handle("POST /auth/logout", a.RequireUser(http.HandlerFunc(a.logout)))
}

type ctxKey struct{}

type requestUser struct {
	user users.User
	sess Session
}

// RequireUser lets only signed-in people through and checks CSRF on writes.
func (a *Auth) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ru, ok := a.current(r)
		if !ok {
			a.deny(w, r)
			return
		}
		if !safeMethod(r.Method) && !a.validCSRF(r, ru.sess) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, ru)))
	})
}

func (a *Auth) current(r *http.Request) (requestUser, bool) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return requestUser{}, false
	}
	sess, ok := a.sessions.Get(c.Value)
	if !ok {
		return requestUser{}, false
	}
	u, ok := a.users.Get(sess.Sub)
	return requestUser{user: u, sess: sess}, ok
}

func (a *Auth) deny(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusSeeOther)
}

func safeMethod(m string) bool { return m == http.MethodGet || m == http.MethodHead }

// validCSRF needs the session token and a same-origin request.
func (a *Auth) validCSRF(r *http.Request, s Session) bool {
	token := r.Header.Get("X-CSRF-Token")
	if token == "" {
		token = r.PostFormValue("csrf")
	}
	return subtle.ConstantTimeCompare([]byte(token), []byte(s.CSRF)) == 1 && a.sameOrigin(r)
}

func (a *Auth) sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		origin = r.Header.Get("Referer")
	}
	u, err := url.Parse(origin)
	return err == nil && origin != "" && u.Host == a.host
}

// UserFrom returns the signed-in user of a request passed by RequireUser.
func UserFrom(ctx context.Context) (users.User, access.Principal) {
	ru, _ := ctx.Value(ctxKey{}).(requestUser)
	return ru.user, access.Principal{Sub: ru.sess.Sub, Admin: ru.sess.Admin}
}

// CSRFToken returns the token forms and scripts must send back.
func CSRFToken(ctx context.Context) string {
	ru, _ := ctx.Value(ctxKey{}).(requestUser)
	return ru.sess.CSRF
}

// safeNext keeps redirects on this site.
func safeNext(next string) string {
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") {
		return "/"
	}
	return next
}

type pendingLogin struct {
	AuthRequest
	Next string `json:"next"`
}

func (a *Auth) login(w http.ResponseWriter, r *http.Request) {
	pl := pendingLogin{AuthRequest: NewAuthRequest(), Next: safeNext(r.URL.Query().Get("next"))}
	b, _ := json.Marshal(pl)
	a.setCookie(w, authCookie, a.signer.sign(base64.RawURLEncoding.EncodeToString(b)), authCookieTTL)
	http.Redirect(w, r, a.provider.AuthURL(pl.AuthRequest), http.StatusSeeOther)
}

func (a *Auth) callback(w http.ResponseWriter, r *http.Request) {
	pl, ok := a.readPending(r)
	if !ok || r.URL.Query().Get("state") != pl.State {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	a.setCookie(w, authCookie, "", -1)
	claims, err := a.provider.Exchange(r.Context(), r.URL.Query().Get("code"), pl.AuthRequest)
	if err != nil {
		http.Error(w, "Sign-in failed", http.StatusBadGateway)
		return
	}
	a.startSession(w, r, claims, pl.Next)
}

func (a *Auth) readPending(r *http.Request) (pendingLogin, bool) {
	var pl pendingLogin
	c, err := r.Cookie(authCookie)
	if err != nil {
		return pl, false
	}
	v, ok := a.signer.verify(c.Value)
	b, err := base64.RawURLEncoding.DecodeString(v)
	if !ok || err != nil || json.Unmarshal(b, &pl) != nil {
		return pl, false
	}
	return pl, true
}

func (a *Auth) startSession(w http.ResponseWriter, r *http.Request, c Claims, next string) {
	u := users.User{Sub: c.Sub, Name: orDefault(c.Name, c.Sub), Email: orDefault(c.Email, c.Sub+"@users.kanso"), LastLogin: time.Now()}
	if _, err := a.users.Upsert(u); err != nil {
		http.Error(w, "Sign-in failed", http.StatusInternalServerError)
		return
	}
	go a.syncAvatar(u.Sub, c.Picture)
	admin := false
	for _, g := range c.Groups {
		admin = admin || g == a.cfg.AdminGroup
	}
	s := a.sessions.Create(u.Sub, admin)
	a.setCookie(w, sessionCookie, s.ID, sessionTTL)
	http.Redirect(w, r, safeNext(next), http.StatusSeeOther)
}

// syncAvatar copies the sign-in's picture in the background: a slow picture
// host must not hold up the sign-in.
func (a *Auth) syncAvatar(sub, picture string) {
	if err := a.users.SyncAvatar(context.Background(), avatarClient, sub, picture); err != nil {
		slog.Warn("profile picture", "sub", sub, "err", err)
	}
}

var avatarClient = &http.Client{Timeout: 15 * time.Second}

func (a *Auth) logout(w http.ResponseWriter, r *http.Request) {
	ru, _ := r.Context().Value(ctxKey{}).(requestUser)
	a.sessions.Delete(ru.sess.ID)
	a.setCookie(w, sessionCookie, "", -1)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *Auth) setCookie(w http.ResponseWriter, name, value string, ttl time.Duration) {
	c := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: a.secure, SameSite: http.SameSiteLaxMode}
	if ttl < 0 {
		c.MaxAge = -1
	} else {
		c.MaxAge = int(ttl.Seconds())
	}
	http.SetCookie(w, c)
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
