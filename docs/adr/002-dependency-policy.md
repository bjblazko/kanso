# 002 — Minimal, Approved Dependencies

**Status:** Accepted
**Created:** 2026-10-26
**Updated:** 2026-10-26

## Context

Kanso should stay small, auditable and long-lived. Some features cannot
reasonably be built from scratch (Markdown parsing, Mermaid diagrams).

## Decision

Every third-party library, font or vendored script needs a check
(need, license, maintenance, known vulnerabilities) and the owner's approval.
Approved on 2026-10-26:

| Dependency | License | Notes |
|---|---|---|
| `github.com/yuin/goldmark/v2` | MIT | raw HTML disabled |
| `go.yaml.in/yaml/v3` | Apache-2.0 / MIT | frozen, security fixes; v4 later |
| mermaid 12.2.0, ESM build | MIT + bundled MIT/ISC/BSD/Apache | ELK chunk (EPL-2.0) removed, layout `dagre`; vendored, lazy |
| IBM Plex Sans/Mono | OFL-1.1 | self-hosted |
| `@playwright/test` 1.64.1 (dev only) | Apache-2.0 | end-to-end tests in `e2e/`; never shipped; GHSA-7mvr-c777-76hp fixed long before this version |

Rejected: `pgx` (the Docmost migration is not part of the project).
OIDC, Web Push, iCalendar and Git access are built on the standard library
and the `git` program.

## Alternatives Considered

- **No dependencies at all:** an own Markdown parser and diagram renderer
  would cost months and add risk.
- **Mermaid with ELK:** better layouts, but adds EPL-2.0 notice duties.

## Consequences

- License texts ship in `web/static/licenses/` and on the in-app
  "Licenses and thanks" page; a test keeps that list complete.
