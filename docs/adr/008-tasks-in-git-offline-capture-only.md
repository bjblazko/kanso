# 008 — Tasks in Git, Offline Only for Capture

**Status:** Accepted
**Created:** 2026-10-30
**Updated:** 2026-10-30

## Context

Many things come up "between door and hinge" and fit no to-do list: the tax
return, sweeping the yard, signing the child up for holiday care. They must be
written down on the phone in seconds, also without network, and sorted later
at the desk on a Kanban board and an Eisenhower matrix. Each person writes
down what they take on; a family board holds what anyone may pick up. Notes
(ADR-004) and to-do lists (ADR-006) live outside Git; tasks are less private
and the owner wants to see who changed what.

## Decision

- Boards live in `boards/<id>/board.yaml` (name, owner: one person or one
  Kanso group), each task in `boards/<id>/tasks/<id>.md` (frontmatter + Markdown
  text), **in the Git history**: one commit per change, authored by the person,
  also for dragging a card.
- One private board per person, made on first use; group boards are made by
  admins. Admins have no access to private boards and see a group board only
  as members of its group.
- Fixed columns `open`, `doing`, `waiting`, `done`; `urgent` and `important`
  place a task in the matrix. Kanban and matrix are two views of the same files.
- No edit locks: the edit form sends the version it read; a changed file is
  not overwritten (409). Dragging changes one field and always wins.
- Only **capture** works offline, with an outbox like notes (ADR-004); the
  device makes the task id, so a second upload is recognised. Boards and the
  edit page need the network.

## Alternatives Considered

- **Outside Git with synced operations (as to-do lists):** no history of who
  changed what; boards are desk work, where the network is there.
- **Fully offline boards:** several times the work (operations, conflicts per
  field) for a case the owner does not have.
- **Lanes configurable per board:** the owner wants fixed columns.
- **One file per board:** every change would rewrite and conflict on one file.

## Consequences

- A deleted task stays in the history; deleting is not without trace (README).
- Every drag is a commit; the history grows with use.
- A task waiting in the outbox for a board that is gone lands on the person's
  own board.
