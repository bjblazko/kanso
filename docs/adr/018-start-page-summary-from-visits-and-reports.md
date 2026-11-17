# 018 — Start Page Summary: Window from Visits, Built on the Reports

**Status:** Accepted
**Created:** 2026-11-17
**Updated:** 2026-11-17

## Context

The start page should say what the others did since one was last here: pages
and recipes changed, notes, tasks, list items. It must show nothing about a
document the viewer may not read, admins included (ADR-014).

"Since the last visit" needs a last visit. The SSO session is no measure: it
lasts 30 days and slides forward once a day, so "since the session began"
would jump between weeks and a day. People are stored in Git; writing a visit
there would make every visit a commit, and that commit would itself show up
as a change.

The reports (ADR-015) already collect every document with its person, path and
read check, and keep the history scan per Git head.

## Decision

- **Place:** `/` is Kanso's start page, its own entry "Start" first in the
  rail, not part of the wiki; the wiki's spaces list lives at `/wiki`
  (changed in 0.10.1 — 0.10.0 had put the summary above the spaces).
- **Window:** since the previous visit to `/`, at least 24 hours. A select
  offers 24 hours, 7 and 30 days instead (`?range=`), and `?own=1` adds one's
  own changes, which are hidden by default.
- **Visits** live in `<state dir>/visits.json` (`internal/visits`): per person
  `previous` and `last`. A visit more than an hour after `last` moves them on;
  within the hour nothing moves, so reloading or coming back to `/` keeps the
  window. Only the default window counts as a visit.
- **Data:** the reports' collection with scope "seen", then only the documents
  the viewer may read. Changes are the commits on their files, with the commit
  author as person, and list items written or ticked, with no person (Kanso
  does not record who ticked). Deleted documents are left out, since their
  rights cannot be checked any more. `internal/reports.Summarize` counts by
  kind, by hour (≤ 48 h) or day, and by person.
- **Charts** come from the reports' `reports.js` with d3: the stacked area
  learns hours and days, a spark line per kind is new. d3 now loads on report
  pages and the start page.
- No "Neu für dich": Kanso records no time for a share, so "shared since your
  last visit" cannot be told from "changed".

## Alternatives Considered

- **A separate activity log written on every change:** exact about who ticked
  what, but a second store beside Git that can drift, and empty for the past.
- **Charts as SVG from Go:** no JavaScript, but a second way of drawing that
  looks different from the reports.
- **Visits in the user files in Git:** a commit per visit, visible as activity.

## Consequences

- One more small file in the state directory; losing it only resets the
  window to 24 hours.
- The start page pays for the reports' collection on every load; the history
  scan is cached per head, the rest is walking the documents in memory.
- Who ticked a list item and when something was shared stay unknown until
  Kanso records them (BACKLOG).
