# 004 — Notes Outside Git, Captured Offline

**Status:** Accepted
**Created:** 2026-10-27
**Updated:** 2026-10-27

## Context

Notes are scratch paper written on the phone, often without network, often
with a photo. They are sorted later by hand: deleted or turned into a wiki
page. Everything else in Kanso lives in the Git history (ADR-000), where a
deleted file stays forever — including a note that held a password. The
capture must also work offline, and Kanso has no frontend framework (ADR-002).

## Decision

- Notes live in `notes/<sub>/<id>/` in the data directory, **not in Git**:
  Kanso adds `/notes/` to the data repository's `.gitignore`. Deleting a note
  removes it for good. A note turned into a wiki page is moved into the space
  and committed then; its history starts there.
- Only the author can see a note; admins cannot either.
- Offline capture uses a hand-written service worker (capture page and its
  files) and an IndexedDB outbox on the device. Each note gets its id on the
  device, so uploading it again after a lost answer is recognised and does
  not create a second note. A note leaves the outbox only after the server
  confirmed it.
- JPEG photos lose their GPS data on upload (notes and wiki), keeping all
  other EXIF data such as the orientation.

## Alternatives Considered

- **Notes in Git like everything else:** history for notes, but "deleted"
  would not mean deleted. Rejected by the owner (2026-10-27).
- **Background Sync API:** not available in Safari; the page uploads when it
  is open (on save, on load, when the network returns, every 30 s).
- **A service-worker library (Workbox):** a dependency for ~60 lines of code.
- **Removing all EXIF data:** photos would show rotated.

## Consequences

- Backups of the data directory still contain notes (they are files).
- Notes wait in the browser until uploaded; a browser may evict that storage
  (Kanso asks for persistent storage). The page shows what is waiting.
- Offline behaviour is tested in Chromium; Playwright's WebKit does not run
  service workers reliably, so iOS is checked by hand.
