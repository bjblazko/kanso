# One cookbook for all recipes — design

**Status:** approved 2026-11-17 (short design)

## Why

Since 0.8.0 recipes appear only in the kitchen. The space that holds them
still shows up empty in the wiki, and "Neues Rezept" still asks for a space,
though there should be one place for recipes. Sharing should stay one setting
for all recipes, as it is today through the space's members.

## Approaches

- **A (chosen): exactly one cookbook space**, marked in its `space.yaml`,
  hidden from the wiki, recipes always created there, its sharing reached from
  the kitchen.
- B: several cookbooks (a private one possible) — more than needed now.
- C: hide spaces that hold only recipes and remember the last space — implicit;
  one ordinary page would bring the space back.

## Design

- `spaces.Space.Cookbook` (`cookbook: true`); `Service.Cookbook()`,
  `Service.SetCookbook(author, slug)` keeps at most one.
- At start: no cookbook yet and all recipes in exactly one space → that space
  is marked (one commit).
- No cookbook: admins get "Kochbuch anlegen" in the kitchen (space "Rezepte",
  the admin as owner, marked); others read that an admin creates it. Admins can
  move the mark under "Alle Bereiche".
- Wiki: the cookbook is not in the spaces bar or on the home page;
  `/s/<cookbook>` redirects to `/recipes`; its settings stay reachable.
- Kitchen: "Neues Rezept" and "Von Webseite" create in the cookbook without
  asking (the server ignores a posted space); without write access a message
  says so. The kitchen toolbar says who sees the cookbook, with the link to its
  settings for owners and admins.
- Unchanged: recipes elsewhere still show in the kitchen; page shares, search,
  history, reports.

Records: ADR-016, ADR-010 updated.
