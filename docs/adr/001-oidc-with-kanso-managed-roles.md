# 001 — OIDC for Identity, Roles and Groups Managed in Kanso

**Status:** Accepted
**Created:** 2026-10-26
**Updated:** 2026-10-26

## Context

Every user, including guests such as grandparents, has an account in the
household's identity provider (Authelia). Permissions are per space and
change more often than identity provider configuration should.

## Decision

Kanso signs users in with OpenID Connect (authorization code + PKCE),
implemented with the Go standard library. The identity provider supplies
identity and one flag: membership in the group `kanso-admin` makes a user
admin. Groups and per-space roles (owner, editor, viewer) are managed in
Kanso and stored in its data directory. A user without a role in a space
does not see it.

## Alternatives Considered

- **Groups from the identity provider:** one source, but every new group
  needs an IdP change.
- **OIDC library (coreos/go-oidc):** less own code, but an extra dependency
  for a small, well-specified flow.

## Consequences

- Removing a role takes effect on the next request.
- Removing admin rights in the IdP takes effect at the next login.
