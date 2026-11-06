# Design

**Created:** 2026-10-26
**Updated:** 2026-11-06

Kanso follows `huepattl-rams-design`. Tokens are copied verbatim into
`web/static/css/tokens.css`; components use only `var(--…)`.

## Screens and their one job

| Screen | Primary job |
|---|---|
| Rail (all screens) | Switch between the parts of Kanso (Notizen, Todos, Aufgaben, Wiki, Küche); bottom bar on phones, left column on wide screens (ADR-005). |
| Start (wiki) | Get into a wiki space. |
| Page view | Read a page. A flat toolbar above the title stays in view while scrolling: Bearbeiten (the only accent), Kochen and Einkaufen (recipes only, also for readers), Unterseite, then Verschieben, Verlauf, Löschen (red text); right, who sees the page as a button to its sharing (or plain text for readers). On phones the quiet actions show only their icon, the word stays for screen readers. |
| Recipe page | A page view; above the meta line a quiet line with servings, time, source (a link only for web addresses) and the tags as small framed links into Küche. |
| Kitchen | Find a recipe. Toolbar: "Neues Rezept" (the only accent); a search field (title, ingredient, tag) and the tags of all visible recipes as framed links that switch on and off (`aria-pressed`, state in the URL); rows with title, tags, time, space. |
| New recipe | Name a recipe and pick its space (only spaces the person may write); opens the editor on a template. |
| Shop for a recipe | Decide what of a recipe goes on which shopping list. Servings and list (an existing shopping list or a new one with a name) on top, changes refresh the preview (a button without script); below every ingredient ticked with its amount, "200 g → 1 kg" and "steht schon drauf" where it adds to an item; "Auf die Liste" is the only accent. Reached by "Einkaufen" on the recipe page and in the cook view (with its servings). |
| Shopping list | A list under Todos with a cart icon ("Einkaufsliste anlegen" next to "Liste anlegen"). Each item: tick, text with the amount, below it the note and "für <recipe>" (links); − and + (quiet, small) when the text starts with a number, − disabled at 1. Tapping the text opens text and note with Speichern and Punkt löschen; no sub-items. The add field asks "Was fehlt? z. B. 2 l Milch". |
| Cook view | Cook from the phone at the stove. Toolbar: "Zur Seite", "Häkchen entfernen" (shown only with script) and the servings stepper (− number + as links, so it works without script); no accent. Below only ingredients (by group) and steps, large, each to tick off; ticks are kept on the device, the screen stays on (Wake Lock) while visible. |
| Page editor | Change a page and save it. A box "Ist ein Rezept" below the title; while ticked, Portionen, Zeit, Tags and Quelle appear (CSS `:has`, values kept) and the preview ends with "So versteht Kanso die Zutaten" (amount · unit · name; lines without an amount muted). |
| Space settings | Decide who may read and write. |
| Licenses and thanks | Show whose work Kanso contains. |
| Note capture | Write something down, now. The field fills the screen, buttons sit below it; "Speichern" is the only accent. |
| Notes list | Find a note to sort. |
| Note → wiki page | Choose where the note goes. |
| Board | See where each task stands and pick the next one. Tabs Kanban and Matrix are links; "Neue Aufgabe" is the only accent. Due words: overdue red, today/tomorrow yellow, always as a word. |
| Task | Read a task and act on it: Bearbeiten is the accent; Übernehmen/Freigeben (group boards only) secondary; delete last. |
| Task editor | Change a task. Column as radio buttons; "Worauf wartest du?" appears only while Wartet is chosen (CSS `:has`, value kept); text with source/preview like the wiki editor, no attachments. |
| Task capture | Write a task down, now: subject (focused), board, urgent/important, due as one-tap choices; "Speichern" is the only accent. Choices wrap on narrow phones instead of a segment row. |
| Page sharing | Decide who else may read or change this one page: list of shares, add form, hint "Gilt auch für N Unterseiten", inherited shares and the space's readers below. |
| Move confirmation | Decide whether a page may go where other people read it: "Bisher" and "Danach" readers side by side, "Verschieben" as the accent, "Abbrechen" beside it. |
| App icon | Three note lines and one accent dot on the dark ink colour; mark inside the central 60 % (maskable safe zone). |

## Layout

Mobile first: top bar (name, search, user menu), navigation below it as a
collapsible block; from 60rem a fixed left navigation. Navigation entries are
links with `aria-current`, actions are framed buttons.

## Screen head (owner's rule, 2026-11-01)

Every screen that shows content (wiki start, space, page, notes, note, to-do
list, board, task) opens with a **screen head** that stays at the top while
the content scrolls: a flat **toolbar**, then the path (breadcrumb) if there
is one, then the **title**. No action buttons below the content.

- Toolbar order: the primary action (the screen's only accent), further
  actions as quiet framed buttons with icon and word, a hairline, then rare or
  destructive actions (delete in red text), and at the right end **who sees
  it** (ADR-007) — the button to the sharing for whoever may change it, plain
  text for everyone else.
- On phones the quiet actions show only their icon; the word stays for screen
  readers. "Who sees it" takes its own row.
- `scroll-padding-top` keeps focused or scrolled-to elements below the head.
- Form pages (editors, settings, capture) keep their submit button at the end
  of the form.

## Choices and actions

Selects are fields, never buttons: the input frame (`--radius-sm`, field
background) plus a chevron drawn from `--fg-2`, so a choice never looks like
an action next to an "Add" button. Buttons keep their own frame and weight.

The line that says who can see something (ADR-007) ends with "Freigabe
ändern" for whoever may change it: space owners (space settings), the owner of
a to-do list (share page), admins on group boards (groups). Notes and private
boards have nothing to share.

## Contrast (checked with contrast.py)

| Pair | Ratio |
|---|---|
| `--fg` on `--bg` (light) | 14.96:1 |
| `--fg-3` on `--bg-2` (light, navigation meta) | 4.53:1 |
| `--fg-3` on `--bg` (dark) | 6.64:1 |

## Deviations from huepattl-rams-design

- **To-do items:** an item's text is a frameless button that opens the item
  for editing (as in reminder apps). It is the item itself, not a separate
  action; a framed edit button in every row would double the list's noise.
  Ticked items fade to 0.62 opacity, which keeps their text at 4.5:1.

- **Account entry:** the picture (or a round placeholder) and the name open
  the account menu, without a frame, as on every platform; it gets the hover
  overlay and the focus ring. Inside, the language is a segmented control of
  two submit buttons, so no "Apply" is needed.

- **Icons:** Lucide is not used (every dependency needs approval). Kanso
  draws its few icons itself, in the same style: line icons, 1.5 px stroke at
  24 px, round caps and joins, `currentColor`.
