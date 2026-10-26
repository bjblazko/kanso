// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"net/url"
	"slices"
	"testing"
	"time"
)

func provider(t *testing.T, f *fakeIdP) *Provider {
	t.Helper()
	p, err := Discover(context.Background(), f.srv.URL, "kanso", "secret", "http://localhost/auth/callback", f.srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func exchange(t *testing.T, f *fakeIdP) (Claims, error) {
	t.Helper()
	p := provider(t, f)
	req := NewAuthRequest()
	f.nonce = req.Nonce
	return p.Exchange(context.Background(), "good-code", req)
}

func TestExchangeRS256(t *testing.T) {
	f := newFakeIdP(t)
	c, err := exchange(t, f)
	if err != nil {
		t.Fatal(err)
	}
	if c.Sub != "anna" || c.Name != "Anna" || c.Email != "anna@example.org" || !slices.Contains(c.Groups, "kanso-admin") {
		t.Errorf("claims = %+v", c)
	}
}

func TestExchangeES256(t *testing.T) {
	f := newFakeIdP(t)
	f.alg, f.kid = "ES256", "ec1"
	if _, err := exchange(t, f); err != nil {
		t.Fatal(err)
	}
}

func TestExchangeRejectsBadTokens(t *testing.T) {
	cases := map[string]func(f *fakeIdP){
		"wrong iss":   func(f *fakeIdP) { f.claims["iss"] = "https://evil" },
		"wrong aud":   func(f *fakeIdP) { f.claims["aud"] = "other" },
		"expired":     func(f *fakeIdP) { f.claims["exp"] = time.Now().Add(-time.Hour).Unix() },
		"future iat":  func(f *fakeIdP) { f.claims["iat"] = time.Now().Add(time.Hour).Unix() },
		"wrong nonce": func(f *fakeIdP) { f.claims["nonce"] = "other" },
		"alg none":    func(f *fakeIdP) { f.alg = "none" },
		"alg HS256":   func(f *fakeIdP) { f.alg = "HS256" },
		"unknown kid": func(f *fakeIdP) { f.kid = "nope" },
		"tampered":    func(f *fakeIdP) { f.tamper = true },
		"wrong azp":   func(f *fakeIdP) { f.claims["azp"] = "other" },
		"no sub":      func(f *fakeIdP) { f.claims["sub"] = "" },
	}
	for name, mutate := range cases {
		f := newFakeIdP(t)
		mutate(f)
		if _, err := exchange(t, f); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestExchangeAcceptsAudienceList(t *testing.T) {
	f := newFakeIdP(t)
	f.claims["aud"] = []string{"other", "kanso"}
	f.claims["azp"] = "kanso"
	if _, err := exchange(t, f); err != nil {
		t.Fatal(err)
	}
}

func TestUnknownKidRefetchesOnce(t *testing.T) {
	f := newFakeIdP(t)
	p := provider(t, f)
	login := func() error {
		req := NewAuthRequest()
		f.nonce = req.Nonce
		_, err := p.Exchange(context.Background(), "good-code", req)
		return err
	}
	if err := login(); err != nil {
		t.Fatal(err)
	}
	f.kid = "nope"
	if login() == nil {
		t.Fatal("unknown kid accepted")
	}
	if f.jwksHit != 2 {
		t.Errorf("jwks fetched %d times, want 2 (initial + one refetch)", f.jwksHit)
	}
}

func TestBadCodeFails(t *testing.T) {
	f := newFakeIdP(t)
	p := provider(t, f)
	if _, err := p.Exchange(context.Background(), "bad", NewAuthRequest()); err == nil {
		t.Error("bad code accepted")
	}
}

func TestAuthURL(t *testing.T) {
	f := newFakeIdP(t)
	p := provider(t, f)
	req := NewAuthRequest()
	u, err := url.Parse(p.AuthURL(req))
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("state") != req.State || q.Get("nonce") != req.Nonce ||
		q.Get("scope") != "openid profile email groups" || q.Get("client_id") != "kanso" || q.Get("code_challenge") == "" {
		t.Errorf("query = %v", q)
	}
}

func TestDiscoverRejectsIssuerMismatch(t *testing.T) {
	f := newFakeIdP(t)
	if _, err := Discover(context.Background(), f.srv.URL+"/x", "kanso", "secret", "http://l/cb", f.srv.Client()); err == nil {
		t.Error("issuer mismatch accepted")
	}
}
