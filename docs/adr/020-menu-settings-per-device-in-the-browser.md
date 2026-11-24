# 020 — Menu Settings per Device, in the Browser

**Status:** Accepted
**Created:** 2026-11-24
**Updated:** 2026-11-24

## Context

More parts are coming (Fokus first). A fixed rail of six fits a phone's
bottom bar only just. The owner wants each person to hide parts they do not
use and to order the rest, per device and without a cookie.

## Decision

The rail is one list in Go (`railParts`) in a default order; new parts go
last. The browser keeps `kanso.menu.<sub>` = `{order, hidden}` in
`localStorage`. `menu.js`, a plain script right after the rail, puts the parts
in order and hides some before the page is drawn. The CSP needs no inline
script for this. Unknown keys are dropped, new parts are appended, and Start
cannot be hidden. On phones more than five visible parts become four and
"Mehr", which also leads to `/me/menu`. On wide screens the column scrolls.
Hidden parts stay reachable by URL.

## Alternatives Considered

- **A cookie read by the server:** no reordering after load, but the owner
  wants no cookie.
- **The server-side profile:** the same on every device. The owner wants a
  choice per device, and it would be one more write path.

## Consequences

- A new device starts with the default menu.
- Without JavaScript the menu shows all parts in the default order.
- Storing settings on the device falls under § 25 TDDDG just as a cookie does.
  It is exempt because the person asked for it, so no consent banner is
  needed.
