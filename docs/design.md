# Design

**Created:** 2026-10-26
**Updated:** 2026-10-30

Kanso follows `huepattl-rams-design`. Tokens are copied verbatim into
`web/static/css/tokens.css`; components use only `var(--…)`.

## Screens and their one job

| Screen | Primary job |
|---|---|
| Rail (all screens) | Switch between the parts of Kanso; bottom bar on phones, left column on wide screens (ADR-005). |
| Start (wiki) | Get into a wiki space. |
| Page view | Read a page. |
| Page editor | Change a page and save it. |
| Space settings | Decide who may read and write. |
| Licenses and thanks | Show whose work Kanso contains. |
| Note capture | Write something down, now. The field fills the screen, buttons sit below it; "Speichern" is the only accent. |
| Notes list | Find a note to sort. |
| Note → wiki page | Choose where the note goes. |
| Board | See where each task stands and pick the next one. Tabs Kanban and Matrix are links; "Neue Aufgabe" is the only accent. Due words: overdue red, today/tomorrow yellow, always as a word. |
| Task | Read a task and act on it: Bearbeiten is the accent; Übernehmen/Freigeben (group boards only) secondary; delete last. |
| Task editor | Change a task. Column as radio buttons; "Worauf wartest du?" appears only while Wartet is chosen (CSS `:has`, value kept); text with source/preview like the wiki editor, no attachments. |
| Task capture | Write a task down, now: subject (focused), board, urgent/important, due as one-tap choices; "Speichern" is the only accent. Choices wrap on narrow phones instead of a segment row. |
| App icon | Three note lines and one accent dot on the dark ink colour; mark inside the central 60 % (maskable safe zone). |

## Layout

Mobile first: top bar (name, search, user menu), navigation below it as a
collapsible block; from 60rem a fixed left navigation. Navigation entries are
links with `aria-current`, actions are framed buttons.

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
