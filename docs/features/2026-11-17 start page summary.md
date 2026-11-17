---
status: done
created: 2026-11-17
updated: 2026-11-17
---

# Start page: what happened

Kanso's start page ("Start", first in the rail, at `/`) shows what changed since one's last visit (at least the
last 24 hours): one sentence ("Seit gestern, 18:40: 4 Seiten geändert, 2 neue
Rezepte, 7 Listeneinträge abgehakt."), a number with a spark line per kind,
the changes over time stacked by kind, who was active, and the changed
documents grouped by space, newest first.

A select switches to 24 hours, 7 or 30 days; "Eigene einblenden" adds one's own
changes, which are hidden at first. Below, "Heute" lists today's meals of one's
cooking plans and the tasks due today or overdue. The wiki's spaces are under
"Wiki" (`/wiki`).

Only documents one may read count, for admins too. Who ticked a list item is
not recorded, so those count under "ohne Namen".

## Related

- [ADR-018](../adr/018-start-page-summary-from-visits-and-reports.md) — window and data
- [ADR-015](../adr/015-reports-on-request-cached-per-head.md) — the reports it builds on
- [ADR-014](../adr/014-admins-manage-but-do-not-read.md) — admins see only what they may read
- Spec: `docs/superpowers/specs/2026-11-17-home-summary-design.md`
