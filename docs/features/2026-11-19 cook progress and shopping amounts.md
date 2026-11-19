---
status: done
created: 2026-11-19
updated: 2026-11-19
---

# Cook progress and amounts on every shopping item

The cook view shows a progress bar with percentage over the steps, the same as
a to-do list's. It counts ticked steps (not ingredients), moves when a step is
ticked or the ticks are reset, and keeps the ticks remembered on the device.

On a shopping list the amount sits at − and + and the item shows without it:
"250 g Butter" reads "Butter − 250 g +". "3x Wasser" and "3 x Wasser" are a
factor of 3. An item without an amount counts as 1 and gets − and + too;
+ makes "Bier" into "2 Bier". An item that starts with a number that is no
amount ("7Up", "2 - 3 Eier") stays as written, without − and +. The stored
text keeps the amount, so sync, moving and adding up on the server are
unchanged.

## Related

- Design: `docs/superpowers/specs/2026-11-19-cook-progress-and-amounts-design.md`
- ADR-011 (shopping lists)
