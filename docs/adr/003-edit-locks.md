# 003 — Edit Locks Instead of Live Collaboration

**Status:** Accepted
**Created:** 2026-10-26
**Updated:** 2026-10-26

## Context

Docmost lets several people edit one page live. Kanso stores plain Markdown
files and is built without frontend frameworks; live co-editing (CRDT/OT)
would be large and fit plain files poorly.

## Decision

Opening a page for editing takes a lock. The editor renews it every minute;
it expires five minutes after the last renewal. Others see who is editing;
the space owner or an admin can break the lock. Saving without a valid lock
is refused (HTTP 409) and the text stays in the browser. Locks live in
memory only. Lists, boards and tasks store one file per entry and need no
locks.

## Alternatives Considered

- **Live collaboration:** best experience, far too much code for the gain.
- **Detect conflicts on save and merge:** no waiting, but merging Markdown
  by hand is unfriendly for family members.

## Consequences

- A restart releases all locks; editors keep their text and take the lock again.
