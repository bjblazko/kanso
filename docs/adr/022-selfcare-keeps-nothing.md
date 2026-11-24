# 022 — Selfcare Keeps Nothing and Points to Help

**Status:** Proposed
**Created:** 2026-11-24
**Updated:** 2026-11-24

## Context

Selfcare offers first aid for stressful moments: a random small step, a
battery check and guided exercises. How someone feels is health data
(Art. 9 GDPR), and the household shares one Kanso. A self-help page can also
meet someone in a real crisis.

## Decision

The content is fixed in Go (`internal/selfcare`, German and English) and
written for Kanso, not taken from other sources. Neither the battery answer
nor the steps someone views are stored, not even on the device. Only when the
person presses "Als Notiz speichern" do the answers of a check-in or "Eine
gute Sache" become a private note. Every Selfcare page shows "Hilfe von
außen" (TelefonSeelsorge and 112). With an empty battery, that block comes
first.

## Alternatives Considered

- **A mood diary with charts:** useful over time, but it keeps health data
  that a shared server, its backups and the admins could see.
- **Content in the wiki:** easy to edit, but random picks and guided steps
  need structure, and a deleted page would break the function.

## Consequences

- Kanso shows no history of how someone felt. The person decides what to
  keep, as a note.
- Changing the content needs a release.
