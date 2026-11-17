# Berichte (reports) — design

**Status:** draft for approval
**Created:** 2026-11-17

## Intent

The owner wants to *see* Kanso: how many documents of which kind exist, who
holds what, and above all **which resources are used** — how much space, where
it sits, whether deleted things still take space in the Git history, and
whether media lie around without anything pointing at them. The second reason
is plain fun: "visualisierung rockt". So the page is generous with charts, but
every chart answers a real question and every number is true.

Everyone gets reports. Admins additionally see the whole installation, broken
down by person. Non-admins see their own documents and, separately, what is
shared with them.

## Goals

- A "Berichte" page in the user menu, for everyone.
- All 14 charts of the approved mockup (`.superpowers/brainstorm/…/reports.html`),
  grouped in tabs, drawn with d3, each with the same numbers as a table.
- Resource answers: storage now and over time, Git history vs working tree,
  deleted-but-in-history, orphaned media, largest files, server process.
- Scopes: *Eigene*, *Mit mir geteilt*, *Alles, was ich sehe*; admins also *Alle*
  with a per-person breakdown.

## Non-goals

- Freeing space. Rewriting Git history or deleting orphaned media is a
  separate, risky feature; the reports only show. (Backlog item.)
- Export (CSV/PDF), saved report configurations, e-mailed reports.
- Live updating; a reload shows new numbers.
- Statistics about logins or page views — Kanso records neither, and must not
  start to for this (privacy, no telemetry rule).

## Anti-goals

- An admin learns *what* someone wrote. Admins see numbers and sizes about
  private things, never titles, file names, bodies or links (ADR-014). A title
  or file name appears only to someone who may read the thing.
- A number that looks precise but is not: sizes say "unkomprimiert" where they
  are, a cached report says from when it is.
- Slow pages: a report never blocks the rest of Kanso.

## Decisions taken with the owner

| Question | Decision |
|---|---|
| Admins: per-person numbers about private things? | **Yes, numbers only (A).** Every person's own report says openly: "Admins sehen von dir nur diese Zahlen, keine Titel oder Inhalte." |
| Which charts? | **All 14** of the mockup. |
| When are numbers computed? | **On request, cached until the next change (A).** |
| How do numbers reach the page? | **Server-rendered table + JSON in the page; d3 draws from the JSON.** Without JS the tables remain. |
| d3 | **Whole `d3.min.js` 7.9.0, vendored** (ISC), loaded only on report pages. Owner asked for d3; dependency check and license file still due before it lands. |
| Order of work | Overview and storage first, then the other tabs. |

## Approaches considered

**Computing.** (A, chosen) on request with a cache keyed by the Git HEAD plus a
change counter for the non-Git data; (B) nightly snapshot — always fast but a
day old, deletions show tomorrow; (C) incremental counters on every change like
the search index — fastest, most code, drifts when counters and files disagree.
A is simple and honest; the cost is one slower request after a change.

**Delivery.** One server render with table and embedded JSON. A separate JSON
API was dropped: a second source to secure and test for no gain.

**d3.** Whole bundle (chosen) vs. ~15 module files or a build step. Kanso has
no build tool on purpose; one vendored file matches mermaid.

## What the data really is (found in the code)

- **In Git:** `spaces/` (pages, recipes, attachments, `space.yaml`), `boards/`
  (tasks), `mealplans/`, `users/`, `kanso.yaml`.
- **Not in Git** (`.gitignore`): `notes/`, `todos/`, `avatars/`. Deleting a
  note or a list frees its space at once; no history exists for them.
- Pages: `created`, `updated`, `author` = *name* of the last editor (not a sub).
  Tasks: `author` = sub, `created`, `updated`, `done_at`, status, urgent,
  important, due, claimed. Notes: owner (folder), `created`, `updated`, `task`.
  List items: stamps for text and done. Meal plans: weekly files with recipe
  ids. Recipes: tags, source, time, servings.
- Git commit authors carry the user's name and e-mail (`users.User.Author`).

## Who a document "belongs to"

My call, to make "by person" possible where Kanso has no owner field:

| Kind | Person |
|---|---|
| Note, list | its owner |
| Task | `author` (sub) |
| Page, recipe, attachment | whoever's commit **added** the file (first commit), matched to a user by e-mail; unmatched → "Unbekannt" |
| Space | its first Owner member |
| Board, meal plan | belongs to a group, not a person: counted under the group name in "by person" charts, marked as group |

## Scopes

A row of links above the tabs, like the filters of "Was andere sehen":

- **Eigene** (default for non-admins): what belongs to me per the table above.
- **Mit mir geteilt:** what I can see and is not mine.
- **Alles, was ich sehe:** both.
- **Alle** (admins only; default for admins): everything, by person. Titles and
  file names only where the admin may read; the rest appears as counts and
  sizes, a title column says "– (kein Zugriff)".

Storage charts in a non-admin scope count only files of documents in that scope.
The process/server tile and the Git-repo-vs-working-tree line are admin-only
(they describe the installation, not a person).

## Tabs and charts

Route: `/reports/<tab>?scope=<scope>`; tab default `overview`. Each chart card:
title, one-line "what it answers", the chart, a `<details>` "Als Tabelle" with
the same numbers, hover tooltips.

1. **Übersicht** (`overview`)
   - Tiles: Dokumente (+ last 30 days), Speicher aktuell (of which
     attachments), Gelöscht in der Historie (admins; size, file count,
     largest), Verwaiste Medien (count, size); admins also Server: Kanso
     process memory, uptime, data dir size, Git pack size.
   - *Bestand nach Typ*: horizontal stacked bars, one bar per person in *Alle*,
     else one bar per type.
   - *Wachstum über die Zeit*: stacked area, cumulative per type by month,
     from `created` dates (Git first-add date for attachments).
2. **Speicher** (`storage`)
   - *Wo der Platz steckt*: treemap area → kind (Anhänge, Text, gelöscht in Git
     hatched). Areas: spaces, "Notizen", "Listen", "Boards", "Kochpläne".
   - *Speicher über die Zeit* (admins): Git-repo (cumulative unique blob bytes)
     vs. working tree (bytes of the files live at that time), monthly; the gap
     shaded. Both uncompressed; a note gives the real packed size.
   - *Verwaiste Medien*: attachments not referenced from their page's or note's
     Markdown, and attachment folders whose document is gone; name, where,
     size, link (only if readable).
   - *Größte Dateien*: top 20 files of the working tree, same visibility rule.
3. **Aktivität** (`activity`)
   - *Aktivitätskalender*: changes per day, last 12 months. Changes = Git
     commits touching documents in scope + note created/updated + list item
     stamps.
   - *Wann wird geschrieben?*: weekday × hour heatmap from the same events,
     server time zone.
4. **Wiki** (`wiki`)
   - *Wie alt ist das Wiki?*: pages by last change (<1 M, 1–3, 3–6, 6–12 M,
     1–2 J, >2 J), older bins in neutral grey, with a list of the 10 oldest.
   - *Wiki-Baum*: sunburst space → page → subpage, size = Markdown + attachments.
5. **Aufgaben** (`tasks`)
   - *Neu gegen erledigt*: last 16 weeks, mirrored bars (`created` above,
     `done_at` below).
   - *Eisenhower & Durchlaufzeit*: open tasks per quadrant, median days from
     created to done per quadrant (last 12 months), overdue count.
6. **Küche** (`kitchen`)
   - *Meistgekocht*: meal plan entries per recipe, last 12 months, top 10.
   - *Häufigste Einkäufe*: items ticked on shopping lists, normalised text
     (case, whitespace), top 15.
   - Recipes by tag and by preparation time (histogram) — small extras on the
     same tab.
7. **Freigaben** (`sharing`)
   - *Wer teilt mit wem*: force graph, people as dots, a line's weight = number
     of things both see (spaces, shared pages, lists, group boards, meal plans).
     Non-admins: only things they see, so only their part of the net.
   - Next to it the counts "ich teile" / "mit mir geteilt" by kind.

## Computation and caching

- Package `internal/reports`: pure functions from inputs (wiki index, notes,
  todos, tasks, meal plans, users, groups, a Git scan) to report structs. No
  HTTP, testable with fixtures.
- **Git scan** (one pass, cached by HEAD hash):
  `git log --reverse --raw --no-abbrev --format=…` for adds/deletes per commit
  with time and author, plus `git cat-file --batch-check` over the blob ids for
  sizes. From it: per-path first-add author/time, activity events, cumulative
  repo bytes, working-tree bytes over time, blobs not in HEAD ("deleted, in
  history"). Git is called as a program with an argument list (project rule).
- **Non-Git data** (notes, todos) read per request from the services — cheap.
- Cache: one entry per (HEAD, scope, viewer) for the Git-derived part, dropped
  on a new HEAD; computing happens under a single-flight so parallel requests
  wait for one scan. Memory bound: the scan keeps aggregates, not file lists,
  except top-N.
- The page renders "Stand: <time>" from the cache entry.
- Size of the data dir and `.git`: walk with `os.Stat`; `git count-objects -v`
  for the packed size.

## Frontend

- `web/static/vendor/d3/d3.min.js` 7.9.0 + `web/static/licenses/d3.txt` (ISC);
  added to "Lizenzen und Dank" and its completeness test.
- `web/static/js/reports.js`: reads `<script type="application/json"
  id="report-data">`, draws each `[data-chart]` with one function per chart
  kind; tooltips; respects `prefers-reduced-motion` (no force animation, the
  graph settles before drawing).
- Charts colour by role, tokens in `app.css`:
  categorical `--chart-1…6` = blue, aqua, yellow, magenta, green, violet —
  the reference palette **without orange** (orange is Kanso's primary action).
  Validated: light on `#F4F4F0` all checks pass, worst adjacent CVD ΔE 9.1,
  contrast WARN for aqua/yellow/magenta → relief by direct labels or the table;
  dark on `#161718` all pass, CVD ΔE 8.4. Sequential = one hue from `--bg-2` to
  blue. Hatch for "gelöscht". One series → single blue, no legend.
- Recorded in `docs/design.md` under "Deviations from huepattl-rams-design":
  chart colours are identity, not signal, only inside charts.
- Layout: cards in a grid (two columns from ~64rem, one below); tabs as links
  (`aria-current`), scopes as the quiet filter row of "Was andere sehen".

## Privacy and visibility

- Report numbers are computed over what the scope allows; *Alle* only for
  admins (`requireAdmin`).
- Any title, file name or link passes the same read check as the rest of
  Kanso (`pageRole`, list visibility, board visibility).
- The note "Admins sehen von dir nur diese Zahlen …" on every non-admin report.
- Nothing leaves the machine; no new cookies or storage.

## States

- Empty installation / empty scope: tiles show 0, charts say "Noch nichts
  da." instead of an empty axis.
- First request after a change: computed in the request; a slow scan (>1 s)
  is fine, no spinner needed beyond the browser's.
- Git scan error: the Git-based charts say "Historie gerade nicht lesbar" and
  log the error; the rest renders.
- Very many people (>6) in a stacked chart: top 5 + "Andere".
- No JS: tables only, charts hidden.

## Testing

- `internal/reports`: unit tests per computation with fixed fixtures (a temp
  Git repo with adds, edits, deletes; notes; tasks).
- `internal/web`: HTTP tests per tab and scope — numbers in the tables, admin
  sees *Alle*, non-admin gets 403 for `scope=all`, an admin's *Alle* view
  never contains a title of a page they cannot read, the "Admins sehen" note.
- Playwright: the menu entry, switching tabs and scopes, a chart SVG appears,
  table view opens; desktop and phone.

## Docs

ADR-015 "Reports computed on request, cached per Git HEAD" (and the admin
numbers decision, referencing ADR-014); arc42 building block `reports` and
runtime flow; design.md rows for the Berichte screens and the deviation;
feature doc; CHANGELOG; BACKLOG: freeing history space / deleting orphans.

## Left to the builder

Exact JSON shapes, chart function boundaries, the Git log format string, the
normalisation of shopping items beyond case/whitespace, tooltip wording.
