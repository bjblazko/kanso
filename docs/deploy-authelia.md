# Deploying with Authelia

**Created:** 2026-10-26
**Updated:** 2026-10-27

Kanso signs people in with OpenID Connect (see
[ADR-001](adr/001-oidc-with-kanso-managed-roles.md)). This is the client
Authelia needs.

## Authelia client

```yaml
identity_providers:
  oidc:
    # Kanso reads name, e-mail and groups from the ID token. Authelia ≥ 4.39
    # puts them only into userinfo unless a claims policy asks for them.
    claims_policies:
      kanso:
        id_token: [name, email, email_verified, preferred_username, groups]
    clients:
      - client_id: kanso
        client_name: Hüpattl! Kanso
        # authelia crypto hash generate pbkdf2 --variant sha512 --password '<secret>'
        client_secret: '$pbkdf2-sha512$…'
        redirect_uris:
          - https://kanso.example.org/auth/callback
        scopes: [openid, profile, email, groups]
        grant_types: [authorization_code]
        response_types: [code]
        authorization_policy: one_factor
        require_pkce: true
        pkce_challenge_method: S256
        token_endpoint_auth_method: client_secret_basic
        claims_policy: kanso
```

Put everyone who should administer Kanso into the Authelia group
`kanso-admin` (or set `KANSO_ADMIN_GROUP`). Everything else — groups,
who may read or write which space — is managed inside Kanso.

## Kanso

```sh
KANSO_ADDR=127.0.0.1:8090
KANSO_BASE_URL=https://kanso.example.org
KANSO_DATA_DIR=/srv/kanso/data        # content, a Git repository
KANSO_STATE_DIR=/var/lib/kanso        # sessions and keys
KANSO_OIDC_ISSUER=https://id.example.org
KANSO_OIDC_CLIENT_ID=kanso
KANSO_OIDC_CLIENT_SECRET_FILE=/run/secrets/kanso-oidc
```

On NixOS, `flake.nix` provides `packages.<system>.default` (built with Go
1.27.3 from its own nixpkgs pin); the NixOS module lives with the host
configuration. Run it behind a reverse proxy that terminates TLS. `git` must be on the
`PATH`. Build with Go 1.27.3 or newer (GO-2026-6809 in `crypto/x509` is fixed
there). Back up both directories; the data directory alone is enough to
rebuild everything except sessions.
