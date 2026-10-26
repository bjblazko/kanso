# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Repository skeleton, configuration from environment, `/healthz`.
- Safe file paths, slugs, atomic Markdown documents with YAML frontmatter.
- Git history for the data directory: commit per change with author, log, show, diff, move.
- Access model: roles viewer, editor, owner per space; admins; groups in `kanso.yaml`.
- User profiles in `users/<sub>.yaml`, created at sign-in, with interface language.
- OpenID Connect sign-in (code flow + PKCE, RS256/ES256 ID token verification) with the Go standard library.
- Sessions (30 days, sliding), CSRF protection with token and origin check, password-less dev login on loopback only.
- German and English interface texts, language from profile or browser.
- Documentation: arc42 architecture, ADRs 000–003, feature descriptions.
