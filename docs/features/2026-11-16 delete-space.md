---
status: done
created: 2026-11-16
updated: 2026-11-16
---

# Delete a space

Whoever may manage a space finds "Bereich löschen" as the last section of its
settings. The confirmation says what is lost — pages, how many of them are
recipes, attachments, meal plan entries that cook its recipes and shared
pages — and that the content stays in the Git history. A space with pages
asks for its name to be typed; an empty one only for the button. While
someone edits a page of the space, it is not deleted and the confirmation
says who is editing.

Deleting removes `spaces/<slug>/` in one commit ("Bereich gelöscht: <Name>")
and drops its pages from the wiki index and the search while the repository
is locked. Meal plans show a deleted recipe's title without a link, as for
recipes one may not read. The start page says "Bereich „<Name>“ gelöscht."

## Related

- [Design](../superpowers/specs/2026-11-16-delete-space-design.md)
