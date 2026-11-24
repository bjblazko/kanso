# Menü anpassbar + Fokus – Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** A rail customisable per device (hide, reorder, "Mehr" on phones) and a Pomodoro page "Fokus" as the first new function.
**Architecture:** `railItems` in Go renders the rail and the "Mehr" list; `web/static/js/menu.js` (classic, synchronous) applies `localStorage` preferences; `/me/menu` with `menu-settings.js`; `/focus` with `focus.js` (module).
**Tech Stack:** Go templates, plain JS, Playwright (clock, permissions).
**Spec:** `docs/superpowers/specs/2026-11-24-menu-and-focus-design.md`

## Global Constraints

SPDX header everywhere; no dependency; no cookie; no server state for preferences/timer; CSP stays (no inline script); Rams: Start/Pause is the only accent on Fokus.

## Review Focus

- Corrupt or old JSON in localStorage → default menu, no JS error (Task 2 Playwright).
- All items except Start hidden → rail shows only Start, no empty "Mehr" (Task 2).
- Timer runs, tab reloaded → continues with the right remaining time (Task 4, clock).
- Notifications denied → sound only, no error (Task 4).
- Current page hidden in the menu → page still reachable (Task 2: hide Wiki, open /wiki).

### Task 1: railItems in Go, rail from the list, "Mehr" markup, Fokus module
Files: `internal/web/rail.go` (+test), `render.go` (`view.Rail`, module `focus`), `layout.html`, `icons.html` (`timer`, `more`), i18n.
- [ ] Test `TestRailItems`: page HTML contains in order `data-key="start"…"focus"`, `class="rail-more"`, link `/me/menu`; `/focus` marks Fokus with `aria-current`.
- [ ] Implement `type railItem struct{Key, Href, Icon, Label string; Current bool}`, `func (s *Server) rail(v view) []railItem`. Commit.

### Task 2: menu.js + CSS (scroll, Mehr) + /me/menu
Files: `web/static/js/menu.js`, `web/static/js/menu-settings.js`, `web/templates/me_menu.html`, `internal/web/handlers_me_menu.go` (+test), `app.css`, `e2e/tests/menu.spec.js`.
- [ ] Go test `TestMenuPage` (200, one row per item, script menu-settings.js).
- [ ] Playwright: hide Wiki + move Fokus up → after reload the rail order is right and Wiki is missing; `/wiki` still loads; corrupt JSON → default; phone (mobile project): with 7 items 4 + Mehr, Mehr opens the rest + "Menü anpassen"; all hidden except Start → no Mehr.
- [ ] Implement. Commit.

### Task 3: /focus page
Files: `internal/web/handlers_focus.go` (+test), `web/templates/focus.html`, `web/static/js/focus.js`, CSS.
- [ ] Go test `TestFocusPage`: 200, time `25:00`, buttons Start/Zurücksetzen/Überspringen, duration fields.

### Task 4: Timer behaviour (Playwright with page.clock)
- [ ] Tests in `e2e/tests/focus.spec.js`: start → after 25 min the phase is "Pause", the `Notification` stub got "Fokus vorbei" and the AudioContext stub was played; reload in between continues; 4 focus rounds → long break 15:00; denied permission → no error; title shows remaining time.
- [ ] Implement focus.js. Commit.

### Task 5: Docs, legal check
- ADR-020 (menu per device in localStorage), ADR-021 (notifications only with an open tab, no Web Push; offline-push feature without push), arc42 ch. 9 + building blocks, feature docs `2026-11-24 menu.md`, `2026-11-24 focus.md`, `docs/design.md` rows, CHANGELOG, update the navigation feature doc. `just lint`, `just test`, full e2e. No deploy.
