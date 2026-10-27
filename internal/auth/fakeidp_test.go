// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeIdP is a minimal OIDC provider for tests.
type fakeIdP struct {
	srv     *httptest.Server
	rsaKey  *rsa.PrivateKey
	ecKey   *ecdsa.PrivateKey
	alg     string         // RS256, ES256, none, HS256
	kid     string         // kid put into the token header
	claims  map[string]any // overrides
	tamper  bool
	nonce   string // nonce to echo; set by the test from the AuthRequest
	jwksHit int
}

func newFakeIdP(t *testing.T) *fakeIdP {
	t.Helper()
	rk, _ := rsa.GenerateKey(rand.Reader, 2048)
	ek, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	f := &fakeIdP{rsaKey: rk, ecKey: ek, alg: "RS256", kid: "rsa1", claims: map[string]any{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", f.discovery)
	mux.HandleFunc("/jwks", f.jwks)
	mux.HandleFunc("/token", f.token)
	f.srv = httptest.NewServer(mux)
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeIdP) discovery(w http.ResponseWriter, _ *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"issuer":                           f.srv.URL,
		"authorization_endpoint":           f.srv.URL + "/authorize",
		"token_endpoint":                   f.srv.URL + "/token",
		"jwks_uri":                         f.srv.URL + "/jwks",
		"response_types_supported":         []string{"code"},
		"code_challenge_methods_supported": []string{"S256"},
	})
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (f *fakeIdP) jwks(w http.ResponseWriter, _ *http.Request) {
	f.jwksHit++
	pad := func(b []byte) []byte { out := make([]byte, 32); copy(out[32-len(b):], b); return out }
	_ = json.NewEncoder(w).Encode(map[string]any{"keys": []map[string]string{
		{"kty": "RSA", "kid": "rsa1", "alg": "RS256", "use": "sig",
			"n": b64(f.rsaKey.N.Bytes()), "e": b64(big.NewInt(int64(f.rsaKey.E)).Bytes())},
		{"kty": "EC", "kid": "ec1", "alg": "ES256", "crv": "P-256",
			"x": b64(pad(f.ecKey.X.Bytes())), "y": b64(pad(f.ecKey.Y.Bytes()))},
	}})
}

func (f *fakeIdP) token(w http.ResponseWriter, r *http.Request) {
	user, pass, ok := r.BasicAuth()
	if !ok || user != "kanso" || pass != "secret" || r.FormValue("code") != "good-code" || r.FormValue("code_verifier") == "" {
		http.Error(w, `{"error":"invalid_grant"}`, http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "at", "token_type": "Bearer", "id_token": f.idToken()})
}

func (f *fakeIdP) idToken() string {
	now := time.Now().Unix()
	c := map[string]any{"iss": f.srv.URL, "aud": "kanso", "sub": "anna", "name": "Anna",
		"email": "anna@example.org", "groups": []string{"familie", "kanso-admin"},
		"exp": now + 300, "iat": now, "nonce": f.nonce}
	for k, v := range f.claims {
		c[k] = v
	}
	hb, _ := json.Marshal(map[string]string{"alg": f.alg, "kid": f.kid, "typ": "JWT"})
	cb, _ := json.Marshal(c)
	signing := b64(hb) + "." + b64(cb)
	sig := f.sign([]byte(signing))
	if f.tamper {
		c["sub"] = "mallory"
		cb, _ = json.Marshal(c)
		signing = b64(hb) + "." + b64(cb)
	}
	return signing + "." + b64(sig)
}

func (f *fakeIdP) sign(data []byte) []byte {
	h := sha256.Sum256(data)
	switch f.alg {
	case "RS256":
		s, _ := rsa.SignPKCS1v15(rand.Reader, f.rsaKey, crypto.SHA256, h[:])
		return s
	case "ES256":
		r, s, _ := ecdsa.Sign(rand.Reader, f.ecKey, h[:])
		out := make([]byte, 64)
		r.FillBytes(out[:32])
		s.FillBytes(out[32:])
		return out
	}
	return []byte("sig")
}
