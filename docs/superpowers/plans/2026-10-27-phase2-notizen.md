# Phase 2 – Notizen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Private notes that are captured on the phone in seconds, also offline and with photos,
and sorted later by hand: delete, or turn into a wiki page.

**Architecture:** A new domain package `internal/notes` stores notes as Markdown folders under
`notes/<sub>/<id>/` in the data directory, **outside Git** (`/notes/` in the data repo's
`.gitignore`). `internal/photo` removes GPS data from JPEGs. The wiki gains `Adopt` to turn a note
folder into a page (first commit then). The web layer adds `/notes/…` pages; a hand-written service
worker, an IndexedDB outbox and a web app manifest make `/notes/new` work offline.

**Tech Stack:** Go 1.27.3 stdlib, goldmark v2, go.yaml.in/yaml/v3; plain HTML/CSS/JS; Playwright.

**Spec:** `docs/superpowers/specs/2026-10-27-notes-design.md` (built on
`docs/superpowers/specs/2026-10-26-kanso-design.md`)

**Branch:** `phase2-notizen` off `main`.

## Global Constraints

- Every source file starts with `SPDX-License-Identifier: Apache-2.0`.
- No new dependency (Go or JS). Service worker, IndexedDB, manifest, icons, EXIF: own code.
- Functions ~30 lines, cyclomatic ≤ 10; packages by domain under `internal/<capability>`.
- Git is called as a program with an argument list, never via a shell. Notes are never committed.
- CSP stays without inline scripts. `img-src` gains `blob:` (photo previews) — nothing else.
- Only the author sees, edits or deletes a note. Admins get 404 like everyone else.
- All UI text in `internal/i18n/de.yaml` and `en.yaml` (the key-parity test must stay green).
- UI follows `huepattl-rams-design` and `docs/design.md`; one primary action per screen.
- Upload limit 50 MiB per file (`maxAttachment`), originals are never scaled.
- Note IDs: `store.NewID` format, exactly `^[a-z2-7]{12}$`; client-supplied IDs are validated.
- Docs in the same commit as the change: CHANGELOG, ADR-004, arc42, `docs/features/2026-10-26 notes.md`.
- Unit tests first (TDD); Playwright spec for every user-facing flow.

## Review Focus

1. **Response lost after the server stored the note** (phone loses network mid-upload): the retry
   with the same ID must answer "exists", never create a second note. → Task 3, Task 8.
2. **Session expired while notes wait offline:** the outbox keeps them, the page says to sign in,
   and after signing in they upload. → Task 8 (Playwright).
3. **Client-supplied note IDs and file names are attacker input:** `../x`, upper case, 13 chars,
   empty name, `..` in file names → 400 / safe name, never a path outside the note. → Task 3.
4. **Photos that are not clean JPEGs** (HEIC, PNG, truncated JPEG, EXIF without GPS, big-endian
   EXIF): stored unchanged or cleaned, never a panic, never a lost upload. → Task 2.
5. **Promoting a note with photos:** the new page shows the photos (links `attachments/…` keep
   working) and the note is gone from `/notes`. → Task 7.

---

### Task 1: Shared attachment naming and render base

Refactor without behaviour change, so notes can reuse wiki pieces.

**Files:**
- Create: `internal/store/names.go` (+ move tests from `internal/wiki/attachments_test.go` that test naming)
- Modify: `internal/wiki/attachments.go`, `internal/markdown/render.go`, `internal/web/handlers_pages.go`,
  `internal/web/handlers_history.go`, `internal/markdown/render_test.go`

**Interfaces:**
- Produces: `store.AttachmentName(upload string) string`, `store.UniqueFile(dir, name string) string`
  (moved from wiki, unchanged behaviour); `markdown.PageCtx{Base string}` replacing `PageID` —
  attachment links `attachments/x` become `Base + "/attachments/x"`; pages pass `Base: "/p/" + id`.

- [ ] **Step 1:** Move the naming tests to `internal/store/names_test.go`, calling `store.AttachmentName`
  and `store.UniqueFile`; change `render_test.go` to `PageCtx{Base: "/p/abc"}`. Run
  `go test ./internal/store ./internal/markdown` → FAIL (undefined).
- [ ] **Step 2:** Move the functions; wiki calls the store versions; rename the field and its callers.
- [ ] **Step 3:** `go test ./...` → all PASS.
- [ ] **Step 4:** Commit `refactor: share attachment naming, render base path`.

### Task 2: Remove GPS from JPEG photos

**Files:**
- Create: `internal/photo/gps.go`, `internal/photo/gps_test.go`
- Modify: `internal/wiki/attachments.go` (clean wiki uploads too — same anti-goal: no location in
  pages the family sees), `internal/wiki/attachments_test.go`

**Interfaces:**
- Produces: `photo.StripGPS(b []byte) []byte` — returns b unchanged unless it is a JPEG whose EXIF
  has a GPS IFD; `photo.Clean(name string, r io.Reader, max int64) (io.Reader, error)` — for names
  ending `.jpg`/`.jpeg` (any case) reads up to max+1 bytes and returns the stripped bytes; other
  names pass r through. Over max → `photo.ErrTooLarge`; callers map it to their own error.

- [ ] **Step 1: Tests** (build EXIF in a test helper `jpegWithExif(order binary.ByteOrder, tags …)`
  producing SOI, APP1 `Exif\0\0` + TIFF with IFD0 {Orientation=6, GPSInfo→GPS IFD with
  GPSLatitude rational 48/1, 8/1, 1234/100}, then a minimal SOS/EOI):
  - `TestStripGPSRemovesLocationKeepsOrientation` (little- and big-endian): result has no IFD0
    entry 0x8825, Orientation still 6, the latitude rational bytes no longer occur anywhere in the
    output, output length == input length.
  - `TestStripGPSLeavesOtherFilesAlone`: PNG signature bytes, JPEG without APP1, JPEG with EXIF
    but no GPS, truncated JPEG (cut at 3, 20, 40 bytes), random bytes → returned unchanged, no panic.
  - `TestCleanOnlyTouchesJPEGNames`: `x.PNG` passes through; `x.JPG` is stripped; over max →
    `ErrTooLarge`.
  - Wiki: `TestAttachmentLosesGPS` — `AddAttachment` with the GPS JPEG stores a file without the
    latitude bytes.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. Algorithm (in place, length unchanged): walk JPEG segments from SOI
  until SOS; in the first APP1 starting `Exif\0\0`, read the TIFF byte order and IFD0; find entry
  0x8825; bounds-check every offset against the APP1 segment; zero the GPS IFD (2 + 12·n + 4
  bytes) and every out-of-line value it points to; remove the 0x8825 entry from IFD0 by shifting the
  following entries and the next-IFD offset 12 bytes up, decrement the count, zero the freed 12
  bytes. Any bounds failure → return the input unchanged.
- [ ] **Step 4:** `go test ./internal/photo ./internal/wiki` → PASS.
- [ ] **Step 5:** Commit `feat(photo): remove GPS location from JPEG uploads`.

### Task 3: Notes store

**Files:**
- Create: `internal/notes/notes.go`, `internal/notes/attachments.go`, `internal/notes/notes_test.go`

**Interfaces:**
- Consumes: `store.NewID`, `store.ReadDoc/WriteDoc/WriteAtomic`, `store.AttachmentName`,
  `store.UniqueFile`, `photo.Clean`, `store.Repo.Commit`.
- Produces:
  ```go
  type Meta struct { ID string `yaml:"id"`; Created time.Time `yaml:"created"`; Updated time.Time `yaml:"updated"` }
  type Note struct { Owner string; Meta Meta; Body string; Attachments []string }
  type Upload struct { Name string; R io.Reader }
  var ErrNotFound, ErrBadID, ErrConflict, ErrTooLarge error
  func NewStore(dataDir string, repo *store.Repo) (*Store, error) // ensures "/notes/" in .gitignore, committed once
  func (s *Store) Create(owner, id, body string, files []Upload, max int64) (Note, bool, error) // bool = created; id "" → new ID
  func (s *Store) Get(owner, id string) (Note, error)
  func (s *Store) List(owner string) []Note                          // newest Created first
  func (s *Store) Save(owner, id, body string, version time.Time) (Note, error) // ErrConflict if Updated != version
  func (s *Store) AddAttachment(owner, id, filename string, r io.Reader, max int64) (string, error)
  func (s *Store) OpenAttachment(owner, id, name string) (*os.File, error)
  func (s *Store) Delete(owner, id string) error
  func (s *Store) Dir(owner, id string) (string, error)              // absolute folder, for Adopt
  func (s *Store) All() []Note                                       // every owner, for the search index
  func Title(n Note) string // first non-empty line, leading "#"s, "- ", "* " and spaces stripped, max 80 runes; "" if none
  OnChange func(n Note, deleted bool)                                // field on Store
  ```
  Create appends each photo as `![](attachments/<name>)` on its own line at the end of the body.
  Owner must pass the users sub pattern `^[A-Za-z0-9._@-]{1,128}$`, else `ErrBadID`.

- [ ] **Step 1: Tests:**
  - `TestCreateAndGet`, `TestCreateIsIdempotent` (same owner+id twice → second returns `false`,
    body of the first stays, one folder), `TestCreateRejectsBadIDs` (`../x`, `ABCDEFGHIJKL`,
    `abc`, 13 chars → `ErrBadID`), `TestCreateWithPhotosLinksThem`,
    `TestFileNamesStayInside` (`../../evil.jpg` → stored as `evil.jpg` inside `attachments/`).
  - `TestOtherOwnersSeeNothing` (Get/List/Delete for another owner → `ErrNotFound`/empty).
  - `TestSaveDetectsConflict` (stale version → `ErrConflict`, file unchanged).
  - `TestNotesStayOutOfGit`: after Create, `git status --porcelain` in the data dir is empty;
    `.gitignore` contains `/notes/` exactly once after `NewStore` ran twice.
  - `TestDeleteRemovesFolder`, `TestListNewestFirst`, `TestTitle` (table: `"\n\n# Einkauf\nx"` →
    `Einkauf`; `"- Milch"` → `Milch`; 100 × `a` → 80 runes; only images → `""`).
- [ ] **Step 2:** `go test ./internal/notes` → FAIL.
- [ ] **Step 3:** Implement; writes go through a mutex in `Store`; files 0640, dirs 0750.
- [ ] **Step 4:** PASS; `go vet ./...`.
- [ ] **Step 5:** Commit `feat(notes): private notes store outside Git`.

### Task 4: Search covers own notes

**Files:**
- Modify: `internal/search/index.go`, `internal/search/index_test.go`, `internal/web/handlers_search.go`,
  `web/templates/search.html`, i18n

**Interfaces:**
- Consumes: `notes.Store.All`, `notes.Store.OnChange`, `notes.Title`.
- Produces: `search.Doc{ID, Space, Owner, Title, Body}` and `Hit.Owner`; `Search(q string,
  allowed func(d Doc) bool, limit int)`; `web.BuildIndex(w *wiki.Wiki, n *notes.Store)`. A note doc
  has `Space ""`, `Owner sub`; a page has `Owner ""`.

- [ ] **Step 1: Tests:** search: `TestAllowedSeesDoc` (filter gets the doc); web:
  `TestSearchFindsOwnNotesOnly` — anna's note "Zählerstand Keller" is found by anna under the
  heading "Notizen", not by ben, not by an admin.
- [ ] **Step 2:** FAIL. **Step 3:** Implement; results template links notes to `/notes/<id>`.
- [ ] **Step 4:** PASS. **Step 5:** Commit `feat(search): own notes in search results`.

### Task 5: Notes pages – list, view, capture without JS, delete

**Files:**
- Create: `internal/web/handlers_notes.go`, `internal/web/handlers_notes_test.go`,
  `web/templates/notes.html`, `note_view.html`, `note_new.html`, `note_delete.html`,
  `e2e/tests/notes.spec.js`
- Modify: `internal/web/server.go` (Deps.Notes, routes), `cmd/kanso/main.go`,
  `internal/web/server_test.go` (buildTestDeps), `web/templates/layout.html` (nav entry "Notizen"
  first in the sidebar), `web/templates/icons.html` (icon `note`), `web/static/css/app.css`, i18n

**Interfaces:**
- Consumes: Task 3 store, `markdown.PageCtx{Base: "/notes/" + id}`.
- Produces routes: `GET /notes`, `GET /notes/new`, `POST /notes` (multipart: `id` optional,
  `body`, `files` repeated; JSON when `Accept: application/json`: 201 `{"id","status":"created"}`,
  200 `{"id","status":"exists"}`, 400 `{"error":"invalid"}`, 413 `{"error":"too_large"}`; without
  JSON: 303 to `/notes/new?saved=1`), `GET /notes/{id}`, `GET /notes/{id}/attachments/{name}`
  (same headers as wiki attachments), `GET|POST /notes/{id}/delete`.
- Capture page markup (Task 8 builds on it): `<form id="capture" method="post" action="/notes"
  enctype="multipart/form-data">`, `textarea#note-body name=body`, `input#camera type=file
  name=files accept="image/*" capture="environment"`, `input#pick type=file name=files multiple
  accept="image/*,application/pdf"`, both shown as labelled buttons "Kamera" / "Foto wählen",
  `ul#previews`, `p#capture-status role=status`, primary button "Speichern".
  A note without text and photos is refused (400 / message).
- Empty state on `/notes`: "Noch keine Notizen." + primary "Neue Notiz".
- Title in lists: `notes.Title`, else "Foto vom 27.10.2026, 14:03" (with photos) or "Leere Notiz".

- [ ] **Step 1: Go HTTP tests:** `TestCaptureWithoutJS` (multipart POST with body + PNG → 303,
  listed on `/notes`, image served with `Content-Type: image/png`), `TestCaptureJSONIsIdempotent`,
  `TestEmptyNoteRefused`, `TestNotesArePrivate` (ben and admin get 404 on anna's note, its
  attachment and delete), `TestDeleteNote`, `TestNotesNavEntry`.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. Rams: capture page has one job; on phones the
  textarea fills the screen height minus the button row.
- [ ] **Step 4:** PASS. **Step 5: Playwright** `notes.spec.js`: write a note via the page (no
  offline yet), it appears in `/notes`, open, delete with confirmation. Run `just e2e` → PASS.
- [ ] **Step 6:** Commit `feat(notes): list, view, capture and delete notes`.

### Task 6: Edit notes with the wiki editor

**Files:**
- Create: `web/templates/note_edit.html`
- Modify: `web/static/js/editor.js`, `web/templates/page_edit.html`, `internal/web/handlers_notes.go`,
  `internal/web/handlers_pages.go` (preview), tests, `e2e/tests/notes.spec.js`, i18n

**Interfaces:**
- editor.js reads its URLs from the form: `data-save-url`, `data-upload-url`, `data-view-url`,
  `data-preview-base`; heartbeat only when the form has a `lock` field; 409 with
  `{"error":"conflict"}` shows `data-msg-conflict` and keeps the text.
- Routes: `GET /notes/{id}/edit`, `POST /notes/{id}` (fields `body`, `version` = RFC 3339 nano of
  `Updated`), `POST /notes/{id}/attachments` (JSON `{"name","markdown"}` like pages).
- `POST /api/preview` accepts `base` = `/p/<id>` or `/notes/<id>`; anything else → 400.

- [ ] **Step 1: Tests:** `TestEditNoteSaves`, `TestEditNoteConflict` (two saves with the same
  version → second 409), `TestPreviewBaseValidated`; existing page editor tests stay green.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. **Step 4:** PASS, plus `editor.spec.js` unchanged
  green; add Playwright "edit a note and add a photo". **Step 5:** Commit
  `feat(notes): edit notes with source and preview`.

### Task 7: Note → wiki page

**Files:**
- Create: `internal/wiki/adopt.go`, `internal/wiki/adopt_test.go`, `web/templates/note_promote.html`
- Modify: `internal/web/handlers_notes.go`, tests, `e2e/tests/notes.spec.js`, i18n

**Interfaces:**
- Consumes: `notes.Store.Dir`, `notes.Store.Delete`, `notes.Title`.
- Produces: `func (w *Wiki) Adopt(a store.Author, space, parentID, title, srcDir string) (Page, error)`
  — moves srcDir (same filesystem) to a new unique slug folder below parent, rewrites `index.md`
  with a fresh `PageMeta` (new ID, order last, author, created = now) keeping the body, commits
  `Seite aus Notiz angelegt: <title>`, updates the index and calls `OnChange`.
- Routes: `GET /notes/{id}/promote` (title prefilled; one select "Wo" with options per writable
  space and each of its pages, indented, value `<space>/<parentID>`), `POST /notes/{id}/promote` →
  303 to `/p/<new id>`. Space role < Editor → 403.

- [ ] **Step 1: Tests:** `TestAdoptMovesFolderAndCommits` (page readable by ID, attachments listed,
  one commit with the message, source folder gone), `TestPromoteNeedsEditor` (viewer of the space
  → 403, note unchanged), `TestPromotedPhotoStillShows` (GET `/p/<new>/attachments/<name>` → 200,
  rendered page contains that URL), note missing from `/notes` and the search.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. **Step 4:** PASS; Playwright "turn a note into a wiki
  page". **Step 5:** Commit `feat(notes): turn a note into a wiki page`.

### Task 8: Offline capture (service worker, outbox, manifest, icons)

**Files:**
- Create: `web/static/js/capture.js`, `web/templates/sw.js.tmpl`, `internal/web/handlers_pwa.go`,
  `internal/web/handlers_pwa_test.go`, `scripts/gen-icons/main.go`, `web/static/icons/`
  (`icon.svg`, `icon-192.png`, `icon-512.png`, `apple-touch-icon.png`), `e2e/tests/offline.spec.js`
- Modify: `internal/web/server.go` (routes outside RequireUser: `GET /sw.js`,
  `GET /manifest.webmanifest`; CSP `img-src 'self' data: blob:`), `web/templates/layout.html`
  (`<link rel="manifest">`, `apple-touch-icon`, `theme-color`), `note_new.html`,
  `internal/web/server_test.go` (CSP), licenses manifest if icons need listing (own work: no),
  i18n

**Interfaces:**
- `GET /notes/token` (inside RequireUser) → `{"csrf": "<token>"}`; when signed out the client sees
  a redirect/401 and shows "Zum Hochladen bitte anmelden" with a link to `/auth/login?next=/notes/new`.
- `/sw.js`: `Cache-Control: no-cache`, `Service-Worker-Allowed: /`; cache name `kanso-<version>`;
  install precaches the fingerprinted CSS, fonts, `capture.js`, icons and `/notes/new` (stored only
  if `ok && !redirected`); navigation to `/notes/new` is network-first with cache fallback; every
  other request goes to the network untouched; activate deletes other `kanso-*` caches and claims
  clients.
- Manifest: `name "Hüpattl! Kanso"`, `short_name "Kanso"`, `start_url "/notes/new"`,
  `display "standalone"`, `background_color`/`theme_color` from the light tokens, icons 192/512
  (`purpose "any maskable"`).
- capture.js: IndexedDB `kanso`, store `outbox` (`{id, body, files: [{name, type, blob}],
  created}`); client ID = 12 chars from `abcdefghijklmnopqrstuvwxyz234567` via
  `crypto.getRandomValues`; on submit: put → clear form → status → sync. Sync on submit, page
  load, `online`, and every 30 s while visible; one note at a time; delete from the outbox only on
  201/200; 400/413 mark the entry failed and keep it (status lists it with its title); calls
  `navigator.storage.persist()` once. The waiting count is a `<details>` whose list shows the
  titles of waiting notes (spec: "Anzahl, auf Tipp die Titel"). Status texts: "Gespeichert", "Gespeichert · {n} warten auf
  Netz", "{n} warten auf Netz", "Zum Hochladen bitte anmelden", "{n} konnten nicht hochgeladen
  werden". Previews via `URL.createObjectURL`, each with "Entfernen".
- Icon: own geometric mark drawn by `scripts/gen-icons` with `image/png` (dark square, light
  note lines, one orange dot = the accent); the generator is committed, the PNGs too.

- [ ] **Step 1: Go tests:** `TestServiceWorkerHeaders`, `TestServiceWorkerListsAssets` (every
  precached URL returns 200), `TestManifest` (valid JSON, start_url, icons exist),
  `TestTokenEndpoint`, CSP test updated.
- [ ] **Step 2:** FAIL. **Step 3:** Implement. **Step 4:** PASS.
- [ ] **Step 5: Playwright** `offline.spec.js` (Chromium only; `test.skip` on WebKit with the
  reason): (a) open `/notes/new`, wait for `navigator.serviceWorker.ready`, `setOffline(true)`,
  reload → page loads; write a note with a photo (`setInputFiles` from a Buffer) → status "1 wartet
  auf Netz"; `setOffline(false)` → "Gespeichert", note and photo on `/notes`. (b) Review Focus 1:
  route-intercept the first POST to abort after the server answered (`route.fetch()` then
  `route.abort()`) → one note only after the retry. (c) Review Focus 2: clear cookies, go offline,
  save, online → "Zum Hochladen bitte anmelden"; sign in via the link → note uploaded.
- [ ] **Step 6:** Commit `feat(notes): capture offline with photos, installable`.

### Task 9: Seed, docs, reviews

**Files:**
- Modify: `internal/devseed/seed.go` (+test: two notes for anna, one for ben),
  `docs/adr/004-notes-outside-git-offline-capture.md` (new), `docs/architecture/arc42.md`
  (building blocks `notes`, `photo`; runtime "offline capture and sync"; risks: browser storage can
  be evicted, iOS keyboard; ADR list), `docs/features/2026-10-26 notes.md` (status done, updated,
  Related: ADR-004, spec, plan), `CHANGELOG.md`, `README.md` (privacy: notes not in Git and
  deleted for good; notes wait in the browser until uploaded; GPS removed from JPEGs), `docs/design.md`
  (capture screen job)

- [ ] **Step 1:** Seed test → FAIL → implement → PASS.
- [ ] **Step 2:** Docs as listed.
- [ ] **Step 3:** `huepattl-rams-design` review of `/notes/new`, `/notes`, promote (phone and
  desktop screenshots via Playwright); fix findings with e2e coverage.
- [ ] **Step 4:** `huepattl-legal-check` (no new dependency; new browser storage; own icons) →
  findings table in the final report.
- [ ] **Step 5:** `go test ./...`, `golangci-lint run ./...` (0 issues), `just e2e` → all PASS.
- [ ] **Step 6:** Commit `docs: notes`.
