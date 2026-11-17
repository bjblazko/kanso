---
status: done
created: 2026-11-17
updated: 2026-11-17
---

# Rezept generieren

In the kitchen, "Rezept generieren" turns ingredients one fancies or has in
the fridge into one recipe made up by the AI. Wishes can be added: warm or
cold, mild or spicy, a direction (home-style, Italian, Asian …), what must not
go in (meat, fish, vegetables, gluten, lactose), time, servings and a short
"Sonst noch".

The suggestion is not stored. "Neu würfeln" throws it away and asks again with
the same wishes (and "something else than these"), "Wünsche ändern" goes back
to the filled form, and "Übernehmen" creates it in the cookbook, tagged
`ki-generiert`. The button shows only when an AI connection is on; only
people who may write to the cookbook can use it.

The AI writes nothing but recipes. Choices come from fixed lists, free text is
cleaned, capped and sent as marked data, and an answer that is not a recipe
never reaches the person.

## Related

- [ADR-017](../adr/017-generated-recipes-guard-the-prompt.md) — the guards
- [ADR-013](../adr/013-ai-openai-compatible-through-kanso-services.md) — the AI connection
- [ADR-016](../adr/016-one-cookbook-space.md) — the cookbook
- Spec: `docs/superpowers/specs/2026-11-17-recipe-generate-design.md`
