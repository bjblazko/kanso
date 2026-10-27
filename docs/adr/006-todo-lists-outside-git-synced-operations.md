# 006 — To-do Lists Outside Git, Synced as Operations

**Status:** Accepted
**Created:** 2026-10-27
**Updated:** 2026-10-27

## Context

Small plans ("Gartenarbeit", "Weihnachten") need a plain list to tick on the
phone, often without network, sometimes shared with family members. A task
list in a wiki page would mean a Git commit per tick and an edit lock; the
Kanban boards of a later phase are too heavy. Ticked items must stay where
they are. Several people may change a shared list on several devices, some of
them offline.

## Decision

- To-do lists are their own part of Kanso (`/todos`), next to notes and wiki.
- One JSON file per list in `todos/<id>.json`, **not in Git** (`/todos/` in
  the data repository's `.gitignore`). Lists are flat (no folders); items have
  one level of sub-items.
- Every change is an **operation** made on the device (create, rename, add,
  text, done, sub, delete) with a device id and device time. The page applies
  it to its copy (IndexedDB) at once and keeps it in an outbox until the
  server confirmed it. A sync sends all waiting operations and gets back the
  current state of every visible list; waiting operations are applied again
  on top.
- Conflicts: per field of an item the later change wins; delete always wins,
  and tombstones (kept 30 days) stop late operations from reviving an item.
  Sending an operation twice changes nothing.
- The list owner may rename, share (people or Kanso groups, no read-only
  shares) and delete; everyone it is shared with may tick and edit items.
  Admins have no extra access.
- The page is drawn in the browser so it opens offline; the service worker
  serves it for `/todos` and `/todos/<id>`. Sharing and deleting a list are
  server pages, online only.

## Alternatives Considered

- **Task lists in wiki pages:** a commit per tick, an edit lock per page,
  no offline use. Rejected by the owner.
- **One file per item** (as planned for shopping lists): fewer write
  conflicts, but a list's order would need its own file; one file per list
  is simpler for a household and a whole list syncs in one answer.
- **Last write wins per item or per list:** loses a tick when someone else
  changed the text at the same time.
- **Server-rendered pages with JavaScript on top:** would not open offline.

## Consequences

- To-do lists have no history; a deleted list is gone (backups of the data
  directory still hold it).
- Device clocks decide conflicts; a badly wrong clock can lose a change.
- To-do lists need JavaScript.
- Waiting changes live in browser storage, which a browser may evict
  (Kanso asks for persistent storage; installed home-screen apps are safer).
