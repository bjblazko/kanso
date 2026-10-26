// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"
)

const clockSkew = 2 * time.Minute

type jwtHeader struct {
	Alg string `json:"alg"`
	Kid string `json:"kid"`
}

type idClaims struct {
	Claims
	Iss   string   `json:"iss"`
	Aud   audience `json:"aud"`
	Azp   string   `json:"azp"`
	Exp   int64    `json:"exp"`
	Iat   int64    `json:"iat"`
	Nonce string   `json:"nonce"`
}

// audience accepts the "aud" claim as a string or a list.
type audience []string

func (a *audience) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*a = audience{one}
		return nil
	}
	return json.Unmarshal(b, (*[]string)(a))
}

func (p *Provider) verify(ctx context.Context, raw, nonce string) (Claims, error) {
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return Claims{}, errors.New("id token: malformed")
	}
	var h jwtHeader
	if err := decodePart(parts[0], &h); err != nil {
		return Claims{}, err
	}
	if err := p.checkSignature(ctx, h, parts); err != nil {
		return Claims{}, err
	}
	var c idClaims
	if err := decodePart(parts[1], &c); err != nil {
		return Claims{}, err
	}
	return c.Claims, p.checkClaims(c, nonce, time.Now())
}

func decodePart(s string, v any) error {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return fmt.Errorf("id token: %w", err)
	}
	return json.Unmarshal(b, v)
}

func (p *Provider) checkSignature(ctx context.Context, h jwtHeader, parts []string) error {
	if h.Alg != "RS256" && h.Alg != "ES256" {
		return fmt.Errorf("id token: algorithm %q not allowed", h.Alg)
	}
	key, err := p.keys.get(ctx, h.Kid)
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if !verifySig(h.Alg, key, digest[:], sig) {
		return errors.New("id token: bad signature")
	}
	return nil
}

func verifySig(alg string, key crypto.PublicKey, digest, sig []byte) bool {
	switch k := key.(type) {
	case *rsa.PublicKey:
		return alg == "RS256" && rsa.VerifyPKCS1v15(k, crypto.SHA256, digest, sig) == nil
	case *ecdsa.PublicKey:
		if alg != "ES256" || len(sig) != 64 {
			return false
		}
		r, s := new(big.Int).SetBytes(sig[:32]), new(big.Int).SetBytes(sig[32:])
		return ecdsa.Verify(k, digest, r, s)
	}
	return false
}

func (p *Provider) checkClaims(c idClaims, nonce string, now time.Time) error {
	switch {
	case c.Iss != p.issuer:
		return errors.New("id token: wrong issuer")
	case !slices.Contains(c.Aud, p.clientID):
		return errors.New("id token: wrong audience")
	case c.Azp != "" && c.Azp != p.clientID:
		return errors.New("id token: wrong authorized party")
	case now.After(time.Unix(c.Exp, 0).Add(clockSkew)):
		return errors.New("id token: expired")
	case time.Unix(c.Iat, 0).After(now.Add(clockSkew)):
		return errors.New("id token: issued in the future")
	case c.Nonce != nonce:
		return errors.New("id token: wrong nonce")
	case c.Sub == "":
		return errors.New("id token: no subject")
	}
	return nil
}

// keySet caches the issuer's signing keys.
type keySet struct {
	url  string
	hc   *http.Client
	mu   sync.Mutex
	keys map[string]crypto.PublicKey
}

// get returns the key with kid, refetching the set once if it is unknown.
func (ks *keySet) get(ctx context.Context, kid string) (crypto.PublicKey, error) {
	ks.mu.Lock()
	defer ks.mu.Unlock()
	if k, ok := ks.keys[kid]; ok {
		return k, nil
	}
	if err := ks.fetch(ctx); err != nil {
		return nil, err
	}
	if k, ok := ks.keys[kid]; ok {
		return k, nil
	}
	return nil, fmt.Errorf("id token: unknown key %q", kid)
}

type jwk struct {
	Kty, Kid, Crv, N, E, X, Y string
}

func (ks *keySet) fetch(ctx context.Context) error {
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := getJSON(ctx, ks.hc, ks.url, &set); err != nil {
		return fmt.Errorf("jwks: %w", err)
	}
	ks.keys = map[string]crypto.PublicKey{}
	for _, k := range set.Keys {
		if pub, err := k.publicKey(); err == nil {
			ks.keys[k.Kid] = pub
		}
	}
	return nil
}

func (k jwk) publicKey() (crypto.PublicKey, error) {
	switch {
	case k.Kty == "RSA":
		n, e := decodeInt(k.N), decodeInt(k.E)
		if n == nil || e == nil || !e.IsInt64() {
			return nil, errors.New("jwk: bad RSA key")
		}
		return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
	case k.Kty == "EC" && k.Crv == "P-256":
		return p256Key(k.X, k.Y)
	}
	return nil, fmt.Errorf("jwk: unsupported key type %q", k.Kty)
}

// p256Key builds a P-256 key from base64url coordinates; the standard
// library checks that the point lies on the curve.
func p256Key(x, y string) (*ecdsa.PublicKey, error) {
	xb, errX := base64.RawURLEncoding.DecodeString(x)
	yb, errY := base64.RawURLEncoding.DecodeString(y)
	if errX != nil || errY != nil || len(xb) != 32 || len(yb) != 32 {
		return nil, errors.New("jwk: bad EC key")
	}
	point := append(append([]byte{4}, xb...), yb...)
	return ecdsa.ParseUncompressedPublicKey(elliptic.P256(), point)
}

func decodeInt(s string) *big.Int {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) == 0 {
		return nil
	}
	return new(big.Int).SetBytes(b)
}
