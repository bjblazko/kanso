# 019 — Task Attachments Next to the Board

**Status:** Accepted
**Created:** 2026-11-23
**Updated:** 2026-11-23

## Context

The tasks design (spec 2026-10-30) left attachments out: a note's photos
stayed with the note, and the task linked back to it. In use, a note with a
photo turned into a task lost the photo: only the author sees the note, and
the task looked incomplete even to them. Tasks render the same Markdown as
pages and notes, so people expect `![](attachments/…)` to work there too.

## Decision

Tasks get attachments like notes and wiki pages. They are stored in
`boards/<board>/attachments/<task>/`, in Git, next to the board's
`tasks/<task>.md`, which stays where it was. The editor uploads with the same
endpoint shape (`POST …/attachments`, JSON with the Markdown). Only someone
who sees the board can read or add a file. Moving a task to another board
moves the folder in a second commit. Deleting a task removes the folder in
the same commit as the task file.

Turning a note into a task copies the note's files into the task under the
same names and keeps the photo lines in the text. The note keeps its own
files.

## Alternatives Considered

- **A task as a folder (`tasks/<id>/index.md` plus `attachments/`).** This is
  tidier on disk, but it needs a migration of every task and changes every
  read path, the offline capture and the reports.
- **Keep the photos in the note and link to it.** That was the old state, and
  members of a group board cannot see a private note.
- **Move the photos from the note instead of copying them.** That would break
  the note, which stays the record of what was captured.

## Consequences

- Copying a note's photos doubles their size in the repository. Photos are
  small after `photo.Clean`, and this happens once per conversion.
- The reports do not yet list task attachments under "Verwaiste Medien"
  (backlog).
- Offline capture still has no photos.
