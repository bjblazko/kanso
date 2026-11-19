# Cook progress and amounts on every shopping item — design

**Status:** approved 2026-11-19 (short design, small change)

## Intent

Cooking and shopping should feel like the to-do lists: you see how far you
are, and you change "how many" with − and + instead of retyping.

## Goals

1. **Cook view:** a progress bar with percentage over the steps, the same as
   a to-do list's. It counts ticked steps only (not ingredients) and moves at
   once when a step is ticked or the ticks are reset; after a reload it shows
   the ticks remembered on this device.
2. **Shopping list amounts:**
   - An item without an amount ("Bier Alkoholfrei") counts as 1 and gets
     − 1 +; + writes "2 Bier Alkoholfrei". − is disabled at 1.
   - "3x Wasser" and "3 x Wasser" are recognised as a factor of 3.
   - The amount shows only at the stepper; the item's text shows without it
     ("Wasser  − 3 +", "Butter  − 250 g +"). The stored text keeps the amount
     ("3x Wasser"), so the data format, sync and merging stay as they are.
   - An item that starts with a number that is not an amount ("7Up",
     "2 - 3 Eier") stays as written and gets no stepper.

## Non-goals

Offline for the cook view or a different PWA start page (asked separately).
No server or data format change.

## Approaches

- **Display only, stored text unchanged (chosen):** the client parses the
  leading amount as today and only draws it differently. No migration, the
  server's merging of recipe amounts keeps working.
- Amount as its own field on the item: cleaner data, but a format change in
  `internal/todos`, sync and every list file — far more than asked.

## Left to the builder

The cook bar reuses `.todo-meter`/`.todo-percent`; the percent text comes
from `todos.percent`. The checkbox keeps the full text as its accessible name.
