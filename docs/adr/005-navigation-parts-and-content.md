# 005 — Navigation: Parts of Kanso and Their Content

**Status:** Accepted
**Created:** 2026-10-27
**Updated:** 2026-10-27

## Context

Kanso grows from a wiki into a family all-rounder: notes, wiki, then tasks,
lists, calendar and extras (spec, phases). The first navigation was one side
bar listing "Notes", the wiki's spaces ("Haus", …) and "Groups" side by side.
With more spaces and parts, what a thing *is* (a part of Kanso) mixed with
what is *in* it (a wiki space); spaces were not recognisable as part of the
wiki, and an administration page sat among content. Kanso is mobile first.

The owner chose among three proposals (2026-10-27): A, a sectioned side bar;
**B, a rail of parts plus a content side bar**; C, tabs at the top.

## Decision

- **Rail of parts:** a fixed rail names the parts of Kanso, each with icon and
  label: today *Notizen* and *Wiki*; tasks, lists and calendar join when they
  exist (no entries for parts that are not built). The current part is marked
  (`aria-current`). On phones the rail is a bar at the bottom, within thumb
  reach; on wide screens a column at the left. Beyond five parts the last
  entry becomes "Mehr".
- **Content side bar per part:** the side bar shows only what is in the
  current part, under that part's heading — for the wiki: "Wiki-Bereiche",
  the spaces and the open space's page tree. Parts without inner structure
  (notes) have no side bar.
- **Administration is not content:** group management moves to the user menu
  ("Gruppen verwalten", admins only).
- **Every part has its own URL** (`/notes`, `/`, later `/tasks`, …) so other
  pages (the homelab start page) can link straight to a part.

## Alternatives Considered

- **A — sectioned side bar:** smallest change, but grows with every part and
  sits behind "Menü" on phones (two taps to reach anything).
- **C — tabs at the top:** clear on desktop, but on phones a horizontally
  scrolling strip at the top, far from the thumb.

## Consequences

- Phones keep a bar of about 3.5 rem at the bottom; pages leave room for it
  (the capture field is sized so "Speichern" stays above it).
- New parts must bring an icon, a label and a URL, and decide whether they
  need a content side bar.
- The wiki's start page is `/`; it lists the wiki spaces.

## Related

- [Feature: navigation](../features/2026-10-27%20navigation.md)
- `docs/design.md` — screens and their jobs
