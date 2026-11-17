# Start Page of Its Own Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** `/` is Kanso's start page (summary + "Heute"); the wiki's spaces list moves to `/wiki`.

**Architecture:** Split the 0.10.0 `home` handler and template: `start` (summary, today) on `/`, `wiki` (spaces) on `/wiki`. New rail entry and module `start`.

**Tech Stack:** Go, html/template, i18n YAML, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-17-start-page-design.md`

## Global Constraints

- Screen-head rule: sticky toolbar + title above content; no buttons below.
- de and en texts; no new dependency; d3 loads on report pages and the start page only.
- Summary behaviour, rights and visit logic unchanged from 0.10.0 (ADR-018).

## Review Focus

1. `/wiki` without the summary must not record a visit nor load d3. (Task 1 `TestWikiListsSpacesOnly`.)
2. Old bookmarks of `/?deleted=` — deleting a space now lands on `/wiki?deleted=`; `/` ignores `deleted`. (Task 1 `TestDeleteSpaceLandsOnWiki`.)
3. Rail: exactly one `aria-current` on `/` (Start) and on `/wiki` (Wiki). (Task 1 `TestRailMarksStartAndWiki`.)

---

### Task 1: Start page and wiki page

**Files:**
- Modify: `internal/web/routes.go` (`GET /wiki` → `s.wikiHome`), `internal/web/handlers_home.go` (`home` renders `start` with `startPage{Summary}`; new `wikiHome` renders `wiki` with `homePage` minus `Summary`), `internal/web/render.go` (`moduleOf("/")` = `start`, `/wiki` in the wiki module's prefixes as exact path), `internal/web/server.go` (pageNames: `home` → `start`, add `wiki`), `internal/web/handlers_space_delete.go` (redirect `/wiki?deleted=`), templates: new `start.html` (from the summary + today part), `home.html` → `wiki.html` (spaces part), `layout.html` (rail "Start" first with icon `home`, Wiki → `/wiki`), i18n `nav.start`.
- Tests: `internal/web/handlers_home_test.go`, `handlers_spaces_test.go`, `screen_head_test.go`, `audience_test.go`, `cookbook_test.go`, `handlers_me_test.go` — move the spaces assertions from `/` to `/wiki`.

**Interfaces:**
- Produces: `func (s *Server) wikiHome(w http.ResponseWriter, r *http.Request)`; `type startPage struct{ Summary *homeSummary }`; module name `"start"`.

- [ ] **Step 1: Write failing tests:** `TestStartPageIsOverall` (`/` has h1 "Was passiert ist", rail link `href="/"` with `aria-current` and text "Start", no `href="/s/haus"` in `<main>`); `TestWikiListsSpacesOnly` (`/wiki` has h1 "Wiki-Bereiche", `href="/s/haus"`, no `home-summary`, no `d3.min.js`; `visits.json` absent afterwards); `TestDeleteSpaceLandsOnWiki` (Location `/wiki?deleted=Haus`); `TestRailMarksStartAndWiki`. Move the existing spaces assertions to `/wiki`.
- [ ] **Step 2:** `go test ./internal/web/ -run 'Start|Wiki|Rail|DeleteSpace'` → FAIL.
- [ ] **Step 3:** Implement as listed; select and own toggle go into the start page's `page-toolbar`.
- [ ] **Step 4:** `go test ./...` → PASS; `just lint` clean.
- [ ] **Step 5: Commit** `feat(start): a start page of its own; the wiki moves to /wiki`

### Task 2: Browser tests and records

**Files:**
- Modify: `e2e/tests/*.spec.js` that expect spaces on `/` (`spaces`, `recipes`, `navigation`, `shell`, `page-sharing`, `home`), ADR-018 (where the page lives, `**Updated:**`), feature doc, arc42 (runtime "Opening the start page", building blocks unchanged), CHANGELOG `[Unreleased]` → Changed.

- [ ] **Step 1:** Update the specs; `home.spec.js` also clicks rail "Start" and "Wiki".
- [ ] **Step 2:** `cd e2e && npx playwright test` → all pass.
- [ ] **Step 3:** Look at `/` and `/wiki` light, dark and phone.
- [ ] **Step 4: Commit** `docs(start): ADR-018, feature doc, arc42, changelog; e2e for /wiki`
