# Hüpattl! Kanso

A small, self-hosted organiser for a family or a small team: a Markdown wiki
with spaces, and later notes, tasks (Kanban and Eisenhower), lists and
calendars. No database: everything lives in plain files in a Git repository
you can read without Kanso.

*Weniger, aber besser* — less, but better.

## Status

Early development. Phase 1 (wiki, login, roles) is being built.

## Run

Requires Go 1.27 and `git` on the `PATH`.

```sh
just run          # dev login on http://localhost:8090, data in ./tmp
just test
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

## Privacy

Kanso sends no telemetry and checks for no updates. It talks only to your
OIDC provider.

## License

Apache-2.0, see [LICENSE](LICENSE). Third-party licenses: [NOTICE](NOTICE).

This software is provided "as is", without warranty of any kind.
