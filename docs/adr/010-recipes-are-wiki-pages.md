# 010 — Recipes Are Wiki Pages with a Convention

**Status:** Accepted
**Created:** 2026-11-06
**Updated:** 2026-11-17

## Context

Recipes live in Instagram videos and on web pages; cooking from a phone means
scrubbing back and forth, and the shopping list is copied by hand. Kanso
should keep recipes, find them by search and tags, show them comfortably at
the stove, and later fill a shopping list from them with amounts added up.
For that it must understand amounts, units and ingredient names, but typing a
recipe should stay as easy as writing a page, and the file should stay
readable without Kanso.

## Decision

A recipe is an ordinary wiki page with `type: recipe` in its frontmatter,
plus optional `servings`, `tags`, `source` and `time` (later also `image`,
`cooked` and `rating`, 1–5 stars shared by everyone who reads it). Its text follows a
Markdown convention: a `## Zutaten` (or `Ingredients`) list of
`[amount] [unit] name[, note]` lines, optional `###` groups, and a
`## Zubereitung` (or `Schritte`, `Steps`, `Method`) list of steps. The pure
package `internal/recipes` reads this. A line it does not understand stays
word for word and counts as an ingredient without an amount; saving is never
blocked. Units come from a fixed list in families (mass, volume, count);
only units of one family are added up.

## Alternatives Considered

- **A recipe type of its own with a form per ingredient:** exact data, but
  slow to type, a second store next to the wiki, and not plain Markdown.
- **Free pages without structure:** easiest to write, but nothing to scale or
  add up; a shopping list would need guessing or an AI.

## Consequences

- Recipes get everything pages have: history, sharing, attachments, search,
  moving.
- Recipes are shown where they are used, not twice (2026-11-17): the wiki's
  spaces bar and the subpage lists leave them out (a plain page below a
  recipe moves up in its place), and a recipe page — view, editor, history,
  sharing — belongs to Küche in the rail, its path leads back to "Rezepte".
  The file stays a page in its space; search still finds it.
- One space is the cookbook that holds the recipes ([ADR-016](016-one-cookbook-space.md)).
- Recognition can fail on unusual lines; the editor preview shows how Kanso
  read each ingredient so the author can fix it.
- An optional AI (OpenAI-compatible endpoint) may later turn pasted text
  into this format; Kanso does not need it.
