# Generate a recipe with the AI — design

**Status:** approved 2026-11-17 (short design)

## Why

The owner has ingredients they fancy or that are in the fridge, plus a few
wishes (warm or cold, spicy, hearty home cooking, no vegetables …), and does not
want to search for a fitting recipe. The AI should make one dish from that.
The AI must only write recipes: no everyday questions, no chat, and no way to
talk it into something else through the form (prompt injection).

## Goals, non-goals

- Goal: one recipe from ingredients and wishes. "Neu würfeln" throws the
  suggestion away and generates a new one from the same wishes, as often as
  wanted, until the person accepts it. Nothing is stored before that.
- Goal: its own function with its own button in the kitchen toolbar, separate
  from "Von Webseite".
- Non-goal: several suggestions to choose from, a conversation, follow-up
  questions from the AI, memory between requests.
- Later, a separate feature: find recipes that contain all chosen ingredients
  (AND).
- Anti-goal: anything the AI answers that is not a recipe reaching the person.

## Approaches

- **A (chosen): shaped like the recipe import.** Form → POST → preview page
  that carries the wishes and the rejected titles as hidden fields → accept
  through the existing import create path. No server state.
- B: keep the wishes and drafts on the server: state that needs expiry and
  cleanup, without any benefit.
- C: a chat-like flow: exactly the open conversation surface that is not wanted.

## Design

- **Toolbar:** a new quiet button "Rezept generieren" next to "Von Webseite",
  shown only when an AI connection is on.
- **Form** `GET /recipes/generate`:
  - Ingredients: free text, comma or line separated, up to 15, each up to
    40 characters.
  - "Nur diese + Grundvorrat" or "darf ergänzen".
  - Warm or kalt or egal; mild or scharf or egal.
  - Direction: a fixed list (egal, gutbürgerlich, italienisch, asiatisch,
    mediterran, orientalisch, mexikanisch …).
  - Without: Fleisch, Fisch, Gemüse, Gluten, Laktose.
  - Time: egal, bis 20, 45 or 90 min.
  - Portionen.
  - "Sonst noch": free text, up to 200 characters.
  - The page shows the AI mark (ADR-013) and the wait hint.
- **Security:**
  - Choice fields are checked against fixed lists in Go; an unknown value is
    refused (400).
  - The free text (ingredients, "Sonst noch", rejected titles) has control
    characters and markup characters removed, is cut to its limit, and is sent
    only in the user message inside clearly marked blocks as data.
  - The system prompt allows only cooking recipes. It treats everything in the
    blocks as a wish about the dish, never as an instruction. Anything else
    gets exactly `KEIN REZEPT`.
  - The answer must pass the shared recipe parser (title plus ingredients),
    otherwise it is discarded with an error message.
  - Each request stands alone: no tools, no Kanso content, no history beyond
    the rejected titles.
  - Kanso logs no text sent or received.
  - Only someone who may write to the cookbook may generate, as with the
    import.
- **Preview** (not stored): the recipe, a short "KI-generiert, nicht
  ausprobiert" hint, and three buttons:
  - Übernehmen: create through the import path, with the tag `ki-generiert`
    and no source.
  - Neu würfeln: the same wishes plus this title on the rejected list (at most
    10 kept).
  - Wünsche ändern: the form, filled in again.
- **Language:** the person's Kanso language. The headings stay the parser's
  `## Zutaten` and `## Zubereitung`, which Kanso translates.
- **Code:**
  - A new package `internal/recipegen` holds the wishes type, validation, the
    prompt (`prompt.md`, embedded) and `Generate(ctx, client, cfg, wishes,
    lang) (Draft, error)`.
  - The answer parser moves out of `recipeimport` so that both use it.
  - Tests cover injection text ending up inside the data block, the refused
    choice values, and a non-recipe answer being discarded.

## Left to the builder

The exact list of directions, the field layout per `huepattl-rams-design`, and
the i18n wording (de/en).

Records: ADR-017 (generation and its injection guards), feature doc,
CHANGELOG, arc42.
