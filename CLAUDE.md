# CLAUDE.md

Hüpattl! Kanso: Go backend, plain HTML/CSS/JS frontend, all content as files
in a Git repository. Design: `docs/superpowers/specs/2026-10-26-kanso-design.md`.

## Rules

- **License:** Apache-2.0. Every source file starts with
  `SPDX-License-Identifier: Apache-2.0`.
- **Dependencies:** as few as possible. Approved: `github.com/yuin/goldmark/v2`,
  `go.yaml.in/yaml/v3`, mermaid (ESM build without ELK, vendored), IBM Plex
  (vendored). Anything else: run the `huepattl-code-quality` dependency check
  and get the owner's approval first. Its license text goes to
  `web/static/licenses/`.
- **Skills:** `huepattl-code-quality` for code, `huepattl-rams-design` for UI
  (project deviations in `docs/design.md`), `huepattl-legal-check` before a
  release or when something new leaves the machine.
- **Structure:** packages by domain under `internal/<capability>`; functions
  ~30 lines, cyclomatic ≤ 10.
- **Git** is called as a program with an argument list, never via a shell.
- **No telemetry, no CDN, no update check.**

## Commands

```sh
just run    # dev login, data in ./tmp
just test   # go test ./...
just lint   # gofmt + go vet
```

## Documentation

- `CHANGELOG.md` (Keep a Changelog), update for every meaningful change.
- `docs/adr/NNN-slug.md`: Context, Decision, Alternatives Considered,
  Consequences; header `**Status:**`, `**Created:**`, `**Updated:**`.
