# Backlog

**Created:** 2026-11-07
**Updated:** 2026-11-07

Known work that is not planned yet: technical debt, follow-ups from reviews,
ideas. Risks that stay are in [arc42 chapter 11](architecture/arc42.md#11-risks-and-technical-debt);
decisions in [ADRs](adr/). Done items are removed here and noted in the CHANGELOG.

## Technical debt

1. **Flaky browser tests with offline lists.** In a full `just e2e` run one
   to-do or shopping-list test fails now and then (a different one each
   time, e.g. "a list with items and a sub-item", "progress: a bar …",
   "shopping list: amounts …"); alone they pass. Find the race (device copy,
   outbox, sync, redraw while a locator waits) and make the tests wait for a
   settled state instead of timing. Seen 2026-11-07.
2. **Introduce aim42** for architecture improvement: collect issues and
   improvements with cost and benefit, linked from arc42 chapter 11.
3. CLAUDE.md names Go end-to-end tests in `internal/e2e`; they live in
   `internal/web` (`*_test.go` against the whole server). Align the text or
   move them.

## Follow-ups from reviews

- Shopping: sums round to the nearest step, not up (250 g + 12 g → 260 g);
  the unit word of the first amount is kept ("3 Dose Tomaten"); `Apply`
  ignores the item and text limits for recipe lines.
- Shopping: a recipe title in "für …" is visible to list members who cannot
  read that recipe.
- Recipes: unticking "Ist ein Rezept" clears servings, time and source (the
  spec said only the type); units are not inflected ("2 Dose"); the parser
  misses "1. 200 g Mehl" in ingredient lists, "3-Korn-Mischung", Msp.,
  Päckchen, Würfel, oz; the kitchen search needs every word to match.
- Recipes: `/recipes` parses every page per request — fine for a household,
  cache by page update time if it grows.
- Meal plan: a deleted recipe is reported as "no read right" when shopping;
  "eingekauft" stays in the week view when its list was deleted.
- Import: windows-1252 bytes 0x80–0x9F are read as Latin-1 (€, quotes); no
  tests yet for the 3 MB page limit, the redirect cap and model errors with
  HTTP 400.

## Ideas

- Tag filter in the wiki search (tags exist on every page since 0.1.0).
- Import recipes from Instagram (caption pasted or a screenshot).
- AI help in the wiki (tidy, summarise, find) as read-only tools (ADR-013).
- Rename meal plans; a confirmation before sending text to an external AI.
