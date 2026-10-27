# CLAUDE.md

Hüpattl! Kanso: Go backend, plain HTML/CSS/JS frontend, all content as files
in a Git repository. Design: `docs/superpowers/specs/2026-10-26-kanso-design.md`.

## Rules

- **License:** Apache-2.0. Every source file starts with
  `SPDX-License-Identifier: Apache-2.0`.
- **Dependencies:** as few as possible. Approved: `github.com/yuin/goldmark/v2`,
  `go.yaml.in/yaml/v3`, mermaid (ESM build without ELK, vendored), IBM Plex
  (vendored); dev-only: `@playwright/test` for e2e. Anything else: run the `huepattl-code-quality` dependency check
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
just seed   # made-up people and spaces in an empty ./tmp (restart run after)
just test   # go test ./...
just lint   # gofmt + go vet
```

## Tests

- Unit tests for every package (`go test ./...`), written first (TDD).
- End-to-end tests for every user-facing flow: Go HTTP tests in `internal/e2e`
  and browser tests with Playwright in `e2e/` (`just e2e`).

## Documentation

Update these in the same commit as the change they describe:

- `CHANGELOG.md` — [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), entries under `[Unreleased]`.
- `docs/adr/NNN-slug.md` — one per significant decision: Context, Decision,
  Alternatives Considered, Consequences; header `**Status:**`, `**Created:**`,
  `**Updated:**`. Next sequential number; list it in arc42 chapter 9.
- `docs/architecture/arc42.md` — when building blocks, runtime flows,
  deployment, quality goals or risks change; bump `**Updated:**`.
- `docs/features/YYYY-MM-DD name.md` — one per feature, frontmatter `status`
  (planned/active/done), `created`, `updated`; link ADRs under `## Related`.
