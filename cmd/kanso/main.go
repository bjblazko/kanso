// SPDX-License-Identifier: Apache-2.0

// Command kanso serves Hüpattl! Kanso.
package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/bjblazko/kanso/internal/config"
)

func main() {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "kanso:", err)
		os.Exit(2)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "ok")
	})
	slog.Info("listening", "addr", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
