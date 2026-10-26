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
- Web shell: Rams design tokens, self-hosted IBM Plex, mobile-first layout, security headers, "Licenses and thanks" page, language switch.
- End-to-end tests with Playwright (desktop Chromium, iPhone WebKit).
- Spaces with members (people or groups as viewer, editor, owner); admins create spaces and manage groups; own line icons.
- Wiki pages as folders with `index.md`: page tree, stable ids (`/p/<id>`), sub-pages, move, delete with confirmation; history kept through `git mv`.
- Safe Markdown rendering (goldmark, GitHub-flavoured): no raw HTML, link allowlist, attachment paths per page, Mermaid blocks marked for the diagram script; preview endpoint for the editor.
- Attachments per page (up to 50 MiB): safe readable file names, fixed content types, SVG and unknown types only as download.
- Edit locks (ADR-003): five minutes after the last heartbeat, owners and admins can break them; saving without the lock is refused (409) and keeps the text.
- Markdown editor: source and live preview (side by side on wide screens, tabs on phones), attach files by button, drag and drop or paste, ⌘S/Ctrl+S, warning before leaving with unsaved changes.
- Mermaid diagrams: mermaid 12.2.0 vendored without the ELK layout (EPL-2.0) and without source maps, loaded only on pages with diagrams and in the editor; strict security level, labels as plain SVG text.
- Page history: every version with author and time, differences of the text to the current version, restore as a new version (respects edit locks).
- Search across titles and texts of all readable spaces: in memory, umlauts match either spelling, last word as prefix, title hits first, plain-text snippets.
- Deployment guide for Authelia (`docs/deploy-authelia.md`).

### Fixed
- Sign-in no longer fails for names with quotes or angle brackets (they are dropped from the Git author only).
- Design review: button links no longer underlined, navigation folded on phones, neutral checkboxes.
- ECDSA keys parsed with the non-deprecated standard-library API.
- Documentation: arc42 architecture, ADRs 000–003, feature descriptions.
