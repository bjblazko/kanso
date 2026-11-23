# Aufgaben: Karten, Etiketten, Avatar, Anhänge – Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Board cards look like Kanban cards, carry "dringend"/"wichtig"
labels and an avatar on every card, and tasks get attachments, including the
photos of a note turned into a task.

**Architecture:** The cards are a template and CSS change in
`tasks_board.html`/`app.css`. `card.Claimant` becomes `card.Face`, which falls
back to the author. Attachments live in `internal/tasks` (`attachments.go`)
under `boards/<board>/attachments/<task>/`. `Move`/`Delete` take the folder
along. Web handlers mirror the note attachments, and the task editor gets
`data-upload-url`.

**Tech Stack:** Go stdlib, html/template, plain CSS/JS, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-23-task-cards-attachments-design.md`

## Global Constraints

- Every new source file starts with `SPDX-License-Identifier: Apache-2.0`.
- No new dependency. Functions ~30 lines, cyclomatic ≤ 10.
- Git as a program with an argument list (via `store.Repo`).
- UI rules from `huepattl-rams-design` and `docs/design.md`. Signal red only for
  "dringend". No shadow except while dragging.
- Upload limit `maxAttachment` (50 MiB), names via `store.SaveUpload`/`ValidAttachmentName`.
- Docs in the same commit: CHANGELOG `[Unreleased]`, ADR 019 (+ arc42 ch. 9),
  feature doc tasks, `docs/design.md` rows Board/Task.

## Review Focus

- Someone who doesn't see the board asks for an attachment → 404 (test in Task 3).
- Moving a task with attachments to another board → images still show (Task 3).
- A note photo whose name already exists in the task folder → no overwrite (`SaveUpload` picks a free name; Task 4 test uses a photo-only note).
- Avatar for an author who has since left the group → still shown, name from the profile (no test needed: `userName` falls back to sub).
- Dragging a card whose whole surface is a link → drag still works and does not open the task (Playwright drag test in `tasks.spec.js` stays green).

---

### Task 1: Avatar on every card (claimant, else author)

**Files:** Modify `internal/web/handlers_tasks.go`, `web/templates/tasks_board.html`, `web/templates/task_view.html`; Test `internal/web/face_test.go`

**Interfaces:** Produces `card.Face *face` (always set). `card.Claimed` stays as the text for screen readers / task page.

- [ ] Test `TestCardShowsAuthorAvatarWhenUnclaimed`: a task by "anna" (name "Anna Becker") on her private board, unclaimed → board HTML contains `class="face" title="Anna Becker"` and `>AB<`. Adjust `TestCardShowsClaimantAvatar` to the field name.
- [ ] Run, FAIL. Implement: `c.Face = s.face(cmp.Or(t.Meta.Claimed, t.Meta.Author))`; templates use `$c.Face`; the screen-reader text is `Claimed`, else `tasks.created_by_short` ("von %s" / "by %s", i18n de+en). Task page: row "Übernommen" unchanged; the avatar also sits in front of "Angelegt von".
- [ ] Run `go test ./internal/web`, PASS. Commit.

### Task 2: Cards and labels

**Files:** Modify `web/templates/tasks_board.html`, `web/static/css/app.css`, `internal/i18n/{de,en}.yaml` (keys `tasks.mark_urgent`/`mark_important` → words "dringend"/"wichtig", "urgent"/"important"); Test `internal/web/handlers_tasks_test.go`, `e2e/tests/tasks.spec.js`

- [ ] Test `TestCardLabels`: an urgent+important task → HTML contains `class="tag tag-urgent">dringend<` and `class="tag">wichtig<`. No `<abbr` in the card.
- [ ] Run, FAIL. Implement the card structure: `.card-tags` (top), `a.card-title` with `::after { inset: 0 }` as a stretched link (the card is `position: relative`), `.card-foot` (meta left, face right). CSS: padding `--space-3`, radius `--radius-md` (4 px), hover `--hover-overlay`, `.card.dragging` gets `--shadow-float`. `.tag`: Plex Sans `--text-xs` 500, 1 px border `--line-strong`, radius `--radius-sm`. `.tag-urgent`: `color: var(--warning-ink)`, border in the same colour.
- [ ] `go test ./internal/web` + `npx playwright test tests/tasks.spec.js` (the drag test must stay green), PASS. Screenshot desktop/phone via Playwright, review against Rams. Commit.

### Task 3: Store attachments

**Files:** Create `internal/tasks/attachments.go`, `internal/tasks/attachments_test.go`; Modify `internal/tasks/write.go` (`Move`, `Delete`)

**Interfaces – Produces:**
- `func (s *Store) AddAttachment(a Actor, boardID, taskID, filename string, r io.Reader, max int64) (string, error)`
- `func (s *Store) OpenAttachment(a Actor, boardID, taskID, name string) (*os.File, error)` – ErrNotFound for an invisible board or a bad name
- `func (s *Store) CopyAttachments(a Actor, t Task, srcDir string, names []string) error` – copies files, one commit
- `ErrTooLarge = store.ErrTooLarge`
- Path: `attachmentRel(board, task) = boards/<board>/attachments/<task>`

- [ ] Tests: `TestAttachmentAddOpen` (round trip, commit exists), `TestAttachmentInvisibleBoard` (ben opens anna's private one → ErrNotFound), `TestAttachmentBadName` (`../x` → ErrNotFound), `TestMoveTakesAttachments` (after Move, Open on the new board works, the old path is gone), `TestDeleteRemovesAttachments`, `TestCopyAttachments`.
- [ ] Run, FAIL. Implement; in `Move`, if the folder exists, `s.repo.Move(author, oldRel, newRel, msg)`; in `Delete`, `os.RemoveAll` + commit on both paths.
- [ ] `go test ./internal/tasks`, PASS. Commit.

### Task 4: Web: upload, serve, editor, note → task

**Files:** Create `internal/web/handlers_task_attachments.go` + test; Modify `internal/web/routes.go`, `web/templates/task_edit.html`, `web/static/js/editor.js` (remove the comment "tasks have no attachments"), `internal/web/handlers_notes_task.go`, `internal/tasks/views.go` (`FromNote` keeps photo lines), `internal/web/handlers_notes_task_test.go`

**Interfaces – Consumes:** Task 3. Routes: `POST /tasks/b/{board}/{task}/attachments`, `GET /tasks/b/{board}/{task}/attachments/{name}`.

- [ ] Tests: `TestTaskAttachmentUploadAndServe` (upload → JSON with `markdown`, GET 200 with the bytes, another person without the board → 404), `TestTaskEditOffersUpload` (`data-upload-url=` in the editor), rename `TestNoteToTaskDropsPhotoLinks` → `TestNoteToTaskCopiesPhotos` (body contains `![](attachments/zettel.png)`, GET of the attachment on the task delivers "png", the note still has its attachment), `FromNote` unit test in `internal/tasks/views_test.go` adjusted (the photo line stays in `rest`).
- [ ] Run, FAIL. Implement: handlers like `uploadNoteAttachment`/`serveNoteAttachment`; editor form gets `data-upload-url`, `data-msg-upload-failed`, upload label + `input#file` like `note_edit.html`. `noteToTask` calls `CopyAttachments` with the note's attachments after `Create` (photo-only note: title as before, body = photo lines).
- [ ] `go test ./...`, PASS. Playwright: in `tasks.spec.js` "a note becomes a task" with a photo (`setInputFiles` with `e2e/fixtures` image, if present, else a small PNG buffer) → the task page shows `img`. PASS. Commit.

### Task 5: Docs and release

- [ ] ADR `docs/adr/019-task-attachments.md` (Context: spec 2026-10-30 ruled out attachments; Decision; Alternatives: task as folder, keep link to note; Consequences), list it in arc42 ch. 9, bump `**Updated:**`. Mark the old spec line 246 as superseded by ADR-019.
- [ ] Feature doc `docs/features/2026-10-26 tasks.md` (updated, text), `docs/design.md` Board/Task rows, CHANGELOG `[Unreleased]`. Check `internal/reports/storage.go` orphans: if task attachments are not counted, add a backlog entry.
- [ ] `just lint`, `just test`, `just e2e` (tasks + notes), PASS. Commit; release (0.12.0), deploy as in 0.11.3 after a legal check.
