# Phase 1: Fundament + Wiki – Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A runnable `kanso` binary: Authelia login (OIDC), admin/owner/editor/viewer per space, Kanso-managed groups, spaces with a page tree, Markdown pages with source/preview editing, attachments, Mermaid, edit locks, Git history with diff/restore, search, German + English UI in the Rams design.

**Architecture:** One Go binary, frontend embedded via `embed`, server-rendered `html/template` pages plus small vanilla JS modules. All content lives in a data directory that is a Git repository (one commit per save, author = user); runtime-only state (sessions, VAPID keys later) lives in a separate state directory. Packages are cut by domain (`internal/<capability>`), not by layer.

**Tech Stack:** Go 1.27 (stdlib), `github.com/yuin/goldmark/v2` v2.1.9, `go.yaml.in/yaml/v3`, `git` CLI, mermaid 12.2.0 ESM build without ELK (vendored), IBM Plex Sans/Mono WOFF2 (vendored), plain HTML/CSS/JS.

**Spec:** `docs/superpowers/specs/2026-10-26-kanso-design.md`

## Global Constraints

- Go module path `github.com/bjblazko/kanso`; `go 1.27` in `go.mod`. License Apache-2.0 (SPDX header `// SPDX-License-Identifier: Apache-2.0` in every source file).
- Allowed third-party code: only the four approved dependencies above. Anything else needs the dependency check and the user's approval first.
- No telemetry, no update check, no CDN. Outbound HTTP only to the configured OIDC issuer.
- Roles: global `admin` = member of OIDC group `kanso-admin` (claim `groups`, re-read on each login, never stored as authority). Per space `owner` > `editor` > `viewer`, granted to users or Kanso groups; highest wins; no role → space invisible (404, not 403).
- Groups are defined in Kanso (`data/kanso.yaml`), not taken from Authelia.
- Edit lock per page: expires 5 min after the last heartbeat; owner/admin can break it; held in memory only.
- Every save = one Git commit, author `Name <email>` of the acting user. Git is called as a program with an argument list, never through a shell.
- UI: `huepattl-rams-design` (tokens copied verbatim from `~/.claude/skills/huepattl-rams-design/references/tokens.css`); UI copy in German and English, per-user language, default from `Accept-Language`, fallback `de`.
- Markdown: goldmark without raw HTML (`html.WithUnsafe` never set); links/images only `http`, `https`, `mailto`, relative, and `/p/<id>`.
- Mermaid: vendored ESM build, ELK chunk removed, `layout: 'dagre'`, `securityLevel: 'strict'`, loaded only on pages containing a mermaid block.
- Security headers on every response: `Content-Security-Policy: default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; object-src 'none'; base-uri 'none'; frame-ancestors 'none'` (`unsafe-inline` for styles only, required by Mermaid's SVG), `X-Content-Type-Options: nosniff`, `Referrer-Policy: same-origin`.
- Code quality: functions ~30 lines, cyclomatic ≤ 10 (`huepattl-code-quality`); measure with `gofuncmetrics` fallback if `golangci-lint` is absent.

## Review Focus

1. **Path traversal / hostile names** – slugs, attachment filenames and URL segments like `../`, `%2e%2e`, absolute paths, NUL, Windows names; expected: rejected or sanitised, never leaving the data dir. Tests in Task 2 and Task 13.
2. **Umlauts and odd titles** – „Über uns“, „Größe & Gewicht“, emoji-only, 300-char titles, duplicate titles in one folder; expected: readable unique slug (`ueber-uns`, `-2` suffix), title kept verbatim in frontmatter. Test in Task 11.
3. **Saving after the lock was lost** – lock expired or broken while editing; expected: save refused with 409 and the text kept in the browser, never overwriting the other person. Test in Task 14.
4. **Access withdrawn mid-session** – role or group membership removed while logged in; expected: next request denied (roles resolved per request, not cached in the session). Test in Task 10.
5. **XSS via content** – `[x](javascript:alert(1))`, raw `<script>` in Markdown, HTML in titles, Mermaid labels with HTML; expected: rendered inert. Tests in Task 12 and Task 16.

---

## File Structure

```
cmd/kanso/main.go                 # flags/env → config, wiring, ListenAndServe
internal/config/config.go         # Config struct + Load
internal/store/path.go            # safe path join, slugify
internal/store/file.go            # atomic write, frontmatter read/write
internal/store/git.go             # Repo: Init, Commit, Log, Show, Diff, Move
internal/access/model.go          # Role, Member, Groups, Resolve, Can
internal/access/config.go         # load/save data/kanso.yaml (groups)
internal/users/users.go           # User, profiles in data/users/<sub>.yaml
internal/auth/oidc.go             # discovery, PKCE, code exchange
internal/auth/jwt.go              # JWKS, ID token verification
internal/auth/session.go          # session store (state dir), cookies
internal/auth/middleware.go       # RequireUser, CSRF, dev login
internal/i18n/i18n.go + de.yaml, en.yaml
internal/spaces/spaces.go         # Space CRUD, members (space.yaml)
internal/wiki/tree.go             # page tree + id index
internal/wiki/pages.go            # create/update/move/delete pages
internal/wiki/locks.go            # edit locks
internal/wiki/history.go          # log/show/diff/restore via store.Repo
internal/markdown/render.go       # goldmark config + URL rewriting
internal/search/index.go          # in-memory index
internal/web/server.go            # routes, headers, templates
internal/web/handlers_*.go        # one file per domain: spaces, pages, admin, search
web/templates/*.html, web/static/{css,js,fonts,vendor}/
web/static/licenses/              # license texts for "Licenses and thanks"
scripts/vendor-mermaid.sh, scripts/vendor-plex.sh
```

---

### Task 1: Repository skeleton and config

**Files:**
- Create: `go.mod`, `LICENSE` (Apache-2.0 full text), `NOTICE`, `README.md` (EN, short: what, run, configure), `CHANGELOG.md` (Keep a Changelog, `[Unreleased]`), `CLAUDE.md` (project rules: license, dependency policy, skills to use, doc conventions as in homelab2), `docs/adr/000-filesystem-and-git-storage.md`, `docs/adr/001-oidc-with-kanso-managed-roles.md`, `.gitignore`, `justfile` (`run`, `test`, `lint`, `vendor`)
- Create: `internal/config/config.go`, `cmd/kanso/main.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `type Config struct { Addr, DataDir, StateDir, BaseURL, OIDCIssuer, OIDCClientID, OIDCClientSecret, AdminGroup string; DevLogin bool }`; `func Load(getenv func(string) string) (Config, error)`. Env names `KANSO_ADDR` (default `127.0.0.1:8090`), `KANSO_DATA_DIR`, `KANSO_STATE_DIR`, `KANSO_BASE_URL`, `KANSO_OIDC_ISSUER`, `KANSO_OIDC_CLIENT_ID`, `KANSO_OIDC_CLIENT_SECRET` (or `KANSO_OIDC_CLIENT_SECRET_FILE`), `KANSO_ADMIN_GROUP` (default `kanso-admin`), `KANSO_DEV_LOGIN` (`true` only accepted when `Addr` host is loopback).

- [ ] **Step 1: Tests** – `TestLoadDefaults` (only data/state/base set → `Addr == "127.0.0.1:8090"`, `AdminGroup == "kanso-admin"`); `TestLoadRequiresOIDCUnlessDevLogin` (no issuer, no dev → error); `TestDevLoginRejectedOnPublicAddr` (`KANSO_ADDR=0.0.0.0:8090`, dev true → error); `TestSecretFromFile` (temp file content trimmed).
- [ ] **Step 2:** `go test ./internal/config/` → FAIL (undefined Load).
- [ ] **Step 3:** Implement `Load`; `main.go` loads config, serves `GET /healthz` → `200 ok`, exits non-zero with the error message on bad config.
- [ ] **Step 4:** `go test ./... && go vet ./...` → PASS; `go run ./cmd/kanso` with dev env → `curl -s localhost:8090/healthz` prints `ok`.
- [ ] **Step 5:** `git init` exists already; commit `feat: repository skeleton and configuration`.

### Task 2: Safe paths and files

**Files:** Create `internal/store/path.go`, `internal/store/file.go`; Test `internal/store/path_test.go`, `internal/store/file_test.go`

**Interfaces:**
- Produces: `func Join(root string, parts ...string) (string, error)` – cleaned path inside `root`, error `ErrOutside` otherwise; `func Slugify(title string) string`; `func UniqueSlug(dir, slug string) string`; `func NewID() string` (12 chars, base32 lowercase, crypto/rand); `func WriteAtomic(path string, data []byte) error` (temp file + fsync + rename, creates parents 0750/0640); `type Doc[M any] struct { Meta M; Body string }`; `func ReadDoc[M any](path string) (Doc[M], error)`; `func WriteDoc[M any](path string, d Doc[M]) error` (`---\n<yaml>---\n\n<body>`); `func ReadYAML(path string, v any) error`; `func WriteYAML(path string, v any) error`.

- [ ] **Step 1: Tests** – `Join` rejects `../x`, `a/../../x`, `/etc/passwd`, `a\x00b`, and accepts `a/b`; `Slugify("Über uns") == "ueber-uns"`, `Slugify("Größe & Gewicht") == "groesse-gewicht"`, `Slugify("🙂") == "seite"`, `Slugify(strings.Repeat("a",300))` has length 80; `UniqueSlug` returns `x-2` when `x` exists; `NewID` 12 chars `[a-z2-7]`; doc round-trip keeps body bytes and meta; `ReadDoc` on file without frontmatter returns empty meta + full body.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement. Slugify: lower-case, ä→ae ö→oe ü→ue ß→ss, other non `[a-z0-9]` → `-`, collapse, trim, max 80, empty → `seite`.
- [ ] **Step 4:** `go test ./internal/store/` → PASS.
- [ ] **Step 5:** Commit `feat(store): safe paths, slugs, atomic documents`.

### Task 3: Git repository wrapper

**Files:** Create `internal/store/git.go`; Test `internal/store/git_test.go`

**Interfaces:**
- Produces: `type Author struct{ Name, Email string }`; `type Repo struct` (dir, mutex); `func OpenRepo(dir string) (*Repo, error)` (runs `git init -b main` and an initial commit if no `.git`; sets local `user.name=Kanso`, `user.email=kanso@localhost` as committer); `func (r *Repo) Commit(a Author, msg string, paths ...string) error` (`git add -A -- paths…` then `git commit --author`; no-op when nothing staged); `func (r *Repo) Move(a Author, from, to, msg string) error`; `type Revision struct{ Hash string; Author Author; Time time.Time; Message string }`; `func (r *Repo) Log(path string, limit int) ([]Revision, error)` (`--follow`); `func (r *Repo) Show(hash, path string) ([]byte, error)`; `func (r *Repo) Diff(from, to, path string) (string, error)` (unified).
- All writes serialised by `Repo`'s mutex; every caller writes files *and* commits inside `r.Locked(func() error)` → also produce `func (r *Repo) Locked(fn func() error) error`.

- [ ] **Step 1: Tests** (skip with `t.Skip` if `git` missing): commit two versions of `a.md` with different authors → `Log` returns 2 revisions newest first with correct author names; `Show(oldHash,"a.md")` returns old content; `Diff` contains `-old` and `+new`; `Move` keeps `Log` history of the new path; commit with unchanged file returns nil and adds no revision; author name containing `"` or newline is rejected.
- [ ] **Step 2:** Run → FAIL.
- [ ] **Step 3:** Implement with `exec.CommandContext("git", "-C", dir, …)`, env `GIT_TERMINAL_PROMPT=0`, `LC_ALL=C`; parse log with `--format=%H%x1f%an%x1f%ae%x1f%aI%x1f%s%x1e`.
- [ ] **Step 4:** PASS.
- [ ] **Step 5:** Commit `feat(store): git history for the data directory`.

### Task 4: Access model

**Files:** Create `internal/access/model.go`, `internal/access/config.go`; Test `internal/access/model_test.go`

**Interfaces:**
- Produces: `type Role int` with `None < Viewer < Editor < Owner` and `String()`/`ParseRole`; `type Principal struct{ Sub string; Admin bool }`; `type Member struct{ User, Group string; Role Role }` (exactly one of User/Group; YAML `user:`/`group:`/`role:`); `type Groups map[string][]string` (group → subs); `func Resolve(p Principal, members []Member, g Groups) Role` (admin → Owner); `type Action int` (`Read, Write, ManageSpace, BreakLock`); `func Can(r Role, a Action) bool` (Read≥Viewer, Write≥Editor, ManageSpace/BreakLock≥Owner); `func LoadGroups(dataDir string) (Groups, error)` / `func SaveGroups(dataDir string, g Groups) error` (file `data/kanso.yaml`, key `groups`).

- [ ] **Step 1: Tests** – table test of `Resolve`: no entry → None; user viewer + group editor → Editor; admin with no entry → Owner; group with unknown sub → None. `Can` table for all 4×4. Groups round-trip via temp dir.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(access): roles, groups, permission resolution`.

### Task 5: User profiles

**Files:** Create `internal/users/users.go`; Test `internal/users/users_test.go`

**Interfaces:**
- Produces: `type User struct{ Sub, Name, Email, Lang string; LastLogin time.Time }` (YAML); `type Store struct`; `func NewStore(dataDir string, repo *store.Repo) *Store`; `func (s *Store) Upsert(u User) (User, error)` (keeps `Lang` if already set; commits `users/<sub>.yaml` with author = that user); `func (s *Store) Get(sub string) (User, bool)`; `func (s *Store) List() []User` (sorted by Name); `func (s *Store) SetLang(sub, lang string) error` (only `de`/`en`).
- File name: `Slugify`-safe form of sub; reject subs containing path separators.

- [ ] **Step 1: Tests** – upsert creates file and commit; second upsert keeps `Lang`; `SetLang("x","fr")` errors; `List` sorted.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(users): profiles stored as files`.

### Task 6: OIDC client (stdlib only)

**Files:** Create `internal/auth/oidc.go`, `internal/auth/jwt.go`; Test `internal/auth/oidc_test.go` (fake provider with `httptest.Server`, RSA + ECDSA keys generated in test)

**Interfaces:**
- Produces: `type Provider struct`; `func Discover(ctx context.Context, issuer, clientID, secret, redirectURL string, hc *http.Client) (*Provider, error)` (`/.well-known/openid-configuration`, requires `code` flow and `S256`); `type AuthRequest struct{ State, Nonce, Verifier string }`; `func NewAuthRequest() AuthRequest`; `func (p *Provider) AuthURL(r AuthRequest) string` (scopes `openid profile email groups`, PKCE S256); `type Claims struct{ Sub, Name, Email string; Groups []string }`; `func (p *Provider) Exchange(ctx context.Context, code string, r AuthRequest) (Claims, error)` (token endpoint with `client_secret_basic`, verifies ID token: signature via JWKS (`RS256`, `ES256`; reject `none`/HS*), `iss`, `aud` contains client id, `azp` if present, `exp`, `iat` ±2 min skew, `nonce`); JWKS cached, refetched once on unknown `kid`.

- [ ] **Step 1: Tests** – happy path RS256 and ES256 return claims incl. groups; each of these returns an error: wrong `iss`, wrong `aud`, expired, wrong nonce, `alg: none`, `alg: HS256`, unknown `kid` after refetch, tampered payload; `AuthURL` contains `code_challenge_method=S256` and the state.
- [ ] **Step 2–4:** FAIL → implement (`crypto/rsa.VerifyPKCS1v15`, `crypto/ecdsa.Verify` with r||s split, `encoding/base64.RawURLEncoding`) → PASS.
- [ ] **Step 5:** Commit `feat(auth): OIDC code flow with PKCE using the standard library`.

### Task 7: Sessions, middleware, CSRF, dev login

**Files:** Create `internal/auth/session.go`, `internal/auth/middleware.go`; Test `internal/auth/session_test.go`, `internal/auth/middleware_test.go`

**Interfaces:**
- Consumes: `Provider`, `Claims` (Task 6), `users.Store` (Task 5), `access.Principal` (Task 4).
- Produces: `type Session struct{ ID, Sub, CSRF string; Admin bool; Expires time.Time }`; `type Sessions struct` persisted to `<StateDir>/sessions.json` (0600), TTL 30 days sliding; `func NewSessions(stateDir string) (*Sessions, error)`; `Create(sub string, admin bool) Session`, `Get(id string) (Session, bool)`, `Delete(id string)`; `type Auth struct` with `func (a *Auth) Routes(mux *http.ServeMux)` → `GET /auth/login`, `GET /auth/callback`, `POST /auth/logout`, and when dev login enabled `GET/POST /auth/dev` (form: sub, name, admin checkbox); `func (a *Auth) RequireUser(next http.Handler) http.Handler` (redirect to login for GET, 401 otherwise); `func UserFrom(ctx context.Context) (users.User, access.Principal)`; `func CSRFToken(ctx context.Context) string`; CSRF check for all non-GET/HEAD: header `X-CSRF-Token` or form field `csrf` equal to session token **and** `Origin`/`Referer` host equal to `BaseURL` host.
- Cookies: `kanso_session` (HttpOnly, Secure unless BaseURL is http://localhost, SameSite=Lax, Path=/); auth-request state in a short-lived cookie `kanso_auth` (10 min) holding state/nonce/verifier, HMAC-signed with a key from `<StateDir>/cookie.key` (created 0600 on first start).
- Admin flag = `slices.Contains(claims.Groups, cfg.AdminGroup)`, refreshed on every login.

- [ ] **Step 1: Tests** – session survives `NewSessions` reload; expired session not returned; `RequireUser` redirects GET to `/auth/login?next=…` with a relative `next` only (absolute `next=https://evil` dropped); POST without CSRF → 403; POST with token but foreign Origin → 403; callback with mismatched state → 400; dev route returns 404 when disabled.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(auth): sessions, CSRF protection and dev login`.

### Task 8: i18n

**Files:** Create `internal/i18n/i18n.go`, `internal/i18n/de.yaml`, `internal/i18n/en.yaml`; Test `internal/i18n/i18n_test.go`

**Interfaces:**
- Produces: `type Bundle struct`; `func Load() (*Bundle, error)` (embedded files); `func (b *Bundle) T(lang, key string, args ...any) string` (falls back to `de`, then returns the key); `func Negotiate(acceptLanguage, userLang string) string` (`de`/`en`); template func name `t`.

- [ ] **Step 1: Tests** – `TestBundlesHaveSameKeys` (every key in de exists in en and vice versa); missing key returns key; `Negotiate("en-US,en;q=0.9","")=="en"`, `Negotiate("fr","")=="de"`, user lang wins.
- [ ] **Step 2–4:** FAIL → implement → PASS. Keys are added by later tasks to both files (the same-keys test enforces it).
- [ ] **Step 5:** Commit `feat(i18n): German and English UI strings`.

### Task 9: Web shell, design system, licenses page

**Files:**
- Create: `internal/web/server.go`, `web/embed.go`, `web/templates/layout.html`, `web/templates/error.html`, `web/templates/licenses.html`, `web/static/css/tokens.css` (verbatim copy), `web/static/css/app.css`, `web/static/css/fonts.css`, `scripts/vendor-plex.sh`, `web/static/fonts/…` (Plex Sans 400/500/600, Mono 400, Latin subset WOFF2), `web/static/licenses/*.txt` (goldmark, yaml, mermaid + its bundled packages, IBM Plex OFL, Go BSD), `docs/design.md` (screen jobs, deviations section empty)
- Test: `internal/web/server_test.go`

**Interfaces:**
- Consumes: Auth (Task 7), i18n (Task 8).
- Produces: `type Server struct` with `func New(deps Deps) (*Server, error)`; `type Deps struct{ Cfg config.Config; Auth *auth.Auth; I18n *i18n.Bundle; Users *users.Store; Repo *store.Repo; … }` (later tasks add fields); `func (s *Server) Handler() http.Handler`; `func (s *Server) render(w http.ResponseWriter, r *http.Request, name string, data any)` (injects user, lang, csrf, nav spaces); static files under `/static/` with `Cache-Control: public, max-age=31536000, immutable` and versioned URLs (`?v=<build hash>`).
- Layout (mobile first): top bar (product name „Hüpattl! Kanso“, search field, user menu), collapsible left navigation (spaces → page tree) as `<nav>` with `<a>` links and `aria-current`, main content. Light/dark via tokens. Footer link „Lizenzen und Dank“ / “Licenses and thanks” → `/about/licenses`. User menu offers the language switch `POST /me/lang` (`de`/`en`, calls `users.Store.SetLang`) and logout.

- [ ] **Step 1: Tests** – every response carries the CSP/nosniff/Referrer headers from Global Constraints; unauthenticated `/` redirects to login; `/about/licenses` (logged in) lists every file in `web/static/licenses/` and every module in `go.mod` `require` (`TestLicensesComplete` parses `go.mod`); unknown route renders `error.html` with 404. `POST /me/lang` with `en` makes the next page render English strings.
- [ ] **Step 2:** FAIL.
- [ ] **Step 3:** Implement; `vendor-plex.sh` downloads a pinned `@ibm/plex-sans`/`@ibm/plex-mono` npm tarball version (≥ 2 weeks old), verifies its sha256 (constant in the script), copies the WOFF2 subsets + `LICENSE.txt`.
- [ ] **Step 4:** PASS; run app with dev login, check light and dark at 360 px and 1280 px width; `python3 ~/.claude/skills/huepattl-rams-design/scripts/contrast.py` on fg/bg pairs used in `app.css`.
- [ ] **Step 5:** Commit `feat(web): application shell, design tokens, licenses page`.

### Task 10: Spaces and membership

**Files:** Create `internal/spaces/spaces.go`, `internal/web/handlers_spaces.go`, `internal/web/handlers_admin.go`, `web/templates/space_*.html`, `web/templates/admin_groups.html`; Test `internal/spaces/spaces_test.go`, `internal/web/handlers_spaces_test.go`

**Interfaces:**
- Consumes: `access.*` (Task 4), `store.*` (Tasks 2–3).
- Produces: `type Space struct{ Slug, Name, Description, Icon string; Members []access.Member }` (file `spaces/<slug>/space.yaml`; Icon = one of a fixed set of built-in SVG icon names, no emoji); `type Service struct`; `func NewService(dataDir string, repo *store.Repo) *Service`; `List() []Space`; `Get(slug string) (Space, bool)`; `Create(a store.Author, name, desc string) (Space, error)`; `Update(a store.Author, s Space) error`; `func (svc *Service) RoleOf(p access.Principal, slug string, g access.Groups) access.Role` (reads the current file/cache on every call – no caching in the session); `Visible(p, g) []Space`.
- Routes: `GET /` (visible spaces), `GET/POST /admin/spaces/new` (admin only), `GET/POST /s/{space}/settings` (owner: name, description, members as user/group + role), `GET/POST /admin/groups` (admin: create group, add/remove members from `users.List()`).
- Group and member changes are committed with the acting admin/owner as author.

- [ ] **Step 1: Tests** – viewer sees space in `Visible`, outsider does not and gets 404 on `/s/{space}`; editor gets 403 on settings; owner can add a group member; **removing a user's group membership makes the very next request with the same session return 404** (Review Focus 4); non-admin POST `/admin/groups` → 403; space slug from name „Gesundheit Timo“ → `gesundheit-timo`.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(spaces): spaces, members and Kanso-managed groups`.

### Task 11: Page tree and page CRUD

**Files:** Create `internal/wiki/tree.go`, `internal/wiki/pages.go`, `internal/web/handlers_pages.go`, `web/templates/page_view.html`, `web/templates/page_new.html`; Test `internal/wiki/pages_test.go`

**Interfaces:**
- Produces: `type PageMeta struct{ ID, Title string; Order int; Created, Updated time.Time; Author string }` (YAML frontmatter); `type Page struct{ Meta PageMeta; Space, Dir string; Body string }`; `type Node struct{ Meta PageMeta; Dir string; Children []*Node }`; `type Wiki struct`; `func NewWiki(dataDir string, repo *store.Repo) (*Wiki, error)` (scans all `spaces/*/pages/**/index.md`, builds tree + `id → (space, dir)` index); `Tree(space string) []*Node` (sorted by Order, then Title); `ByID(id string) (Page, bool)`; `Create(a store.Author, space, parentID, title string) (Page, error)`; `Save(a store.Author, id, title, body string) error` (rename of title does **not** rename the folder); `Move(a store.Author, id, newParentID string, order int) error` (`git mv`, rejects moving into own subtree); `Delete(a store.Author, id string) error` (only leaf pages; children must be moved/deleted first).
- Routes: `GET /p/{id}` (view, needs Read), `GET/POST /s/{space}/new?parent={id}` (Write), `POST /p/{id}/move`, `POST /p/{id}/delete` (Write).

- [ ] **Step 1: Tests** – create „Über uns“ twice under same parent → dirs `ueber-uns`, `ueber-uns-2`, titles verbatim; `ByID` after `NewWiki` reload finds both; `Move` updates index and history follows (`repo.Log` length unchanged); moving parent into its child → error; deleting page with children → error; viewer POST create → 403; page in invisible space → 404.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(wiki): page tree, ids and page lifecycle`.

### Task 12: Markdown rendering and preview

**Files:** Create `internal/markdown/render.go`; Modify `internal/web/handlers_pages.go`; Test `internal/markdown/render_test.go`

**Interfaces:**
- Produces: `type Renderer struct`; `func New() *Renderer` (goldmark v2 with GFM tables, strikethrough, task lists, autolinks, heading IDs; no unsafe HTML); `func (r *Renderer) Render(src []byte, ctx PageCtx) (template.HTML, Info, error)`; `type PageCtx struct{ PageID string }`; `type Info struct{ HasMermaid bool; Headings []Heading }`; AST transformer: relative link/image `attachments/x.png` → `/p/{id}/attachments/x.png`; destinations not matching the allowlist (Global Constraints) replaced by `#`; fenced block with info `mermaid` → `<pre class="mermaid">` with escaped text.
- Route: `POST /api/preview` (form `body`, `page`; Write on that page) → HTML fragment.

- [ ] **Step 1: Tests** – `[x](javascript:alert(1))` → no `javascript:` in output; `<script>alert(1)</script>` → escaped/omitted; `![a](attachments/b.png)` with PageID `abc` → `src="/p/abc/attachments/b.png"`; `![a](../../etc/passwd)` → `#`; mermaid fence → `HasMermaid` true and `<pre class="mermaid">` containing `A--&gt;B`; table and `- [x]` render; `/p/xyz` link kept.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(markdown): safe rendering with attachment and mermaid handling`.

### Task 13: Attachments

**Files:** Create `internal/wiki/attachments.go`; Modify `internal/web/handlers_pages.go`; Test `internal/wiki/attachments_test.go`

**Interfaces:**
- Produces: `func (w *Wiki) AddAttachment(a store.Author, id, filename string, r io.Reader, max int64) (name string, err error)` (sanitised filename: base name only, Slugify on stem, keep lower-case extension, unique; size limit 50 MiB → `ErrTooLarge`); `func (w *Wiki) OpenAttachment(id, name string) (*os.File, error)`; `ListAttachments(id string) []string`; `DeleteAttachment(a store.Author, id, name string) error`.
- Routes: `POST /p/{id}/attachments` (multipart, Write) → JSON `{ "name": "...", "markdown": "![..](attachments/...)" }`; `GET /p/{id}/attachments/{name}` (Read) with `Content-Type` from a fixed extension map (png, jpg, jpeg, gif, webp, pdf, txt, csv, zip; else `application/octet-stream`), `Content-Disposition: attachment` for everything except images and pdf (`inline`), `X-Content-Type-Options: nosniff`. SVG served as attachment download only.

- [ ] **Step 1: Tests** – filenames `../../x.png`, `C:\\evil.png`, `a\x00.png`, `.htaccess` stay inside `attachments/` with safe names; > limit → `ErrTooLarge` and no file left; `GET` of `x.svg` has `Content-Disposition: attachment`; viewer upload → 403.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(wiki): attachments per page`.

### Task 14: Edit locks

**Files:** Create `internal/wiki/locks.go`; Modify `internal/web/handlers_pages.go`; Test `internal/wiki/locks_test.go`

**Interfaces:**
- Produces: `type Lock struct{ PageID, Sub, Name string; Token string; Expires time.Time }`; `type Locks struct` (mutex, injectable `now func() time.Time`); `func NewLocks(ttl time.Duration, now func() time.Time) *Locks` (ttl 5 min); `Acquire(pageID, sub, name string) (Lock, error)` (`ErrLocked{Holder Lock}` if held by someone else and not expired; same user → refresh/new token); `Heartbeat(pageID, token string) error`; `Release(pageID, token string)`; `Break(pageID string)`; `Check(pageID, token string) error`.
- Routes: `GET /p/{id}/edit` (Write; acquires lock, otherwise shows read view with „{Name} bearbeitet gerade“ and, for owner/admin, a „Sperre aufheben“ button), `POST /p/{id}/lock/heartbeat`, `POST /p/{id}/lock/break` (BreakLock), `POST /p/{id}` save requires valid token → else **409** with JSON `{ "error": "lock_lost" }`.

- [ ] **Step 1: Tests** – second user cannot acquire; after `ttl` without heartbeat second user can; heartbeat extends; save with expired/broken token → 409 and file unchanged (Review Focus 3); viewer GET edit → 403.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(wiki): edit locks with heartbeat`.

### Task 15: Editor UI (source / preview)

**Files:** Create `web/templates/page_edit.html`, `web/static/js/editor.js`; Modify `web/static/css/app.css`, i18n files

**Interfaces:**
- Consumes: `/api/preview` (Task 12), lock routes (Task 14), attachment upload (Task 13).
- Behaviour: textarea (monospace, Plex Mono) + preview pane; ≥ 960 px side by side, below that a two-tab switch „Quelltext“/„Vorschau“ (`role="tablist"`); preview refresh debounced 400 ms; heartbeat every 60 s; on 409 show a blocking notice „Die Sperre ist abgelaufen. Dein Text bleibt hier; kopiere ihn, bevor du die Seite verlässt.“ and keep the text; `beforeunload` warning when dirty; drag-and-drop/paste of files uploads and inserts the returned Markdown at the cursor; one primary action „Speichern“ (accent); `Ctrl/Cmd+S` saves. ES module, no globals, CSRF token from `<meta name="csrf">`.

- [ ] **Step 1:** Add handler test: `GET /p/{id}/edit` renders textarea with current body and `<meta name="csrf">`.
- [ ] **Step 2–4:** FAIL → implement → PASS; manual check in browser via dev login: type, preview updates, paste image inserts `![…](attachments/…)`, second browser session sees lock notice, stop heartbeat (devtools offline > 5 min with a short test TTL flag `KANSO_LOCK_TTL` dev-only) → save shows 409 notice with text intact.
- [ ] **Step 5:** Commit `feat(web): markdown editor with source and preview`.

### Task 16: Mermaid

**Files:** Create `scripts/vendor-mermaid.sh`, `web/static/vendor/mermaid/` (generated), `web/static/js/mermaid-loader.js`; Modify `page_view.html`, `page_edit.html`, licenses list; Test `internal/web/vendor_test.go`

**Interfaces:**
- `vendor-mermaid.sh`: `npm pack mermaid@12.2.0` into a temp dir, verify sha512 integrity from `npm view mermaid@12.2.0 dist.integrity` against a constant in the script, copy `dist/mermaid.esm.min.mjs` + `dist/chunks/mermaid.esm.min/*` **except** `elk-*.mjs`, write `VERSION` and collect license texts of bundled packages into `web/static/licenses/`.
- `mermaid-loader.js`: if `document.querySelector('pre.mermaid')`, `import('/static/vendor/mermaid/mermaid.esm.min.mjs')`, `initialize({ startOnLoad: false, securityLevel: 'strict', layout: 'dagre', theme: dark ? 'dark' : 'neutral', fontFamily: 'IBM Plex Sans' })`, `run({ querySelector: 'pre.mermaid' })`; render errors shown inline as text under the block. Called after each preview refresh too.

- [ ] **Step 1: Tests** – `TestNoELKVendored` (no file matching `elk-*` and no `org.eclipse.elk` string in `web/static/vendor/mermaid/`); page with mermaid includes the loader script, page without does not.
- [ ] **Step 2–4:** FAIL → run script → implement → PASS; browser check: flowchart and sequence diagram render; label `<img src=x onerror=alert(1)>` renders as text (Review Focus 5); a `flowchart-elk` diagram shows an inline error, not a broken page.
- [ ] **Step 5:** Commit `feat: vendored mermaid without ELK, lazy loaded`.

### Task 17: History, diff, restore

**Files:** Create `internal/wiki/history.go`, `web/templates/page_history.html`, `web/templates/page_revision.html`; Test `internal/wiki/history_test.go`

**Interfaces:**
- Consumes: `Repo.Log/Show/Diff` (Task 3).
- Produces: `func (w *Wiki) History(id string, limit int) ([]store.Revision, error)`; `func (w *Wiki) Revision(id, hash string) (Page, error)`; `func (w *Wiki) Diff(id, from, to string) ([]DiffLine, error)` with `type DiffLine struct{ Kind byte; Text string }` (`' '`, `'+'`, `'-'`, parsed from unified diff, headers dropped); `func (w *Wiki) Restore(a store.Author, id, hash string) error` (new commit „Wiederhergestellt: <hash[:7]>“, requires lock like Save).
- Routes: `GET /p/{id}/history`, `GET /p/{id}/history/{hash}` (rendered + diff to current), `POST /p/{id}/history/{hash}/restore` (Write). `hash` must match `^[0-9a-f]{7,40}$`.

- [ ] **Step 1: Tests** – three saves → history 3 entries with authors; diff between 1 and 3 has expected +/- lines; restore makes current body equal revision 1 and adds a 4th entry; hash `HEAD;rm` → 400.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(wiki): page history, diff and restore`.

### Task 18: Search

**Files:** Create `internal/search/index.go`, `internal/web/handlers_search.go`, `web/templates/search.html`; Test `internal/search/index_test.go`

**Interfaces:**
- Produces: `type Doc struct{ ID, Space, Title, Body string }`; `type Index struct`; `func NewIndex() *Index`; `Put(d Doc)`; `Remove(id string)`; `type Hit struct{ ID, Space, Title, Snippet string; Score int }`; `func (ix *Index) Search(q string, allowed func(space string) bool, limit int) []Hit` (tokens: lower-case, umlaut-folded like Slugify, prefix match on last token; title hits weigh ×3; snippet = 160 chars around first hit, plain text).
- `Wiki` updates the index on Create/Save/Delete/Restore (`Wiki` gets an `OnChange func(Page, deleted bool)` hook set in `main.go`).
- Route: `GET /search?q=` → results only from spaces where the user has Read.

- [ ] **Step 1: Tests** – „größe“ finds „Groesse“ and „Größe“; prefix „kind“ finds „Kindergarten“; page in a space the user cannot read never appears; title hit ranks above body hit.
- [ ] **Step 2–4:** FAIL → implement → PASS.
- [ ] **Step 5:** Commit `feat(search): permission-aware in-memory search`.

### Task 19: Wiring, lint, review, docs

**Files:** Modify `cmd/kanso/main.go`, `README.md`, `CHANGELOG.md`, `docs/design.md`; Create `docs/deploy-authelia.md` (OIDC client snippet for homelab2: `client_id = "kanso"`, redirect `https://kanso.<domain>/auth/callback`, scopes `openid profile email groups`, `require_pkce = true`, `pkce_challenge_method = "S256"`, `token_endpoint_auth_method = "client_secret_basic"`)

- [ ] **Step 1:** Wire all services in `main.go` (functions ≤ 30 lines: `buildDeps`, `buildServer`, `run`); graceful shutdown on SIGTERM.
- [ ] **Step 2:** `go test ./... && go vet ./... && gofmt -l .` → no output besides PASS; `golangci-lint run ./...` if installed, else `go -C ~/.claude/skills/huepattl-code-quality/scripts/gofuncmetrics run . $PWD | sort -rn | head` → report every function > 30 lines or cyclomatic > 10 to the user.
- [ ] **Step 3:** `govulncheck ./...` (ask before installing) or OSV API query for each module.
- [ ] **Step 4:** End-to-end with dev login: create space as admin, add group „Familie“ as editor, second dev user edits a page with a Mermaid diagram and an image, history shows both authors, search finds it, viewer cannot edit, switch language to English. Rams review per `~/.claude/skills/huepattl-rams-design/references/review.md` on start, page view, editor, settings at 360 px and 1280 px.
- [ ] **Step 5:** `huepattl-legal-check` on the repository (licenses page complete, no outbound calls except OIDC, disclaimer in README and About). Update CHANGELOG `[Unreleased]`. Commit `chore: wire phase 1, docs and checks`.
