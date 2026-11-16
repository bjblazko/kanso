# 014 — Admins Manage Spaces but Do Not Read Them

**Status:** Accepted
**Created:** 2026-11-16
**Updated:** 2026-11-16

## Context

Since [ADR-001](001-oidc-with-kanso-managed-roles.md) admins were owners of
every space. A page in a space without members therefore said "Sichtbar nur
für Admins", even for the admin who wrote it, and every member of the
Authelia group `kanso-admin` could read every wiki page. The owner wants
"visible to" to mean exactly the people entered for it, admins included.

## Decision

Being admin no longer gives a role in a space: `access.Resolve` returns only
what the person or their groups are entered for. Whoever creates a space
becomes its owner. Admins keep the administration: creating spaces, groups,
boards, meal plans and the AI connection, and for every space its settings
(members, name, deletion) through "Alle Bereiche" (`/admin/spaces`), which
lists each space with its members and page count but never its content. A
space without members is offered there with "Mich als Besitzer eintragen";
that is a commit like every other change of members. The visibility line
names only members: "Nur für dich sichtbar", "Sichtbar für dich, Anna",
"Niemand hat Zugriff".

## Alternatives Considered

- **Keep admins as readers** and only say "dich und Admins": honest, but an
  admin still reads everything, which is what the owner did not want.
- **Admins see nothing of foreign spaces:** an orphaned space could only be
  rescued by editing `space.yaml` by hand.
- **Take over orphaned spaces automatically at sign-in:** Kanso knows admins
  only from their session, and an automatic read right is what this decision
  removes.

## Consequences

- After the update, spaces without members disappear from everyone's wiki
  until an admin takes them over in "Alle Bereiche".
- An admin can still give themselves access, visibly in the space's members
  and in the Git history.
- Search, kitchen and meal plans follow, because they all ask `Resolve`.
