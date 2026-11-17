# 016 — One Cookbook Space Holds the Recipes

**Status:** Accepted
**Created:** 2026-11-17
**Updated:** 2026-11-17

## Context

Recipes are wiki pages (ADR-010) and since 0.8.0 appear only in the kitchen.
The space that held them stayed in the wiki, empty, and "Neues Rezept" still
asked for a space. There should be one place for recipes, filtered by tags,
and one setting that says who sees all of them — as the members of that space
did already.

## Decision

- Exactly one space is the **cookbook**: `cookbook: true` in its `space.yaml`;
  `spaces.Service.SetCookbook` keeps at most one.
- At start, if none is marked and exactly one space holds recipes and nothing
  else, Kanso marks it (one commit, author "Kanso").
- Without a cookbook, admins create it from the kitchen (space "Rezepte", the
  admin as owner); admins can move the mark under "Alle Bereiche".
- The cookbook is not in the wiki (spaces bar, home page); its space page
  redirects to the kitchen; its settings stay where they are and the kitchen
  toolbar says who sees it, with a link there for owners and admins.
- "Neues Rezept" and the import create in the cookbook; the server ignores a
  posted space.

## Alternatives Considered

- **Several cookbooks**, the choice shown only when there are more than one:
  allows private recipes, but nobody needs that now.
- **Hide spaces that hold only recipes, remember the last space**: no new
  concept, but implicit — one ordinary page would bring the space back.
- **Recipes owned by groups like meal plans**: a rewrite of ADR-010 with a
  data migration for no gain.

## Consequences

- One setting decides who sees and writes recipes; page shares still add
  people to single recipes (ADR-009) but cannot hide one.
- A recipe only one person sees is not possible in a shared cookbook.
- Recipes created before in other spaces stay there and still show in the
  kitchen.
