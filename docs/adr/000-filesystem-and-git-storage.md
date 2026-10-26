# 000 — Files in a Git Repository Instead of a Database

**Status:** Accepted
**Created:** 2026-10-26
**Updated:** 2026-10-26

## Context

Kanso is for small systems with few users (a family). Docmost, which it
replaces, needs Postgres and Redis. Content should stay readable and
recoverable without the application.

## Decision

All content lives in a data directory: Markdown with YAML frontmatter,
YAML for configuration, iCalendar files for events. The data directory is a
Git repository; every save is one commit with the acting user as author.
Git is used through the `git` program. Runtime-only state (sessions, keys)
lives in a separate state directory. Indexes (page ids, search) are built in
memory at start.

## Alternatives Considered

- **SQLite:** robust, but content is no longer plain files.
- **Copies in a `.history/` folder:** no extra tool, but diffs and moves
  would have to be built.
- **go-git library:** no external program, but a large dependency.

## Consequences

- History, diff and restore come from Git; backups are plain files.
- Writes are serialised by one lock; fine for a handful of users.
- `git` must be installed where Kanso runs.
