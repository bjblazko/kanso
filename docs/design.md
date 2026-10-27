# Design

**Created:** 2026-10-26
**Updated:** 2026-10-26

Kanso follows `huepattl-rams-design`. Tokens are copied verbatim into
`web/static/css/tokens.css`; components use only `var(--…)`.

## Screens and their one job

| Screen | Primary job |
|---|---|
| Start | Get into a space. |
| Page view | Read a page. |
| Page editor | Change a page and save it. |
| Space settings | Decide who may read and write. |
| Licenses and thanks | Show whose work Kanso contains. |
| Note capture | Write something down, now. The field fills the screen, buttons sit below it; "Speichern" is the only accent. |
| Notes list | Find a note to sort. |
| Note → wiki page | Choose where the note goes. |
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

- **Icons:** Lucide is not used (every dependency needs approval). Kanso
  draws its few icons itself, in the same style: line icons, 1.5 px stroke at
  24 px, round caps and joins, `currentColor`.
