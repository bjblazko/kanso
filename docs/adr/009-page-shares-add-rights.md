# 009 — Page Shares Add Rights

**Status:** Accepted
**Created:** 2026-10-30
**Updated:** 2026-10-30

## Context

A space usually belongs to one person or group. Sometimes one page of it
must reach someone else, without the rest of the space: a wish list from
"Timos Space" for Christina. Until now only the whole space could be shared,
and "Freigabe ändern" on a page opened the space's settings.

## Decision

- A page carries its own `members` in its frontmatter, the same shape as a
  space's members (person or Kanso group, role viewer/editor/owner).
- A person's role on a page is the **highest** of their space role and the
  members of the page and every page above it. Subpages inherit; a page share
  only adds rights, it never takes any away. Admins stay owners everywhere.
- Every check that used the space role for a page uses the page role: page,
  attachments, history, preview, editing, moving, creating subpages, search,
  the page tree in the navigation, note → wiki page targets.
- Someone who reads a page only through a share sees, in the wiki navigation
  and on the wiki start page, a section "Für dich freigegeben" with those
  pages and their subpages, and nothing else of the space.
- Changing a page's members needs the owner role on that page (space owners,
  admins, or a page share with "Verwalten"); `/p/<id>/share` edits them.

## Alternatives Considered

- **Shares in the space file:** would not move with the page and makes the
  space file a list of exceptions.
- **Shares that also restrict (deny for some space members):** harder to
  explain and to see; the owner asked only for exceptions that open.
- **A rail entry "Für dich freigegeben" across all parts:** a fifth entry on
  the phone bar for something rare; shared to-do lists already show "von …"
  in their overview.

## Consequences

- A page's members move with it; a subpage moved out of a shared page loses
  what it inherited.
- Role checks walk the page's ancestors (in memory); fine for a household.
- The line that says who can see a page names the page's people too, and
  its "Freigabe ändern" link leads to the page's sharing, not the space's.
