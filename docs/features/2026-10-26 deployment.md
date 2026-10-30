---
status: done
created: 2026-10-26
updated: 2026-10-30
---

# Deployment

Runs on the homelab NAS as a Nix flake input (`flake.nix`, `buildGoModule`)
with a NixOS module, an Authelia OIDC client and Caddy in front; in daily use
next to Docmost. The host configuration lives in the private homelab
repository, not here.

## Related

- Design: [spec](../superpowers/specs/2026-10-26-kanso-design.md)
- Guide: [Authelia](../deploy-authelia.md)
