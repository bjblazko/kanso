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
| Page view | Read a page. A flat toolbar above the title stays in view while scrolling: Bearbeiten (the only accent), Kochen, Einkaufen and Einplanen (recipes only, also for readers), Unterseite, then Verschieben, Verlauf, Löschen (red text); right, who sees the page as a button to its sharing (or plain text for readers). On phones the quiet actions show only their icon, the word stays for screen readers. |
| Recipe page | A page view; above the meta line a quiet line with servings, time, source (a link only for web addresses) and the tags as small framed links into Küche; below it the rating: one framed group of five star buttons (filled = given, in the text colour, not the accent; `aria-pressed` on the given one; pressing it again clears) with "4 von 5 Sternen" beside it — readers see only the stars, nothing when not rated; below the text "Gekocht": "8× gekocht, zuletzt am …", then the days — from the meal plans of one's groups (a link into that week) and noted with "Heute gekocht" ("von Hand") — latest first, future days marked "geplant". Other pages show their tags as links into the search. |
| Search | Find anything one may read, with filters: "Nur Rezepte", the minimum stars ("ab 3/4 Sternen", "5 Sterne", once a recipe is among the hits) and the tags of the hits as framed links that switch on and off (`aria-pressed`, state in the URL); a tag alone (no words) lists everything with it. Hits in groups: Rezepte (56 px photo, stars, tags, "Alle in der Küche zeigen"), Seiten (space and tags), Notizen (own only). The kitchen uses the same search for its words. |
| Kitchen | Find a recipe. Toolbar: "Neues Rezept" (the only accent), "Von Webseite", then the tabs Rezepte · Kochplan; a search field (title, ingredient, tag) and the minimum stars (once any recipe is rated) and the tags of all visible recipes as framed links that switch on and off (`aria-pressed`, state in the URL); rows with a 56 px square photo (the recipe's `image`, else its first attached photo, else an empty square), title, stars (14 px, read-only), tags, time, space. |
| New recipe | Name a recipe and pick its space (only spaces the person may write); opens the editor on a template. |
| Shop for a recipe | Decide what of a recipe goes on which shopping list. Servings and list on top — an existing shopping list or "Neue Liste „Einkauf: <Rezept>“": the name is derived, never asked for; changes refresh the preview (a button without script); below every ingredient ticked with its amount, "200 g → 1 kg" and "steht schon drauf" where it adds to an item; "Auf die Liste" is the only accent. Reached by "Einkaufen" on the recipe page and in the cook view (with its servings). |
| Shopping list | A list under Todos with a cart icon ("Einkaufsliste anlegen" next to "Liste anlegen"). Each item: tick, text with the amount, below it the note and "für <recipe>" (links); − and + (quiet, at least 44 px wide) when the text starts with an amount: grams and millilitres by 10 below 100 and by 50 from there (snapping to the step: 87 g → 90 g), kilograms and litres by ½, anything else by 1; − disabled at the smallest step. Between them the amount itself as a quiet button: tapping it opens a small number field (decimal keyboard) for an exact amount; Enter or leaving the field keeps it, Escape does not. Tapping the text opens text and note with Speichern and Punkt löschen; no sub-items. The add field asks "Was fehlt? z. B. 2 l Milch". |
| Meal plan | See and change what the group cooks this week. Toolbar: "Einkaufen" (the only accent, sends the ticked days), then the kitchen tabs Rezepte · Kochplan. Below "‹ Vorige · KW 46 · 9.–15.11. · Nächste ›", then Monday to Sunday: a day's checkbox (only with meals), its name and date (today marked), "+ Gericht"; under it the meals with a 40 px photo (as in the kitchen), servings, "eingekauft" once shopped, and a quiet bin to take it off. A recipe the person cannot read shows its title without a link. |
| Shop for the week | The ticked days' meals as sections ("Di · Curry · 4 Portionen"), each with its lines ticked; later meals already count the earlier ones ("800 g → 1,6 kg"). List choice and refresh as for one recipe, a new list is "Einkauf KW 46 (9.–15.11.)"; without ticked days every day with meals counts, an empty week says so; meals already shopped and recipes the person cannot read are named below, not offered. "Auf die Liste" is the only accent. |
| Plan a meal | Pick day, recipe (or plan, when coming from a recipe) and servings (empty = the recipe's). |
| Take a meal off | Plain confirmation; when its ingredients are on a list, "Entfernen und Zutaten abziehen" is the accent and "Nur aus dem Plan nehmen" the alternative. |
| AI connection | An admin registers one AI: on/off, base URL, model, token (password field, never shown again; "Token ist gesetzt" instead), remove token. A plain note says who processes sent text. "Speichern" is the accent; "Speichern und Verbindung testen" shows the outcome in words. |
| Recipe from a web page | URL and space; a line says where text goes (the AI's host) or that only pages with recipe data work. The button says the wait can take up to 3 minutes and cannot be pressed twice. |
| Imported draft | Title, servings, time, "from the page's recipe data" or "with AI", the page's photos as squares, each with "Wohin?" (Nicht übernehmen · Oben · Nach Schritt n · Am Ende; the dish on top and step photos below their step preset, others not taken), how Kanso reads the ingredients, the text to check; only "Rezept anlegen" (accent) creates the page. |
| Cook view | Cook from the phone at the stove. Toolbar: "Zur Seite", "Heute gekocht" (for those who may change the recipe; afterwards "Heute gekocht ✓" as text), "Häkchen entfernen" (shown only with script) and the servings stepper (− number + as links, so it works without script); no accent. Below only ingredients (by group) and steps, large, each to tick off, a step's photo right below it; ticks are kept on the device, the screen stays on (Wake Lock) while visible. |
| Page editor | Change a page and save it. Tags (free, comma-separated, created on save) for every page, then a box "Ist ein Rezept"; while ticked, Portionen, Zeit and Quelle appear (CSS `:has`, values kept) and the preview ends with "So versteht Kanso die Zutaten" (amount · unit · name; lines without an amount muted). |
| Space settings | Decide who may read and write. |
| About Kanso | Say what Kanso is and whose work it contains: name and version, copyright, licence, links to GitHub and huepattl.de, the disclaimer, "Was Kanso wohin schickt", then licenses and thanks. Reached from the footer line "Hüpattl! Kanso 0.1.0 · Über Kanso" on every page; `/about/licenses` shows the same page. |
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

## Phones (owner's devices: Motorola Edge 60 Neo, iPhone 16 Pro Max)

- No `viewport-fit=cover` and no safe-area padding: Chrome on Android reports
  a bottom inset that comes and goes with its toolbar, so the bottom bar jumped
  while scrolling. The browser keeps the page above the system bar instead.
- The top bar shows only the profile picture on phones; the search gets the room.
- Toolbars stay one row: rare page actions (Verschieben, Verlauf, Löschen)
  sit behind "⋯" on phones and in the row on wide screens.
- Buttons that belong together share a row in equal parts (Kamera · Foto
  wählen; Liste anlegen · Einkaufsliste anlegen); a lone primary action
  takes the full width below them.
- Shopping items are a grid: tick, text and − amount + in one row, note and
  "für …" below across the width.
- Ticks meant for a thumb are 1.25 rem.

## AI mark (ADR-013)

Functions that send text to the AI are marked with a word, not a colour (no
rainbow or gradient buttons, no second accent): a frameless badge on the
surface step, `[↗ KI · extern: openrouter.ai]` or `[↗ KI · im eigenen Netz:
ollama.intranet…]`, inside the sentence that says what happens — never framed
like a button, never in the row of actions. Works in greyscale; the host is the
honest part.

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
