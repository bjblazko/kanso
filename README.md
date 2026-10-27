# Hüpattl! Kanso

A small, self-hosted organiser for a family or a small team: a Markdown wiki
with spaces, and later notes, tasks (Kanban and Eisenhower), lists and
calendars. No database: everything lives in plain files in a Git repository
you can read without Kanso.

*Weniger, aber besser* — less, but better.

## Status

Phase 1 is done: Markdown wiki with spaces, page tree, editor with live
preview, attachments, Mermaid diagrams, edit locks, history with diff and
restore, search; sign-in with OpenID Connect; roles per space; German and
English. Next: notes, tasks (Kanban and Eisenhower), lists, calendars.

## Run

Requires Go 1.27.3 or newer and `git` on the `PATH`.

```sh
just run          # dev login on http://localhost:8090, data in ./tmp
just test         # unit and HTTP tests
just e2e          # browser tests (Playwright, Chromium + WebKit)
```

## Configuration

| Variable | Meaning |
|---|---|
| `KANSO_ADDR` | Listen address, default `127.0.0.1:8090` |
| `KANSO_DATA_DIR` | Content (a Git repository, created if missing) |
| `KANSO_STATE_DIR` | Runtime state: sessions, keys |
| `KANSO_BASE_URL` | Public URL, e.g. `https://kanso.example.org` |
| `KANSO_OIDC_ISSUER` | OpenID Connect issuer, e.g. Authelia |
| `KANSO_OIDC_CLIENT_ID` | OIDC client id |
| `KANSO_OIDC_CLIENT_SECRET` / `_FILE` | OIDC client secret, or a file holding it |
| `KANSO_ADMIN_GROUP` | Group claim that makes a user admin, default `kanso-admin` |
| `KANSO_DEV_LOGIN` | `true` enables a password-less test login (loopback only) |

Deployment with Authelia: [docs/deploy-authelia.md](docs/deploy-authelia.md).
Architecture: [docs/architecture/arc42.md](docs/architecture/arc42.md).

## Privacy

Kanso sends no telemetry and checks for no updates. It talks only to your
OIDC provider. It stores the name and e-mail address the provider sends, in
`users/<id>.yaml` in the data directory, and writes them into the history as
author of each change. Logs contain errors and request paths, no addresses.

## License

Apache-2.0, see [LICENSE](LICENSE). Third-party licenses: [NOTICE](NOTICE).

This software is provided "as is", without warranty of any kind.
