# 015 — Reports Are Computed on Request, Cached per Git HEAD

**Status:** Accepted
**Created:** 2026-11-17
**Updated:** 2026-11-17

## Context

The owner wants "Berichte": how many documents of which kind, who holds what,
and above all which resources Kanso uses — storage now and over time, what
the Git history keeps of deleted things and old versions, orphaned media. The
expensive part is the history: every commit, every file change, every blob
size. Notes and lists live outside Git (ADR-004, ADR-006) and are cheap to
read. Admins want the whole installation by person; ADR-014 says admins do
not read what they are not entered for.

## Decision

- Reports are computed **when a report page is requested**. The Git part (one
  `git log -M --raw` pass, blob sizes via `git cat-file --batch-check`, the
  files at HEAD, `git count-objects`) is **kept in memory until HEAD moves**.
  Requests that arrive while a scan runs wait for it; a failed scan is not
  kept. Notes, lists and the wiki index are read per request.
- A package `internal/reports` holds the computations as pure functions over
  plain inputs; `internal/web` collects the inputs per viewer and scope.
- Every report has scopes: own, shared with me, all I see; admins also
  **everyone, by person — numbers and sizes only**. Titles, file names and
  links appear only for things the viewer may read; everyone else's report
  says so ("Admins sehen von dir nur diese Zahlen, keine Titel oder Inhalte.").
- A file's person is whoever's commit added it first; a move keeps the person
  (rename detection). Notes and lists belong to their owner, tasks to their
  author, a group's boards and plans to the group.
- Charts are drawn in the browser with vendored d3 from JSON in the page; the
  same numbers are a table under each chart.

## Alternatives Considered

- **A nightly snapshot** in the state directory: always fast, but a day old;
  a deletion shows tomorrow. Rejected: the owner looks at reports right after
  cleaning up.
- **Counters kept on every change**, like the search index: fastest, but most
  code, and counters drift from the files. Rejected.
- **Private things only as one sum for admins**: closer to ADR-014, but "who
  takes the space" could not be answered. Rejected by the owner in favour of
  numbers per person with an open note.

## Consequences

- The first report after a change scans the history: about a second for a
  large family repository; later ones are instant until the next commit.
- Sizes from Git are uncompressed; the report says so and shows the packed
  size beside them.
- Admins learn how many private notes someone has and how large they are —
  never what they say.
