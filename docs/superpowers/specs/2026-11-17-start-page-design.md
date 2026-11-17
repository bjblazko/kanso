# Start page of its own — design

**Status:** approved 2026-11-17 (short design)

## Why

0.10.0 put the summary of what happened on `/`, which is the wiki's page:
the rail calls it "Wiki" and its title is "Wiki-Bereiche". The owner wanted a
start page for all of Kanso, not a part of the wiki.

## Goals, non-goals

- Goal: `/` is the start page — the summary and "Heute", nothing of the wiki.
- Goal: the wiki's spaces list moves to `/wiki`; the rail's "Wiki" points there.
- Non-goal: new content on the start page (quick actions etc.) — the owner
  said: nothing more.

## Approaches

- **A (chosen): `/` is the start page, the wiki moves to `/wiki`.** A new first
  rail entry "Start"; the brand and the sign-in land there.
- B: start page at `/start`, `/` stays the wiki — less moving, but one would
  still land in the wiki after signing in.

## Design

- Rail: "Start" first (icon `home`, module `start`), then Notizen, Todos,
  Aufgaben, Wiki (`/wiki`), Küche. `moduleOf("/")` is `start`; `/wiki` belongs
  to the wiki module.
- `/` renders a new page `start`: screen head with h1 "Was passiert ist", the
  window select and the own toggle in its toolbar; below the sentence, tiles,
  charts, "Zuletzt geändert" and "Heute" as in 0.10.0. The visit is recorded
  as before.
- `/wiki` renders the old page without the summary: spaces with audiences,
  "Für dich freigegeben", the admin's "Bereich anlegen". Deleting a space
  redirects to `/wiki?deleted=…`.
- Links meaning "the wiki" (`/` in the wiki's breadcrumbs, the error page's
  back link stays `/`) point to `/wiki`.
- Release as 0.10.1 under "Changed".

## Left to the builder

Which existing tests move from `/` to `/wiki`; the exact toolbar layout per
`huepattl-rams-design` and the screen-head rule.

Records: ADR-018 updated (where the page lives), feature doc, arc42, CHANGELOG.
