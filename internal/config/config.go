// SPDX-License-Identifier: Apache-2.0

// Package config reads Kanso's settings from the environment.
package config

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

// Config holds everything the server needs to start.
type Config struct {
	Addr             string
	DataDir          string
	StateDir         string
	BaseURL          string
	OIDCIssuer       string
	OIDCClientID     string
	OIDCClientSecret string
	AdminGroup       string
	DevLogin         bool
}

// Load reads the configuration through getenv (usually os.Getenv).
func Load(getenv func(string) string) (Config, error) {
	c := Config{
		Addr:         orDefault(getenv("KANSO_ADDR"), "127.0.0.1:8090"),
		DataDir:      getenv("KANSO_DATA_DIR"),
		StateDir:     getenv("KANSO_STATE_DIR"),
		BaseURL:      strings.TrimSuffix(getenv("KANSO_BASE_URL"), "/"),
		OIDCIssuer:   getenv("KANSO_OIDC_ISSUER"),
		OIDCClientID: getenv("KANSO_OIDC_CLIENT_ID"),
		AdminGroup:   orDefault(getenv("KANSO_ADMIN_GROUP"), "kanso-admin"),
		DevLogin:     getenv("KANSO_DEV_LOGIN") == "true",
	}
	secret, err := readSecret(getenv)
	if err != nil {
		return Config{}, err
	}
	c.OIDCClientSecret = secret
	return c, c.validate()
}

func (c Config) validate() error {
	if c.DataDir == "" || c.StateDir == "" || c.BaseURL == "" {
		return errors.New("KANSO_DATA_DIR, KANSO_STATE_DIR and KANSO_BASE_URL are required")
	}
	if c.DevLogin {
		return checkLoopback(c.Addr)
	}
	if c.OIDCIssuer == "" || c.OIDCClientID == "" || c.OIDCClientSecret == "" {
		return errors.New("KANSO_OIDC_ISSUER, KANSO_OIDC_CLIENT_ID and a client secret are required")
	}
	return nil
}

func checkLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("KANSO_ADDR: %w", err)
	}
	if ip := net.ParseIP(host); host == "localhost" || (ip != nil && ip.IsLoopback()) {
		return nil
	}
	return errors.New("KANSO_DEV_LOGIN is only allowed on a loopback address")
}

func readSecret(getenv func(string) string) (string, error) {
	file := getenv("KANSO_OIDC_CLIENT_SECRET_FILE")
	if file == "" {
		return getenv("KANSO_OIDC_CLIENT_SECRET"), nil
	}
	b, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("KANSO_OIDC_CLIENT_SECRET_FILE: %w", err)
	}
	return strings.TrimSpace(string(b)), nil
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
