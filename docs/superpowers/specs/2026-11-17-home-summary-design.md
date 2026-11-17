# Start page: what happened since your last visit — design

**Status:** approved 2026-11-17

## Why

Whoever opens Kanso wants to see at a glance what the others did since they
were last here: pages changed, recipes added, notes, tasks, list items. Today
`/` shows only the spaces and what is shared with the person. A few charts
make it lighter to read than a plain list.

## Goals, non-goals, anti-goals

- Goal: one summary on `/`, above the spaces, of the changes the viewer may
  read, in a time window that follows their visits.
- Goal: charts in the look of the reports (0.8.0).
- Non-goal: notifications, mails, unread markers on single documents, a
  feed one scrolls through endlessly.
- Non-goal: a new data source; the summary reads what the reports read.
- Anti-goal: anything about a document the viewer may not read reaching the
  page — not its title, not its space, not that it changed. Admins see only
  what they may read (ADR-014); the reports' scope "all" does not exist here.
- Anti-goal: a visit becoming a Git commit or a change in the summary.

## Time window

- **Default: since the last visit, at least 24 hours.** The SSO session is no
  measure: it lasts 30 days and slides forward once a day.
- Kanso keeps per person `previous` and `last` visit to `/` in the state
  directory (`KANSO_STATE_DIR`, next to sessions and `ai.json`), never in Git.
  A visit to `/` more than one hour after `last` moves `previous ← last`,
  `last ← now`; within the hour nothing moves. The window starts at
  `min(previous, now − 24 h)`. First visit ever: 24 hours.
- A select above the summary offers: *seit deinem letzten Besuch* (default),
  24 Stunden, 7 Tage, 30 Tage. The choice is a query parameter
  (`?range=24h|7d|30d`); without it the default applies. The page says the
  window in words ("seit Montag, 18:40").

## What the page shows (top to bottom)

1. **One sentence:** "Seit gestern, 18:40: 4 Seiten geändert, 2 neue Rezepte,
   7 Listeneinträge abgehakt." Nothing happened: "Seit … ist nichts passiert."
   and the charts are left out.
2. **Number tiles per kind** (page, recipe, note, list, task — the reports'
   kinds without attachments), each with the count of changed documents and a
   small line of the changes over the window. Kinds with 0 are left out.
3. **Activity band:** changes over the window (hours for ≤ 2 days, days
   beyond), stacked by kind, with a hover rule; below it who was active, by
   number of changes. List ticks carry no person (todos do not record who
   ticked) and count under "ohne Namen" — never under the owner.
4. **Zuletzt geändert:** the changed documents, grouped by space (lists, notes
   and tasks under their own headings), newest first, each with title link,
   who last changed it and when; at most 20, then "und N weitere".
5. **Heute:** today's entries of the meal plans the viewer may enter, and the
   viewer's tasks due today or overdue. Not "happened", but what one looks for
   next; left out when empty.
6. The spaces list and "Mit mir geteilt" as today, below.

- *Builder's call:* no "Neu für dich" section. Kanso records no time for a
  share (page members change with the page file, list shares are not in
  Git), so "shared since your last visit" cannot be told apart from "changed".
  "Mit mir geteilt" below stays as today.
- **Own changes** are hidden by default; a toggle "Eigene einblenden"
  (`?own=1`) adds them.
- **Deleted documents** are left out: once deleted their rights cannot be
  checked.
- Each chart has its table view, as in the reports.

## Approaches

- **A (chosen): build on the reports' collection.** The same documents and
  events (Git changes with author, notes, list items), the same read check,
  the same cache per Git head (ADR-015), filtered by the window and by "may
  read". Charts reuse the d3 drawers in `reports.js` (stacked area, bars) and
  add small ones (spark line, band). d3 is extended from report pages to the
  start page; the browser caches the file.
- B: a separate activity log written on every change — more exact (who
  ticked), but a second store beside Git that can drift, and empty for the
  past.
- C: charts as SVG from Go without d3 — lighter, but a second way of drawing
  that looks different from the reports.

## Constraints

- No new dependency. d3 rule in `CLAUDE.md` changes to "report pages and the
  start page".
- Functions ~30 lines, cyclomatic ≤ 10; packages by domain: window and visit
  logic in a small package (`internal/visits`), the summary's counting next to
  `internal/reports`, handlers in `internal/web`.
- de and en texts; dark mode; works at phone width; charts follow the
  reports' palette.
- Unit tests first; Go HTTP test that a page in a space the viewer may not read
  never appears; Playwright test of the window select and the own toggle.

## Left to the builder

The exact sentence wording, tile and band sizes per `huepattl-rams-design`,
how the summary reuses the reports cache (a shared collect step or a
narrower one), and the hour/day bucketing details.

Records: ADR-018 (start page summary: window from visits kept in state, built
on the reports' collection), feature doc, CHANGELOG, arc42, CLAUDE.md d3 line.
