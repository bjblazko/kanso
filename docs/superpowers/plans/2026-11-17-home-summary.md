# Start page summary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/` opens with a summary of what changed since the viewer's last visit (at least 24 h, or 24 h / 7 d / 30 d), only from documents they may read, with charts.

**Architecture:** A small `internal/visits` store keeps `previous`/`last` visit per person in the state directory. `internal/reports` gets a pure `Summarize` over changes; `internal/web` feeds it from the reports' collection (`newReport`, scope `seen`, cached scan per HEAD, ADR-015) and renders it above the spaces. Charts reuse `reports.js` with d3.

**Tech Stack:** Go stdlib, html/template, i18n YAML, d3 (vendored), Playwright.

**Spec:** `docs/superpowers/specs/2026-11-17-home-summary-design.md`

## Global Constraints

- No new dependency. d3 may load on report pages and the start page only.
- Every source file starts with `SPDX-License-Identifier: Apache-2.0`.
- Functions ~30 lines, cyclomatic ≤ 10; `just lint` and `just test` green.
- Nothing about a document the viewer may not read reaches the page: not its title, space, person, nor its count. Admins get the same (ADR-014); there is no scope "all" here.
- A visit is never a Git commit; visit times live only in `KANSO_STATE_DIR`.
- de and en texts for every new key; dark mode; phone width without horizontal scroll; charts in the reports' palette (`--chart-1…6`).
- Kinds on the start page: page, recipe, note, list, task (no attachments), in `reports.Kinds` order.

## Review Focus

1. A commit by an author whose e-mail matches no user → person `""`: counted under "ohne Namen", never hidden by the own filter. (Task 2 test `TestSummarizeUnknownPersonStaysWithOwnHidden`.)
2. `?range=` with an unknown value or `?own=` with anything but `1` → default, 200, not 400. (Task 3 test `TestHomeUnknownRangeFallsBack`.)
3. A broken `visits.json` (garbage, empty) → treated as no visits, rewritten on next touch; the page still renders. (Task 1 test `TestTouchSurvivesBrokenFile`.)
4. History scan failing (`HistoryError`) → summary renders from what is there plus the reports' warning text, no 500. (Task 3 test `TestHomeGitErrorStillRenders`.)
5. Bucketing across midnight and the 48 h boundary uses local time; a 48 h window is hours, 49 h is days. (Task 2 test `TestBucketsHoursUpToTwoDays`.)

---

### Task 1: Visit store

**Files:**
- Create: `internal/visits/visits.go`, `internal/visits/visits_test.go`
- Modify: `internal/web/server.go` (Deps gains `Visits *visits.Store`), `cmd/kanso/main.go` (`visits.NewStore(cfg.StateDir)`), `internal/web/server_test.go` (`buildTestDeps` passes `visits.NewStore(cfg.StateDir)`)

**Interfaces:**
- Produces:
  - `func NewStore(stateDir string) *Store` — file `<stateDir>/visits.json`, map sub → `{previous, last}` (RFC 3339).
  - `func (s *Store) Touch(sub string, now time.Time) (previous time.Time, err error)` — if `last` is zero or `now − last > time.Hour`: `previous ← last`, `last ← now`, write atomically (temp file + rename, mode 0o600, like `ai.Store.Save`). Returns `previous` after the move (zero on first visit). Mutex-guarded.
  - `func Since(previous, now time.Time) time.Time` — `now − 24h` when `previous` is zero or later than that, else `previous`.
  - `const Grace = time.Hour`, `const Floor = 24 * time.Hour`.

- [ ] **Step 1: Write the failing tests** in `visits_test.go`:
  - `TestTouchFirstVisit`: `Touch("anna", t0)` → zero previous.
  - `TestTouchWithinHourKeepsWindow`: `Touch(t0)`, `Touch(t0+30m)` → zero; `Touch(t0+2h)` → `t0`; `Touch(t0+2h30m)` → `t0` again.
  - `TestTouchPersists`: a second `NewStore` on the same dir returns `t0` on `Touch(t0+2h)`; file mode is `0o600`.
  - `TestTouchSurvivesBrokenFile`: `visits.json` = `"{nope"` → `Touch` returns zero previous and nil error; file is valid JSON afterwards.
  - `TestSince`: zero → `now−24h`; previous 3 h ago → `now−24h`; previous 3 days ago → that time.
- [ ] **Step 2:** `go test ./internal/visits/` → FAIL (package missing).
- [ ] **Step 3:** Implement `visits.go`; wire Deps, main, `buildTestDeps`.
- [ ] **Step 4:** `go test ./internal/visits/ ./internal/web/` → PASS.
- [ ] **Step 5: Commit** `feat(home): remember the previous visit per person in the state directory`

### Task 2: Summary arithmetic in `internal/reports`

**Files:**
- Create: `internal/reports/recent.go`, `internal/reports/recent_test.go`

**Interfaces:**
- Produces:
  - `type Change struct { Kind Kind; DocID string; When time.Time; Person string; Ticked bool }` — `Ticked` only for list items ticked.
  - `type KindSum struct { Kind Kind; New, Changed, Ticked int; Spark []int }` — `New`: documents created in the window; `Changed`: other documents with a change; `Ticked`: list ticks; `Spark`: changes per bucket.
  - `type PersonSum struct { Person string; N int }` — most first, ties by person; `""` last.
  - `type Summary struct { From, To time.Time; Hourly bool; Buckets []time.Time; Kinds []KindSum; Band map[Kind][]int; People []PersonSum; Docs []Recent; Total int }`
  - `type Recent struct { Doc Doc; When time.Time; Person string }` — one per document, its latest change, newest first.
  - `func Summarize(docs []Doc, changes []Change, from, to time.Time, viewer string, own bool) Summary`
    - Ignores changes outside `[from, to]`, changes whose `DocID` is not in `docs`, docs of kind `Attachment`.
    - `own == false` drops changes with `Person == viewer`; `Person == ""` is never dropped.
    - `Hourly = to − from ≤ 48h`; buckets start at `from` truncated to the local hour (or local midnight), step 1 h / 1 day, through `to`.
    - Kinds with no change are left out; `Total` is the number of kept changes.

- [ ] **Step 1: Write the failing tests** in `recent_test.go` (build `docs` and `changes` by hand, `loc := time.FixedZone("CET", 3600)`):
  - `TestSummarizeCountsNewChangedTicked`: page created in window + edited → `New 1`; recipe created before, edited in window → `Changed 1`; list with 2 ticked changes → `Ticked 2`.
  - `TestSummarizeHidesOwnUnlessAsked`: anna's change hidden for viewer anna with `own=false`, present with `own=true`.
  - `TestSummarizeUnknownPersonStaysWithOwnHidden`: change with `Person ""` kept with `own=false`; `People` ends with `{"" 1}`.
  - `TestSummarizeSkipsOutsideWindowAndUnknownDocs`: change 1 s before `from` and change for a DocID not in docs → `Total 0`, `Kinds` empty.
  - `TestSummarizeRecentNewestFirst`: two docs → `Docs[0]` is the later one, with its latest person.
  - `TestBucketsHoursUpToTwoDays`: `to−from = 48h` → `Hourly`, 49 buckets; `49h` → daily, buckets at local midnight; a change at 23:30 local lands in that day's bucket.
- [ ] **Step 2:** `go test ./internal/reports/ -run 'Summarize|Buckets'` → FAIL (undefined).
- [ ] **Step 3:** Implement `recent.go`.
- [ ] **Step 4:** Same command → PASS; `go test ./internal/reports/` PASS.
- [ ] **Step 5: Commit** `feat(home): summarize recent changes by kind, time and person`

### Task 3: Start page summary (server side)

**Files:**
- Create: `internal/web/handlers_home.go`, `internal/web/handlers_home_test.go`
- Modify: `internal/web/handlers_spaces.go` (`home` moves out; `homePage` gains `Summary *homeSummary`), `web/templates/home.html`, `internal/i18n/de.yaml`, `internal/i18n/en.yaml`

**Interfaces:**
- Consumes: `visits.Store.Touch`, `visits.Since` (Task 1); `reports.Summarize`, `reports.Change`, `reports.Summary` (Task 2); `s.newReport(v, reportCtx{…, scope: scopeSeen})`, `s.addedBy`-style e-mail → sub map, `rb.docs`, `rb.rc.scan.Changes`, `rb.page.HistoryError`.
- Produces:
  - `func (s *Server) recentChanges(rb *reportBuild) []reports.Change` — for every readable doc (`Title != ""`) that is not a list: Git changes on `doc.Path` (not `Deleted`), person from the commit e-mail (unknown → `""`); for lists: item writes (`At.Text`) and ticks (`At.Done`, `Ticked: true`) with person `""`.
  - `type homeSummary struct { Range string; Ranges []sharedTab; Own bool; OwnHref string; From time.Time; Sentence string; Tiles []homeTile; Band, People reportCard; Groups []homeGroup; More int; HistoryError bool }`
  - `type homeTile struct { Kind, Label, Value string; Spark string }` (`Spark` is the card id for its data).
  - `type homeGroup struct { Name string; Items []homeItem }`, `type homeItem struct { Title, Href, Person, When string }` — grouped by space name, then "Listen", "Notizen", "Aufgaben"; at most 20 items over all groups, `More` = the rest.
  - Range keys: `""` (since last visit), `24h`, `7d`, `30d`; anything else → `""`. `own=1` → own on; anything else off. Links keep the other parameter.
  - The handler calls `Touch` only for `range == ""`, so switching the select does not move the visit.
  - Chart data goes into `JSON` under ids `band`, `people`, `spark-<kind>`; the template writes `<script type="application/json" id="report-data">`.
  - Sentence: "Seit {when}: " + parts joined by ", ", from keys `home.part_<kind>_new|changed` and `home.part_list_ticked` with `_one` variants; nothing → `home.nothing` "Seit {when} ist nichts passiert." `{when}` is "heute, 15:04", "gestern, 15:04" or "Mo, 02.01., 15:04" (en: "today, 3:04 PM" etc. via the bundle's format keys).

- [ ] **Step 1: Write the failing tests** in `handlers_home_test.go` (use `setupHaus`, `ts.createPage`, `ts.note`, `otherAdmin`, `newTestServer`; the dev login gives the e-mail mapping the reports use):
  - `TestHomeShowsOthersChanges`: anna creates page "Keller"; admin's `/` contains `Keller`, `1 neue Seite`, `id="report-data"`, and the select with `seit deinem letzten Besuch`.
  - `TestHomeHidesOwnUnlessAsked`: anna's `/` lacks `Keller` in the summary; `/?own=1` has it.
  - `TestHomeNeverShowsUnreadable`: anna writes page "Tagebuch" in a space the second admin (`otherAdmin`) is not a member of, plus a private note "Geheim"; the other admin's `/` and `/?range=30d&own=1` contain neither title and the sentence counts 0 pages (`ist nichts passiert`).
  - `TestHomeUnknownRangeFallsBack`: `/?range=1y&own=yes` → 200, current range is "seit deinem letzten Besuch", own off.
  - `TestHomeRangeDoesNotMoveVisit`: write `visits.json` with anna's `previous`/`last` 3 days ago; `/?range=24h` leaves the file unchanged; `/` then moves it.
  - `TestHomeGitErrorStillRenders`: as `TestReportsGitErrorStillRenders`; `/` → 200 with the reports' history warning.
- [ ] **Step 2:** `go test ./internal/web/ -run TestHome` → FAIL.
- [ ] **Step 3:** Implement `handlers_home.go`, the template part above the spaces list (sentence; range select as a GET form with `<select name="range">` + submit button labelled "Anzeigen", and the "Eigene einblenden"/"Eigene ausblenden" link; tiles with `<div class="report-chart" data-chart="spark" data-id="spark-<kind>" aria-hidden="true">`; band and people cards reusing the `report-table` define from `reports.html` — move that define to `layout.html` or a shared template if needed; groups list), i18n keys `home.*` in de and en.
- [ ] **Step 4:** `go test ./internal/web/` → PASS; `just lint` clean.
- [ ] **Step 5: Commit** `feat(home): summary of recent changes on the start page`

### Task 4: "Heute" section

**Files:**
- Modify: `internal/web/handlers_home.go`, `internal/web/handlers_home_test.go`, `web/templates/home.html`, i18n

**Interfaces:**
- Consumes: `s.MealPlans.For(s.groups(), sub)`, `s.MealPlans.Week(planID, mealplan.WeekKey(now))`, `Week.Days[now.Format("2006-01-02")]`; `s.Tasks.AllBoards()`, `tasks.Actor{Sub, Groups}.Sees`, `s.Tasks.AllTasks()`, `tasks.DueOf(due, now)`.
- Produces: `homeSummary.Today []homeItem` (meals: title → `/mealplan/<id>`, Person = plan name) and `homeSummary.Due []homeItem` (tasks not `Done`, `DueToday` or `DueOverdue`, overdue first; When = "überfällig"/"heute").

- [ ] **Step 1: Write the failing tests:** `TestHomeTodayShowsMealAndDueTask` (group plan with today's entry, a task due today on a board anna sees → both on anna's `/`); `TestHomeTodayHidesOtherBoards` (task due today on a private board of admin → not on anna's `/`); section absent when both empty.
- [ ] **Step 2:** `go test ./internal/web/ -run TestHomeToday` → FAIL.
- [ ] **Step 3:** Implement; template section "Heute" below the summary, above the spaces.
- [ ] **Step 4:** `go test ./internal/web/` → PASS.
- [ ] **Step 5: Commit** `feat(home): today's meals and due tasks on the start page`

### Task 5: Charts, styles, browser test

**Files:**
- Modify: `web/static/js/reports.js`, `web/static/css/app.css`, `web/templates/home.html` (`{{define "scripts"}}` loads `vendor/d3/d3.min.js` and `js/reports.js`, as `reports.html` does)
- Create: `e2e/tests/home.spec.js`

**Interfaces:**
- Consumes: JSON ids from Task 3. `band`: `{keys: [labels], unit: 'hour'|'day', rows: [{label: ISO local time, values: [n per key]}]}` — the shape of `stacked-area` plus `unit`. `people`: the `ranking` shape. `spark-<kind>`: `{values: [n…]}`.
- Produces: `stacked-area` honours `unit` (parse `%Y-%m-%dT%H:%M` / `%Y-%m-%d`, ticks `%H:%M` / `%d.%m.`, tooltip the same; default stays months); new `draw.spark` (2 px line, `--chart-1`, height 28, no axes, no hover).

- [ ] **Step 1: Write the failing browser test** `home.spec.js`: Rita creates a space and page; Otto signs in, opens `/`, sees the sentence containing the page title count, `.report-chart[data-chart="spark"] svg` and `[data-id="band"] svg` visible, selects "7 Tage" + "Anzeigen" → URL has `range=7d`; "Eigene einblenden" → URL has `own=1`. Phone width (390 px): `document.documentElement.scrollWidth <= 390`.
- [ ] **Step 2:** `just e2e` (or `npx playwright test home`) → FAIL (no svg).
- [ ] **Step 3:** Implement JS and CSS (`.home-summary`, `.home-tiles` grid `repeat(auto-fit, minmax(9rem, 1fr))`, tile = `.report-tile` look); look at it in light and dark at 390 px and desktop per `huepattl-rams-design`.
- [ ] **Step 4:** `just e2e` → all pass; `just test` → PASS.
- [ ] **Step 5: Commit** `feat(home): charts on the start page`

### Task 6: Records

**Files:**
- Create: `docs/adr/018-start-page-summary-from-visits-and-reports.md`, `docs/features/2026-11-17 start page summary.md`
- Modify: `CHANGELOG.md` (`[Unreleased]` → Added), `docs/architecture/arc42.md` (building block `visits`, runtime "Opening the start page", ADR 018 row, `**Updated:**`), `CLAUDE.md` (d3 line: "report pages and the start page"), `docs/BACKLOG.md` (idea: record who ticked a list item / when something was shared, for "Neu für dich")

- [ ] **Step 1:** Write the records; ADR covers: window from visits in state not Git, the 1 h grace, built on the reports' collection, no "Neu für dich" and why.
- [ ] **Step 2:** `just test && just lint` → PASS.
- [ ] **Step 3: Commit** `docs(home): ADR-018, feature doc, arc42, changelog`
