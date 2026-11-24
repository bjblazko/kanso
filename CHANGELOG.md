# Changelog

All notable changes to this project are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/spec/v2.0.0.html). The version lives in
`internal/buildinfo/VERSION`; Kanso shows it in the footer and under "Über Kanso".

## [Unreleased]

## [0.15.0] - 2026-11-24

### Added

- Selfcare › Klänge: white, pink or brown noise, waves, and a calm floating chord with quiet chimes, all made in the browser with no sound files. Endless, or with a sleep timer (15–90 minutes) that fades out in the last minute; volume per layer; an optional "Bildschirm anlassen". The choices stay on the device, and the reminder keeps quiet while it plays.

## [0.14.0] - 2026-11-24

### Added

- Selfcare starts with four ways in — Zwischendurch, Kurz durchatmen, Wie geht's mir?, Zeit für mich — and an explicit "Mir geht es gerade gar nicht gut", which puts help from outside first.
- Zwischendurch: a short step between two things. Close what is done, do something small and lifting for yourself, then arrive at what comes next (with "Fokus-Runde starten"); the answers can be saved as a note.
- A reminder: after 30/60/90 minutes with Kanso open (60 by default, off possible, per device), a quiet bar suggests Zwischendurch, with a notification if the tab is in the background. It waits during a Fokus round or a meditation.
- The battery check jumps to its suggestions and offers a matching exercise or meditation.
- Own first-aid ideas per person (calming, lifting or both), mixed into the random steps; private and outside the history.
- Abendrückblick: three questions at the end of the day, as a note on request.
- Time bars: Fokus, meditations and breathing show their progress as a slim bar under the time (yellow while time passes, neutral when paused; while breathing it fills with the in-breath and empties with the out-breath). Start buttons show a stop or pause icon while running.
- Fokus: a soft gong at the end of a phase, a lifting Selfcare step in each break, and a "Gedanken-Parkplatz" for what comes up during a round, saved as a note on request.

- Selfcare: two guided meditations, „Ankommen“ (5 minutes) and „Körperreise“ (10 minutes), spoken by Lyra or Orion in the UI language, with a gong and the text to read along. They are small clips per sentence, which the page plays with the pauses (ADR-023).

## [0.13.0] - 2026-11-24

### Added

- Menu settings: each person chooses which parts of Kanso the menu shows and in which order, on each device ("Menü anpassen"; ADR-020). On phones more than five parts become four and "Mehr", and on wide screens the menu scrolls.
- Selfcare: first aid for stressful moments — a random small step, a battery check with steps that fit, guided exercises (breathing with counting, senses 5-4-3-2-1, check-in, …) and help from outside on every page. Nothing is stored unless saved as a private note (ADR-022).
- Fokus, a Pomodoro timer: 25/5 minutes and a long break after four rounds. The durations can be changed, and the end of a phase brings a tone and, if allowed, a notification from the open tab (ADR-021).

## [0.12.0] - 2026-11-23

### Added

- Tasks have attachments: upload photos and files in the task editor, as in notes. A note turned into a task brings its photos along, and they move with the task to another board (ADR-019).

### Changed

- Board cards look like cards: the whole card opens the task, "dringend" and "wichtig" appear as words instead of D and W, and at the bottom right the round picture with initials shows who took the task, or else who wrote it, on every board.

## [0.11.3] - 2026-11-22

### Changed

- A task taken on a group board shows who took it as a round picture with their initials below; the full name appears on hover. The task page shows the picture next to the name.

## [0.11.2] - 2026-11-19

### Fixed

- On a phone, − amount + on a shopping item is one compact group, so names like "Rinderbouillon" keep their line; names no longer break inside a word (a long word is hyphenated), and on very narrow screens − amount + moves below the name.

## [0.11.1] - 2026-11-19

### Fixed

- The installed app's system bars follow the theme: the pages say they support light and dark and give a theme colour for each, and the manifest's colours are the dark theme's instead of a light background.

## [0.11.0] - 2026-11-19

### Added

- The cook view shows a progress bar over the steps, like a to-do list.
- Every shopping item gets − and +: an item without an amount counts as 1, and "3x" or "3 x" is read as a factor. The amount sits at − and +, the item shows without it; "7Up" or "2 - 3 Eier" stay as written.

## [0.10.1] - 2026-11-17

### Changed

- The summary of what happened is now Kanso's own start page: "Start" is the first entry in the rail and where signing in lands; the wiki's spaces moved to "Wiki" (`/wiki`) and no longer carry the summary.

## [0.10.0] - 2026-11-17

### Added

- The start page says what happened since your last visit (at least the last 24 hours): a sentence, a number and spark line per kind, the changes over time, who was active and the changed documents by space. A select switches to 24 hours, 7 or 30 days; your own changes can be shown too. "Heute" lists today's meals and the tasks due today or overdue. Only what you may read counts, for admins too (ADR-018).

## [0.9.0] - 2026-11-17

### Added

- "Rezept generieren" in the kitchen: ingredients one fancies or has in the fridge, plus wishes (warm or cold, mild or spicy, a direction such as home-style, what must not go in, time, servings, a short note), and the AI makes up one recipe. "Neu würfeln" throws it away and makes another from the same wishes, "Wünsche ändern" goes back to the filled form, "Übernehmen" puts it in the cookbook tagged "ki-generiert". The AI writes only recipes: choices come from fixed lists, free text is cleaned and sent as marked data, and an answer that is not a recipe never shows (ADR-017).

### Fixed

- "Kochpläne" in the user menu: a plan whose group the admin belongs to links to it, so it can be opened from there.

## [0.8.1] - 2026-11-17

### Changed

- One space is the cookbook (ADR-016): new recipes and imports go there without asking for a space, the cookbook no longer shows in the wiki, and the kitchen says who sees the recipes, with a link to change it for owners and admins. A space holding only recipes becomes the cookbook by itself; without one, admins create it from the kitchen, and they can move it under "Alle Bereiche".

## [0.8.0] - 2026-11-17

### Added

- "Berichte" in the user menu: what Kanso holds and what it takes. The overview shows documents, storage and, for admins, what Git still keeps of old versions and deleted files; charts of what there is and how it grew, each with its numbers as a table. Everyone sees their own documents, what is shared with them, or both; admins see everyone by person — numbers only, never titles of things they cannot read.
- Berichte → Speicher: where the space goes per area (with what Git keeps of old versions, for admins), the repository against the current files over time, orphaned media (attachments nothing links to, attachments of deleted pages) and the largest files. Admins also see Kanso's memory, uptime and data size.
- Berichte → Aktivität: a calendar of changes per day over the last year and a weekday × hour map of when people write.
- Berichte → Wiki: how old the pages are, the ten untouched the longest, and the wiki as a sunburst of spaces, pages and subpages.
- Berichte → Aufgaben: new against done per week, and per Eisenhower field the open and overdue tasks and how long they take to get done.
- Berichte → Küche: the meals cooked most, what is bought most, recipes by tag and by preparation time.
- Berichte → Freigaben: who shares with whom as a network, and what you share and what is shared with you.

### Changed

- Recipes appear only in the kitchen: the wiki's spaces bar and subpage lists no longer show them (a page below a recipe moves up), and an opened recipe belongs to Küche, its path leading back to "Rezepte". Search still finds them.

## [0.7.0] - 2026-11-17

### Added

- "Was andere sehen" groups by kind, by document, by person or by right, and filters by right (Lesen, Schreiben, Verwalten): "Nach Person" shows for each person at a glance what they see and may do. Each row now says who may do what instead of only who sees it.

## [0.6.0] - 2026-11-16

### Changed
- Admins no longer read every space (ADR-014): one sees what oneself or one's groups are entered for, admins included. Whoever creates a space becomes its owner. "Sichtbar für …" names only those people ("Nur für dich sichtbar", "Sichtbar für dich, Anna"); "und Admins" is gone.
- A board of a group with only you in it says "Nur für dich sichtbar", like everything else.
- The wiki looks for pages shared with you only in spaces that have shared pages, instead of building every space's tree on each request.
- **Upgrade:** a space without members disappears from the wiki until an admin takes it over under "Alle Bereiche" → "Mich als Besitzer eintragen".

### Added
- "Was andere sehen" in the user menu: everything someone besides you sees — wiki spaces, pages shared one by one, lists, boards, meal plans — each with who that is. Notes never appear; they are always yours alone.
- "Alle Bereiche" for admins (`/admin/spaces`): every space with its members and page count, never its content; its settings and deletion are open to admins who are no members.

## [0.5.0] - 2026-11-16

### Added
- Delete a space: "Bereich löschen" at the end of its settings, for whoever may manage it. The confirmation counts pages, recipes, attachments, meal plan entries and shared pages; a space with pages asks for its name. One Git commit removes it, so the history keeps everything. Not while someone edits one of its pages.

- The wiki's spaces bar folds on wide screens: the panel icon next to "Wiki-Bereiche" leaves a narrow strip, and the device remembers it.

### Fixed
- A recipe test planned "the future" on 2026-11-12 and failed once that day had passed.

## [0.4.0] - 2026-11-11

### Added
- Shopping lists: an item one did not get moves to another shopping list or to a new one ("Einkauf 12.11."): tap it, "Auf andere Liste", "Verschieben". Note and "für …" go along, it arrives unticked, and a like item there adds up ("200 g Mehl" onto "300 g Mehl" gives "500 g Mehl"). Works offline like ticking.

## [0.3.0] - 2026-11-11

### Added
- Stars for recipes: one to five, set on the recipe page by whoever may edit it (pressing the given stars again clears them), kept as `rating` in the page's frontmatter and its history. The kitchen and the search show them in the rows and filter by "ab 3 Sternen", "ab 4 Sternen" and "5 Sterne".

### Fixed
- Recipes imported from a web page get their headings ("Zutaten", "Zubereitung") in the page's language, else in the person's Kanso language; the AI is told that language too, instead of writing "Method" for a German household.

## [0.2.1] - 2026-11-11

### Fixed
- Phones: the bottom bar no longer jumps or leaves an empty strip when scrolling (no `viewport-fit=cover`, no safe-area padding); the top bar shows only the profile picture so the search has room; page toolbars stay one row with Verschieben, Verlauf and Löschen behind "⋯"; Kamera and Foto wählen share a row with Speichern below in full width; the new-list buttons share a row; shopping items keep the note and "für …" below in the full width; the meal plan's day ticks are big enough for a thumb.

## [0.2.0] - 2026-11-10

### Added
- One search with filters: the search shows recipes in their own group with photos and a jump into the kitchen, then pages and own notes; "Nur Rezepte" and the tags of the hits filter by click, and a tag alone lists everything with it. Tags on wiki pages lead into this filtered search; the kitchen's search box uses the same search.
- Tags for every wiki page, not only recipes: free words typed in the editor, created on save, shown on the page as links (recipes into the kitchen, other pages into the search). Unticking "Ist ein Rezept" keeps them.
- "Heute gekocht" in the cook view notes the day in the recipe (once per day, in its history); the recipe page shows "Gekocht": how often and when last, the days from the meal plans and the days noted by hand together (planned days marked).
- The meal plan shows each meal's photo; a recipe page lists the days it is on the meal plans of one's groups ("Im Kochplan"), latest first, each a jump into that week.

### Fixed
- The cook view renders bold, italics and links in steps and ingredients instead of showing the raw Markdown (`__…__`, `**…**`).
- Shopping previews no longer show "Vorschau aktualisieren" when the page updates by itself (only without script), and choices wrap with room between rows.
- Elements marked hidden stay hidden even when a component sets its own display (e.g. "Häkchen entfernen" without script).

## [0.1.0] - 2026-11-07

First numbered release: everything Kanso can do so far.

### Added
- Version number and "Über Kanso" (`/about`): version, copyright, licence (Apache-2.0), links to GitHub and huepattl.de, what Kanso sends where, and the licenses and thanks; the footer of every page reads "Hüpattl! Kanso 0.1.0 · Über Kanso".
- Functions that use the AI carry a mark "KI · extern: host" or "KI · im eigenen Netz: host" (import form, the draft made with AI, the AI connection page); Kanso resolves the host to tell which.
- Repository skeleton, configuration from environment, `/healthz`.
- Safe file paths, slugs, atomic Markdown documents with YAML frontmatter.
- Git history for the data directory: commit per change with author, log, show, diff, move.
- Access model: roles viewer, editor, owner per space; admins; groups in `kanso.yaml`.
- User profiles in `users/<sub>.yaml`, created at sign-in, with interface language.
- OpenID Connect sign-in (code flow + PKCE, RS256/ES256 ID token verification) with the Go standard library.
- Sessions (30 days, sliding), CSRF protection with token and origin check, password-less dev login on loopback only.
- German and English interface texts, language from profile or browser.
- Web shell: Rams design tokens, self-hosted IBM Plex, mobile-first layout, security headers, "Licenses and thanks" page, language switch.
- End-to-end tests with Playwright (desktop Chromium, iPhone WebKit).
- Spaces with members (people or groups as viewer, editor, owner); admins create spaces and manage groups; own line icons.
- Wiki pages as folders with `index.md`: page tree, stable ids (`/p/<id>`), sub-pages, move, delete with confirmation; history kept through `git mv`.
- Safe Markdown rendering (goldmark, GitHub-flavoured): no raw HTML, link allowlist, attachment paths per page, Mermaid blocks marked for the diagram script; preview endpoint for the editor.
- Attachments per page (up to 50 MiB): safe readable file names, fixed content types, SVG and unknown types only as download.
- Edit locks (ADR-003): five minutes after the last heartbeat, owners and admins can break them; saving without the lock is refused (409) and keeps the text.
- Markdown editor: source and live preview (side by side on wide screens, tabs on phones), attach files by button, drag and drop or paste, ⌘S/Ctrl+S, warning before leaving with unsaved changes.
- Mermaid diagrams: mermaid 12.2.0 vendored without the ELK layout (EPL-2.0) and without source maps, loaded only on pages with diagrams and in the editor; strict security level, labels as plain SVG text.
- Page history: every version with author and time, differences of the text to the current version, restore as a new version (respects edit locks).
- Search across titles and texts of all readable spaces: in memory, umlauts match either spelling, last word as prefix, title hits first, plain-text snippets.
- Deployment guide for Authelia (`docs/deploy-authelia.md`), including the claims policy Authelia ≥ 4.39 needs.
- `flake.nix`: Nix package (`buildGoModule`, Go 1.27.3, tests run with git).
- Development data: `just seed` fills an empty `./tmp` with made-up people, a group, three spaces with different roles and sample pages; `just reset` deletes it. `kanso seed` refuses to run without the dev login.
- Dev login: one click per made-up person, and "Switch person" in the user menu returns to the same page.
- Notes (ADR-004): private notes per person, outside the Git history. Quick capture page (`/notes/new`) with camera and photo library; list, view, edit with source and preview, delete; turn a note into a wiki page; search finds your own notes; the seed adds some.
- Offline capture: notes and photos wait on the device (IndexedDB) and upload when the network is back; a lost answer never creates a note twice; a signed-out device keeps its notes and offers to sign in.
- Installable web app (manifest, own icon) whose home-screen icon opens the capture page.
- To-do lists (ADR-006): flat lists with one level of sub-items, outside the Git history; ticked items stay in place and fade; overview by last change; the owner shares with people or groups, renames and deletes; ticking and editing work offline and sync when the network is back.
- Account menu: your picture (round) or a placeholder with your name opens it; inside, who you are, language as two buttons (no "Apply"), group management and signing out, each with an icon. A profile picture named by the sign-in (`picture`, Authelia 4.39 can send it) is copied once per sign-in and served by Kanso.
- Groups can be deleted; members who were never signed in are listed by their id.
- Every page says who can see it (ADR-007): spaces, pages and the space list name the people with access plus admins; notes say only you; to-do lists name their owner and shares.
- To-do lists show progress: a bar with percent between the heading and the items; the overview shows a short bar per list, whose shape tells open, partly done and done apart, green when done.
- Tasks: one private board per person and boards per group, each task a Markdown file in the Git history (ADR-008).
- Tasks part in the rail: a board with the tabs Kanban (Offen · In Arbeit · Wartet · Erledigt; done tasks older than 14 days on request) and Matrix (Sofort, Planen, Kurz erledigen, Später); cards show due words, what a task waits for and who took it.
- A task's page: Markdown text, column, urgent/important, due date, what it waits for, the note it came from (for its author); edit with preview, take or give back a task of a group board, move it to another board, delete it with confirmation. Saving never overwrites a change made meanwhile.
- Quick task capture (`/tasks/new`, "Neue Aufgabe"): subject, board, urgent, important and due (heute, morgen, diese Woche = coming Sunday, or a date); works offline with an outbox like notes, a task sent twice is stored once, a task for a board that is gone lands on the own board.
- Drag cards between Kanban columns and matrix quadrants (desktop); the board redraws from the server, a failed move leaves the card in place. The device remembers the last board and tab.
- Admins manage group boards (`/admin/boards`, "Gruppen-Boards" in the account menu): create one for a group, rename it, delete it after confirmation.
- A note becomes a task ("Als Aufgabe …"): first line as subject (as notes name it), the rest as text without the photo links; the note stays. The note links to its task by id (`/tasks/t/<id>`), so the link survives moving the task; when the task is gone, the note offers "Als Aufgabe …" again.
- `just seed` adds a "Familie" board for the group with tasks in every column and two private tasks for Anna.
- The line that says who can see something links to "Freigabe ändern" for whoever may change it: space owners, list owners, admins on group boards.
- Share a single wiki page (and its subpages) with people or groups, with a role, as an exception to its space (ADR-009): "Freigabe ändern" on a page opens the page's sharing, with a hint how many subpages it covers. Whoever gets a page this way finds it under "Für dich freigegeben" in the wiki navigation and on the wiki start page, and sees nothing else of the space.
- Move a page with its subpages, attachments and history to another space ("Verschieben nach" lists every space you may write); a confirmation shows who sees the page until now and afterwards when that changes; a page being edited stays where it is.
- Recipes (ADR-010): a wiki page becomes a recipe with "Ist ein Rezept" in the editor, with servings, time, tags and source in its frontmatter. Kanso reads `## Zutaten` (amount, unit, name, note; `###` groups) and `## Zubereitung`, and the editor preview shows how it read each ingredient; lines it does not understand stay as they are.
- A recipe page shows servings, time, source and its tags (links into "Küche"), and "Kochen" in its toolbar; search finds recipes by their tags.
- Cook view for recipes (`/recipes/<id>/cook`): servings up and down with amounts scaled and rounded sensibly, ingredients and steps to tick off (kept on the device), screen kept on while it is open.
- "Küche" in the rail: every recipe the person may read, with a search over title, ingredients and tags and tag filters that switch on and off; "Neues Rezept" creates a recipe page from a template in a chosen space.
- Browser test for recipes: create, find by tag, cook with more servings, tick off, clear ticks.
- Shopping lists (ADR-011): "Einkaufen" on a recipe (and in the cook view, with its servings) previews every ingredient ticked and adds them to an existing or a new shopping list; amounts of the same ingredient add up in compatible units (g/kg, ml/l, same counted unit), never into an item already bought, and every item remembers which recipe it is for.
- Shopping lists under "Todos" ("Einkaufsliste anlegen"): type the amount with the item ("2 l Milch"), change the amount with − and + in steps that suit the unit (10 g below 100 g, then 50 g; ½ kg or l; 1 for counted things) or tap the amount between them to type an exact one, add a note with alternatives; items from recipes show what they are for. Offline, sharing and syncing work as for any list.
- AI connection (ADR-013): admins register one OpenAI-compatible AI (Ollama, OpenRouter, OpenAI …; token optional, kept in the state directory, never shown or logged) and test it. When it is on, everyone may use it.
- "Von Webseite" in the kitchen: a recipe page becomes a recipe draft from its JSON-LD recipe data, or with the AI where it has none; the page says where text goes, and the recipe is created only after checking the draft. Pages are fetched only from public http(s) addresses (no own network), with size and time limits.
- Recipe photos: importing offers the page's photos (the dish, step photos from its recipe data, other photos in its content; no icons or thumbnails) to place on top, after a step or at the end; they become attachments (JPEGs without GPS), the one on top is the recipe's `image`. The kitchen shows a small photo per recipe (also the first photo of recipes typed by hand), the cook view shows step photos below their step.
- Import keeps working with messy answers and pages: thinking and text around a fenced answer are dropped, a closing remark after the steps is no step, empty steps are skipped, Latin-1 pages read correctly, JSON-LD with raw line breaks is read, credentials never land in the source. A stored AI token is dropped when the base URL moves to another host.
- Meal plans (ADR-012): admins create a plan for a group ("Kochpläne"); under "Küche › Kochplan" its members see the week from Monday to Sunday, plan meals per day or with "Einplanen" on a recipe, and take them off again.
- Shop for the week: tick days in the meal plan and "Einkaufen" adds their meals to a shopping list with one preview, amounts added up across meals; shopped meals are marked and not offered again. Taking a shopped meal off the plan offers to take its ingredients off the list again, leaving what is bought and what was typed by hand.
- Filling keeps your words after the amount, takes the amount for an item that had none, shows per recipe how much it asked for ("für Brot (500 g)"), adds nothing twice on a double tap, and never ticks an amount that grew after an offline tick.

### Security
- GPS location is removed from JPEG photos on upload (notes and wiki pages); orientation and other EXIF data stay.
- Requires Go 1.27.3 (`go.mod`), which fixes GO-2026-6809 in `crypto/x509`.

### Changed
- A new shopping list gets its name from where it comes from ("Einkauf: Kartoffelcurry", "Einkauf KW 46 (9.–15.11.)") instead of asking for one; "Einkaufen" in the meal plan without ticked days takes every day of the week with meals, and an empty week says so.
- The button that creates a to-do list says "Liste anlegen" (was "Anlegen"), next to the new "Einkaufsliste anlegen".
- Navigation (ADR-005): a rail names the parts of Kanso — Notizen, Todos and Wiki — at the bottom on phones and at the left on wide screens; the side bar shows only the wiki's spaces and pages, under "Wiki-Bereiche"; group management moved to the user menu.
- The page tree shows guide lines for every level, so nested pages stay apart on narrow screens.
- Keeping a folder out of the history is one helper (`Repo.Ignore`) for notes and to-do lists.
- Uploads share one path (`store.SaveUpload`) for safe names, size limit and photo cleaning.
- The Markdown editor serves pages and notes; it reads its URLs from the form.
- Select fields look like fields (frame plus chevron), no longer like buttons.
- Every content screen (wiki start, space, page, notes, note, to-do list, board, task) has a fixed screen head: a toolbar with the actions and who sees it, the path and the title stay at the top while the content scrolls; no buttons below the content any more. On phones the secondary actions show as icons. Pages: Verschieben is its own page.

### Fixed
- Signing in failed with "Bad Request" when a second login started meanwhile (the service worker refreshing a page while signed out); each login now keeps its own cookie.
- Creating a group named as people write it ("Familie", "Großeltern") failed with a bare "Bad Request"; such names are allowed now, and a refused change says why on the page. Failures to save are logged and answered as server errors.
- Sub-items of task lists keep their indent.
- Phones: the bottom bar no longer jumps while Safari's toolbar shows or hides; page heights use `svh`.
- Task lists with blank lines between items no longer show bullets next to the checkboxes.
- On phones the folded menu no longer stretches into an empty grey block on short pages.
- Page history no longer shows commits of another page: Kanso follows real renames itself instead of `git log --follow`, which took look-alike new pages for copies.
- Sign-in no longer fails for names with quotes or angle brackets (they are dropped from the Git author only).
- Design review: button links no longer underlined, navigation folded on phones, neutral checkboxes.
- ECDSA keys parsed with the non-deprecated standard-library API.
- Documentation: arc42 architecture, ADRs 000–003, feature descriptions.
