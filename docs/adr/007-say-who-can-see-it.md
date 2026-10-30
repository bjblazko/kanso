# 007 — Every Page Says Who Can See It

**Status:** Accepted
**Created:** 2026-10-29
**Updated:** 2026-10-30

## Context

Kanso mixes private things (notes, own to-do lists) with shared ones (spaces
and their pages, shared lists). Who sees what follows from rules that live
elsewhere: space members, groups, shares, the admin role. Before writing
something down, a person must know whether the family, a guest or nobody else
will read it, also where nothing can be shared at all.

## Decision

- Every place that shows content carries one line under its heading (in
  lists: in each row) that says who can see it: a lock icon and "Nur für dich
  sichtbar", or a people icon and "Sichtbar für …". Same words, same look
  everywhere; the server renders it as the `audience` partial, `todos.js`
  draws the same markup.
- Whoever may change who sees it gets "Freigabe ändern" at the end of the
  line (2026-10-30): space owners → space settings, the owner of a to-do
  list → its share page, admins on a group board → groups. Where nothing can
  be shared (notes, private boards) there is no link.
- **Spaces and pages:** people with a role, members of a group listed as the
  people they are, sorted by name, the viewer first as "dich", and always
  "und Admins": admins read every space, but Kanso learns who they are only at
  their sign-in, so they stay a role, not names. A space without members says
  "Sichtbar nur für Admins". Pages inherit their space's line.
- **Notes:** "Nur für dich sichtbar" on the list, the capture page and a note.
- **To-do lists:** the owner and everyone the list is shared with (groups by
  name), the viewer as "dich"; an unshared list "Nur für dich sichtbar".
- "Only you" means within Kanso. Whoever runs the server can read the files;
  that is said in the README, not on every page.

## Alternatives Considered

- **A count ("4 Personen").** Shorter, but the question is *who*; families are
  small enough to name everyone.
- **Show it only where something can be shared.** Leaves the private parts
  unexplained; whether a note is private is exactly what people wonder.
- **Name groups instead of people in spaces.** Group names hide who is in
  them; people are what matters. To-do shares keep the group name, because a
  share is given to the group and its members change.

## Consequences

- Every new part of Kanso needs its line; a missing one is a defect.
- The line is computed per request from space members and groups, which is
  cheap for a family.
- Admins are not named; the wording stays honest about that.
