// SPDX-License-Identifier: Apache-2.0

// Command kanso serves Hüpattl! Kanso.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/config"
	"github.com/bjblazko/kanso/internal/i18n"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
	"github.com/bjblazko/kanso/internal/web"
	"github.com/bjblazko/kanso/internal/wiki"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kanso:", err)
		os.Exit(2)
	}
	if err := run(cfg); err != nil {
		slog.Error("kanso stopped", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config) error {
	deps, err := buildDeps(cfg)
	if err != nil {
		return err
	}
	srv, err := web.New(deps)
	if err != nil {
		return err
	}
	return serve(cfg.Addr, srv.Handler())
}

func buildDeps(cfg config.Config) (web.Deps, error) {
	repo, err := store.OpenRepo(cfg.DataDir)
	if err != nil {
		return web.Deps{}, err
	}
	us, err := users.NewStore(cfg.DataDir, repo)
	if err != nil {
		return web.Deps{}, err
	}
	a, err := buildAuth(cfg, us)
	if err != nil {
		return web.Deps{}, err
	}
	sp, err := spaces.NewService(cfg.DataDir, repo)
	if err != nil {
		return web.Deps{}, err
	}
	wk, err := wiki.NewWiki(cfg.DataDir, repo)
	if err != nil {
		return web.Deps{}, err
	}
	bundle, err := i18n.Load()
	return web.Deps{Cfg: cfg, Auth: a, I18n: bundle, Users: us, Repo: repo, Spaces: sp, Wiki: wk, MD: markdown.New(),
		Locks: wiki.NewLocks(5*time.Minute, time.Now)}, err
}

func buildAuth(cfg config.Config, us *users.Store) (*auth.Auth, error) {
	sessions, err := auth.NewSessions(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	var provider *auth.Provider
	if !cfg.DevLogin {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		provider, err = auth.Discover(ctx, cfg.OIDCIssuer, cfg.OIDCClientID, cfg.OIDCClientSecret,
			cfg.BaseURL+"/auth/callback", &http.Client{Timeout: 15 * time.Second})
		if err != nil {
			return nil, err
		}
	}
	return auth.NewAuth(cfg, provider, sessions, us)
}

// serve runs until SIGINT/SIGTERM and then shuts down gracefully.
func serve(addr string, h http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errc := make(chan error, 1)
	go func() { errc <- srv.ListenAndServe() }()
	slog.Info("listening", "addr", addr)
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
