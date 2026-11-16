---
status: done
created: 2026-11-16
updated: 2026-11-16
---

# Fold the spaces bar

On wide screens the wiki's spaces bar (spaces and the open space's pages) can
be folded to a narrow strip, so a page gets the width. Its head is the
`<details>` disclosure that phones already use: "Wiki-Bereiche" and a panel
icon. `nav.js` stores the state on the device in the cookie `kanso_sidenav`
(`open`/`closed`, one year, `SameSite=Lax`, only on wide screens); the server
reads it and draws the next page folded or open, without a jump. The cookie
never leaves the browser and Kanso's own server.

## Related

- [Design](../design.md) (Layout)
