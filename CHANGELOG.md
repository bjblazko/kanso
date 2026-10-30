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
- Deployment guide for Authelia (`docs/deploy-authelia.md`), including the claims policy Authelia ≥ 4.39 needs.
- `flake.nix`: Nix package (`buildGoModule`, Go 1.27.3, tests run with git).
- Development data: `just seed` fills an empty `./tmp` with made-up people, a group, three spaces with different roles and sample pages; `just reset` deletes it. `kanso seed` refuses to run without the dev login.
- Dev login: one click per made-up person, and "Switch person" in the user menu returns to the same page.
- Notes (ADR-004): private notes per person, outside the Git history. Quick capture page (`/notes/new`) with camera and photo library; list, view, edit with source and preview, delete; turn a note into a wiki page; search finds your own notes; the seed adds some.
- Offline capture: notes and photos wait on the device (IndexedDB) and upload when the network is back; a lost answer never creates a note twice; a signed-out device keeps its notes and offers to sign in.
- Installable web app (manifest, own icon) whose home-screen icon opens the capture page.
- To-do lists (ADR-006): flat lists with one level of sub-items, outside the Git history; ticked items stay in place and fade; overview by last change; the owner shares with people or groups, renames and deletes; ticking and editing work offline and sync when the network is back.
- Account menu: your picture (round) or a placeholder with your name opens it; inside, who you are, language as two buttons (no "Apply"), group management and signing out, each with an icon. A profile picture named by the sign-in (`picture`, Authelia 4.39 can send it) is copied once per sign-in and served by Kanso.
- Groups can be deleted; members who were never signed in are listed by their id.
- Every page says who can see it (ADR-007): spaces, pages and the space list name the people with access plus admins; notes say only you; to-do lists name their owner and shares.
- To-do lists show progress: a bar with percent between the heading and the items; the overview shows a short bar per list, whose shape tells open, partly done and done apart, green when done.
- Tasks: one private board per person and boards per group, each task a Markdown file in the Git history (ADR-008).
- Tasks part in the rail: a board with the tabs Kanban (Offen · In Arbeit · Wartet · Erledigt; done tasks older than 14 days on request) and Matrix (Sofort, Planen, Kurz erledigen, Später); cards show due words, what a task waits for and who took it.
- A task's page: Markdown text, column, urgent/important, due date, what it waits for, the note it came from (for its author); edit with preview, take or give back a task of a group board, move it to another board, delete it with confirmation. Saving never overwrites a change made meanwhile.

### Security
- GPS location is removed from JPEG photos on upload (notes and wiki pages); orientation and other EXIF data stay.
- Requires Go 1.27.3 (`go.mod`), which fixes GO-2026-6809 in `crypto/x509`.

### Changed
- Navigation (ADR-005): a rail names the parts of Kanso — Notizen, Todos and Wiki — at the bottom on phones and at the left on wide screens; the side bar shows only the wiki's spaces and pages, under "Wiki-Bereiche"; group management moved to the user menu.
- The page tree shows guide lines for every level, so nested pages stay apart on narrow screens.
- Keeping a folder out of the history is one helper (`Repo.Ignore`) for notes and to-do lists.
- Uploads share one path (`store.SaveUpload`) for safe names, size limit and photo cleaning.
- The Markdown editor serves pages and notes; it reads its URLs from the form.

### Fixed
- Signing in failed with "Bad Request" when a second login started meanwhile (the service worker refreshing a page while signed out); each login now keeps its own cookie.
- Creating a group named as people write it ("Familie", "Großeltern") failed with a bare "Bad Request"; such names are allowed now, and a refused change says why on the page. Failures to save are logged and answered as server errors.
- Sub-items of task lists keep their indent.
- Phones: the bottom bar no longer jumps while Safari's toolbar shows or hides; page heights use `svh`.
- Task lists with blank lines between items no longer show bullets next to the checkboxes.
- On phones the folded menu no longer stretches into an empty grey block on short pages.
- Page history no longer shows commits of another page: Kanso follows real renames itself instead of `git log --follow`, which took look-alike new pages for copies.
- Sign-in no longer fails for names with quotes or angle brackets (they are dropped from the Git author only).
- Design review: button links no longer underlined, navigation folded on phones, neutral checkboxes.
- ECDSA keys parsed with the non-deprecated standard-library API.
- Documentation: arc42 architecture, ADRs 000–003, feature descriptions.
