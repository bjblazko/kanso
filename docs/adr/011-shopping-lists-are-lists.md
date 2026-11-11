# 011 — Shopping Lists Are a Kind of List

**Status:** Accepted
**Created:** 2026-11-07
**Updated:** 2026-11-11

## Context

Recipes (ADR-010) should fill a shopping list, with amounts added up. The
person who shops is usually not the one who cooks: each item must say what it
is for and may carry alternatives. In the shop the list must work offline,
be shared with the family and be ticked by several people — exactly what the
to-do lists (ADR-006) already do. The overall design planned shopping lists
as their own list type with one file per item.

## Decision

- A shopping list is a to-do list with `kind: shopping` in its file. It
  syncs, works offline and is shared exactly like any list (ADR-006).
- Items of a shopping list may carry a `note` (alternatives, hints; up to 300
  characters, changed by the new operation `note`) and `for`, the recipes
  they come from with the amount each asked for. Only the server sets `for`.
- The amount stays in the item's text ("2 l Milch"): one field, no separate
  amount field. The page offers − and + for a leading amount, in steps that suit its unit: grams and millilitres by 10 below 100 and by 50 from there, kilograms and litres by ½, counted things by 1.
- Filling from a recipe is a server page (`/recipes/{id}/shop`): a preview
  with every ingredient ticked, then one change to the list. The pure package
  `internal/shopping` plans it and adds up amounts of the same ingredient in
  compatible units, never into an item already ticked.
- An item not got moves to another shopping list with the operation `move`
  (`list`, `item`, `to`, `lang`): one operation, so offline it neither doubles
  nor disappears; the server takes the item (with note and `for`) off the one
  list as a deletion and adds it unticked to the other, adding it up with a
  like item there as filling does (2026-11-11). Only between shopping lists.

## Alternatives Considered

- **A second, separate list system for shopping:** its own offline storage,
  sync and sharing — a second copy of ADR-006 to keep right.
- **Separate amount and unit fields per item:** exact, but the slow way other
  shopping apps take: create the item, then find the amount field.
- **One file per item** (as first planned): fewer write conflicts, but the
  whole sync and order logic of ADR-006 would have to be built again.

## Consequences

- Shopping lists appear under "Todos" next to plain lists, with their own
  icon, item view and "Neue Einkaufsliste".
- Plain to-do lists are unchanged; the new fields are omitted when empty.
- Adding up relies on the recipe parser understanding the item text; an item
  it cannot read is never changed, a new one is added instead.
- `for` prepares the cooking plan (part 3), which will take amounts off again.
- Adding up keeps what the person wrote after the amount ("200 g Mehl, Typ
  405" + 500 g → "700 g Mehl, Typ 405"); an item without an amount takes the
  recipe's amount.
- Each fill carries an id; sending it twice (double tap, Back) adds nothing.
  It is planned again under the list's lock, against the list as it is then.
- On a shopping list a tick older than the item's last text change is
  ignored: the amount may have grown since, so it was not bought. A tick made
  after the change counts as usual.
