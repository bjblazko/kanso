# 012 — Meal Plans Belong to Groups and Live in Git

**Status:** Accepted
**Created:** 2026-11-07
**Updated:** 2026-11-07

## Context

A household plans next week's meals on one day and shops on the next. Meals
are recipes (ADR-010) assigned to days; chosen days fill a shopping list
(ADR-011). The owner chose one shared plan per group, made by an admin, like
group boards (ADR-008).

## Decision

- `internal/mealplan` keeps plans in the Git history: `mealplans/<id>/plan.yaml`
  (name, group) and one file per ISO week, `mealplans/<id>/<YYYY-Www>.yaml`,
  with the entries of each day. Each change is one commit.
- Members of the plan's group see and change it; admins create and delete
  plans for a group but do not see them unless they are members.
- An entry names the recipe by page id and keeps its title for display. When
  its ingredients went on a shopping list, the entry stores the fill id and
  the list; taking the meal off the plan can then take exactly its share off
  that list again (`shopping.Remove`), never touching items already ticked.
- The plan is a server page and needs the network.

## Alternatives Considered

- **Calendar events (iCalendar):** the calendar does not exist yet, and a
  meal is not a time slot; a calendar layer can show plans later.
- **One plan per person, shared like lists:** more sharing logic for a
  household that plans together anyway.
- **Outside Git like to-do lists:** no offline need here, and the history of
  what was planned is worth keeping.

## Consequences

- Planning and shopping from a plan need the network; ticking the list in the
  shop still works offline.
- A recipe a member cannot read shows in the plan by title only and is
  skipped when shopping.
- Taking a meal off deletes only items its fill made (`new` on the source);
  an item a person typed keeps its words and its own amount. When the list
  is one the person cannot open, the page says so instead of doing less
  silently; a failed subtraction keeps the meal in the plan.
- The shopping preview names its meals by entry id, so a meal added or
  removed meanwhile by someone else shows the preview again instead of
  shifting ticks; a meal counts as shopped only when something of it
  landed on a list that still exists.
