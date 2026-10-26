// SPDX-License-Identifier: Apache-2.0

// Package auth signs people in with OpenID Connect and keeps their sessions.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
)

const scopes = "openid profile email groups"

// Provider is a discovered OpenID Connect issuer.
type Provider struct {
	issuer, clientID, secret, redirect string
	authEndpoint, tokenEndpoint        string
	keys                               *keySet
	hc                                 *http.Client
}

type discovery struct {
	Issuer         string   `json:"issuer"`
	Authorization  string   `json:"authorization_endpoint"`
	Token          string   `json:"token_endpoint"`
	JWKS           string   `json:"jwks_uri"`
	ResponseTypes  []string `json:"response_types_supported"`
	PKCEChallenges []string `json:"code_challenge_methods_supported"`
}

// Discover reads the issuer's configuration.
func Discover(ctx context.Context, issuer, clientID, secret, redirectURL string, hc *http.Client) (*Provider, error) {
	var d discovery
	if err := getJSON(ctx, hc, strings.TrimSuffix(issuer, "/")+"/.well-known/openid-configuration", &d); err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	if err := d.check(issuer); err != nil {
		return nil, err
	}
	return &Provider{
		issuer: d.Issuer, clientID: clientID, secret: secret, redirect: redirectURL,
		authEndpoint: d.Authorization, tokenEndpoint: d.Token,
		keys: &keySet{url: d.JWKS, hc: hc}, hc: hc,
	}, nil
}

func (d discovery) check(issuer string) error {
	if strings.TrimSuffix(d.Issuer, "/") != strings.TrimSuffix(issuer, "/") {
		return fmt.Errorf("oidc: issuer %q does not match %q", d.Issuer, issuer)
	}
	if !slices.Contains(d.ResponseTypes, "code") || !slices.Contains(d.PKCEChallenges, "S256") {
		return errors.New("oidc: provider lacks code flow with PKCE S256")
	}
	return nil
}

// AuthRequest holds the per-login secrets that must survive the redirect.
type AuthRequest struct {
	State    string `json:"s"`
	Nonce    string `json:"n"`
	Verifier string `json:"v"`
}

// NewAuthRequest creates fresh random state, nonce and PKCE verifier.
func NewAuthRequest() AuthRequest {
	return AuthRequest{State: randomString(), Nonce: randomString(), Verifier: randomString()}
}

func randomString() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// AuthURL is where the browser goes to sign in.
func (p *Provider) AuthURL(r AuthRequest) string {
	challenge := sha256.Sum256([]byte(r.Verifier))
	q := url.Values{
		"response_type":         {"code"},
		"client_id":             {p.clientID},
		"redirect_uri":          {p.redirect},
		"scope":                 {scopes},
		"state":                 {r.State},
		"nonce":                 {r.Nonce},
		"code_challenge":        {base64.RawURLEncoding.EncodeToString(challenge[:])},
		"code_challenge_method": {"S256"},
	}
	return p.authEndpoint + "?" + q.Encode()
}

// Claims is what Kanso uses from the ID token.
type Claims struct {
	Sub    string   `json:"sub"`
	Name   string   `json:"name"`
	Email  string   `json:"email"`
	Groups []string `json:"groups"`
}

// Exchange trades the code for tokens and returns the verified claims.
func (p *Provider) Exchange(ctx context.Context, code string, r AuthRequest) (Claims, error) {
	raw, err := p.fetchIDToken(ctx, code, r.Verifier)
	if err != nil {
		return Claims{}, err
	}
	return p.verify(ctx, raw, r.Nonce)
}

func (p *Provider) fetchIDToken(ctx context.Context, code, verifier string) (string, error) {
	form := url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {p.redirect}, "code_verifier": {verifier},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(url.QueryEscape(p.clientID), url.QueryEscape(p.secret))
	var tok struct {
		IDToken string `json:"id_token"`
	}
	if err := doJSON(p.hc, req, &tok); err != nil {
		return "", fmt.Errorf("oidc token: %w", err)
	}
	if tok.IDToken == "" {
		return "", errors.New("oidc token: no id_token")
	}
	return tok.IDToken, nil
}

const maxBody = 1 << 20

func getJSON(ctx context.Context, hc *http.Client, u string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	return doJSON(hc, req, v)
}

func doJSON(hc *http.Client, req *http.Request, v any) error {
	req.Header.Set("Accept", "application/json")
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP %d", req.URL.Path, resp.StatusCode)
	}
	return json.Unmarshal(body, v)
}
