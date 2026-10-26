// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func base() map[string]string {
	return map[string]string{
		"KANSO_DATA_DIR":           "/tmp/data",
		"KANSO_STATE_DIR":          "/tmp/state",
		"KANSO_BASE_URL":           "http://localhost:8090",
		"KANSO_OIDC_ISSUER":        "https://id.example.org",
		"KANSO_OIDC_CLIENT_ID":     "kanso",
		"KANSO_OIDC_CLIENT_SECRET": "s3cret",
	}
}

func TestLoadDefaults(t *testing.T) {
	c, err := Load(env(base()))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != "127.0.0.1:8090" {
		t.Errorf("Addr = %q", c.Addr)
	}
	if c.AdminGroup != "kanso-admin" {
		t.Errorf("AdminGroup = %q", c.AdminGroup)
	}
}

func TestLoadRequiresOIDCUnlessDevLogin(t *testing.T) {
	m := base()
	delete(m, "KANSO_OIDC_ISSUER")
	if _, err := Load(env(m)); err == nil {
		t.Fatal("expected error without issuer")
	}
	m["KANSO_DEV_LOGIN"] = "true"
	if _, err := Load(env(m)); err != nil {
		t.Fatalf("dev login without issuer: %v", err)
	}
}

func TestLoadRequiresDirs(t *testing.T) {
	m := base()
	delete(m, "KANSO_DATA_DIR")
	if _, err := Load(env(m)); err == nil {
		t.Fatal("expected error without data dir")
	}
}

func TestDevLoginRejectedOnPublicAddr(t *testing.T) {
	m := base()
	m["KANSO_DEV_LOGIN"] = "true"
	m["KANSO_ADDR"] = "0.0.0.0:8090"
	if _, err := Load(env(m)); err == nil {
		t.Fatal("expected error for dev login on public address")
	}
}

func TestSecretFromFile(t *testing.T) {
	f := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(f, []byte("fromfile\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m := base()
	delete(m, "KANSO_OIDC_CLIENT_SECRET")
	m["KANSO_OIDC_CLIENT_SECRET_FILE"] = f
	c, err := Load(env(m))
	if err != nil {
		t.Fatal(err)
	}
	if c.OIDCClientSecret != "fromfile" {
		t.Errorf("secret = %q", c.OIDCClientSecret)
	}
}
