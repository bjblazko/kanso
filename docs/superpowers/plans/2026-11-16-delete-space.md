# Bereich löschen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Wer einen Bereich verwalten darf, kann ihn mit allem Inhalt in einem Git-Commit löschen, nach einer Bestätigung, die zeigt, was verloren geht.

**Architecture:** `spaces.Service.Delete` entfernt `spaces/<slug>/` und committet unter `repo.Locked`. Ein Callback darin lässt den Wiki-Index den Bereich vergessen, so dass keine Seite des Bereichs zwischen Löschen und Index-Update gespeichert werden kann. Die Bestätigung zählt über `wiki.SpaceStats` und `mealplan.Store.Uses`; der Handler prüft Sperren über `wiki.Locks.Holder`.

**Tech Stack:** Go stdlib, html/template, Git CLI über `store.Repo`, Playwright.

**Spec:** `docs/superpowers/specs/2026-11-16-delete-space-design.md`

## Global Constraints

- Jede neue Datei beginnt mit `SPDX-License-Identifier: Apache-2.0`.
- Keine neue Abhängigkeit.
- Funktionen ~30 Zeilen, zyklomatisch ≤ 10.
- Git nur über `store.Repo` (Programm mit Argumentliste).
- Commit-Nachricht des Löschens: `Bereich gelöscht: <Name>`.
- Recht: `access.ManageSpace`; Fremde bekommen 404 (`spaceFor`), Leser/Schreiber 403.
- Texte in `internal/i18n/de.yaml` und `en.yaml`.
- Doku im selben Commit: CHANGELOG `[Unreleased]`, Feature-Doc, `docs/design.md`, BACKLOG.

## Review Focus

- Falscher oder leerer Name bei vollem Bereich → 400, Bereich bleibt, Formular zeigt Fehler. (Task 4)
- Name mit Leerzeichen am Rand → wird getrimmt und gilt. (Task 4)
- Eine Seite des Bereichs ist gerade im Editor gesperrt → 409, Name des Bearbeiters, nichts gelöscht. (Task 4)
- Nach dem Löschen: `/p/<id>` 404, Suche findet die Seite nicht, Kochplan zeigt den Rezepttitel ohne Link, Startseite ohne den Bereich. (Task 4)
- Leerer Bereich → löschbar ohne Namenseingabe. (Task 4)

---

### Task 1: `spaces.Service.Delete`

**Files:**
- Modify: `internal/spaces/spaces.go`
- Test: `internal/spaces/spaces_test.go`

**Interfaces:**
- Produces: `var ErrUnknown = errors.New("unknown space")` (auch in `Update` statt des Inline-Fehlers verwenden); `func (svc *Service) Delete(a store.Author, slug string, also func()) error` — unter `svc.mu` und `repo.Locked`: `os.RemoveAll(dataDir/spaces/<slug>)`, `also()` (darf nil sein), Eintrag aus `svc.spaces` löschen, `repo.Commit(a, "Bereich gelöscht: "+name, "spaces/<slug>")`.

- [ ] **Step 1: Failing tests** `TestDeleteRemovesFolderAndCommits` (Bereich mit Datei `pages/x/index.md` anlegen; nach `Delete`: Ordner fehlt, `Get` → false, `List` ohne ihn, `repo.Log("spaces/<slug>", 1)[0]` hat Nachricht `Bereich gelöscht: Haus`, `also` wurde genau einmal aufgerufen), `TestDeleteUnknownSpace` (`errors.Is(err, ErrUnknown)`, `also` nicht aufgerufen).
- [ ] **Step 2:** `go test ./internal/spaces/ -run Delete` → FAIL (undefined).
- [ ] **Step 3:** `Delete` und `ErrUnknown` implementieren.
- [ ] **Step 4:** `go test ./internal/spaces/` → PASS.
- [ ] **Step 5:** Commit `feat(spaces): delete a space with its content`.

### Task 2: Wiki — zählen und vergessen

**Files:**
- Create: `internal/wiki/space.go`
- Test: `internal/wiki/space_test.go`

**Interfaces:**
- Produces:
  ```go
  type SpaceStats struct {
      IDs                                 []string // all page ids of the space
      Pages, Recipes, Attachments, Shared int      // Shared: pages with own Members
  }
  func (w *Wiki) SpaceStats(space string) SpaceStats
  func (w *Wiki) ForgetSpace(space string) // drops the space's pages from the index, then OnChange(p, true) for each, after releasing w.mu
  ```
  `Attachments` summiert `ListAttachments(id)` je Seite (nach dem Lesen des Index, ohne `w.mu` zu halten).

- [ ] **Step 1: Failing tests** `TestSpaceStats` (Bereich „haus“: zwei Seiten, eine davon Rezept mit einem Anhang, eine mit `Members`; anderer Bereich „garten“ mit einer Seite → `{Pages:2, Recipes:1, Attachments:1, Shared:1}`, `len(IDs)==2`), `TestForgetSpace` (danach `ByID` → false für „haus“-Seiten, true für „garten“; `OnChange` mit `deleted=true` genau für die zwei „haus“-Seiten).
- [ ] **Step 2:** `go test ./internal/wiki/ -run 'SpaceStats|ForgetSpace'` → FAIL.
- [ ] **Step 3:** Implementieren.
- [ ] **Step 4:** `go test ./internal/wiki/` → PASS.
- [ ] **Step 5:** Commit `feat(wiki): count and forget a space's pages`.

### Task 3: Kochpläne — Einträge mit Rezepten zählen

**Files:**
- Modify: `internal/mealplan/mealplan.go`
- Test: `internal/mealplan/mealplan_test.go`

**Interfaces:**
- Produces: `func (s *Store) Uses(recipes map[string]bool) int` — Zahl der Einträge in allen Plänen und Wochen (`<plan>/*-W*.yaml`), deren `Recipe` in `recipes` ist.

- [ ] **Step 1: Failing test** `TestUsesCountsEntriesAcrossPlansAndWeeks` (zwei Pläne, zwei Wochen, Einträge mit Rezept a, a, b, c → `Uses({a,b})==3`, `Uses({})==0`).
- [ ] **Step 2:** `go test ./internal/mealplan/ -run Uses` → FAIL.
- [ ] **Step 3:** Implementieren (wie `CookedOn`, aber über `s.All()`).
- [ ] **Step 4:** `go test ./internal/mealplan/` → PASS.
- [ ] **Step 5:** Commit `feat(mealplan): count entries that use given recipes`.

### Task 4: Bestätigung und Löschen im Web, Doku

**Files:**
- Create: `internal/web/handlers_space_delete.go`, `web/templates/space_delete.html`, `internal/web/handlers_space_delete_test.go`, `docs/features/2026-11-16 delete-space.md`
- Modify: `internal/web/routes.go` (neben `/s/{space}/settings`), `web/templates/space_settings.html` (letzter Abschnitt), `web/templates/home.html` (Hinweis), `internal/web/handlers_spaces.go` (`home` liest `?deleted=`), `internal/i18n/de.yaml`, `internal/i18n/en.yaml`, `CHANGELOG.md`, `docs/design.md`, `docs/BACKLOG.md`

**Interfaces:**
- Consumes: Task 1–3; `s.Locks.Holder(id) (wiki.Lock, bool)`, `spaceFor`, `s.forbidden`, `s.render`.
- Produces: `GET /s/{space}/delete` (Bestätigung), `POST /s/{space}/delete` (Feld `name`) → 303 `/?deleted=<urlquery Name>`.
- Daten fürs Template: `spaceDelete{Space spaces.Space; Stats wiki.SpaceStats; Planned int; Error string}`; `NeedsName()` = `Stats.Pages > 0`.

Texte (de; en sinngemäß):
- `space.delete_section`: „Bereich löschen“
- `space.delete_hint`: „Löscht den Bereich mit allen Seiten, Rezepten und Anhängen.“
- `space.delete_confirm`: „„%s“ löschen?“
- `space.delete_counts`: „%d Seiten, davon %d Rezepte, %d Anhänge“
- `space.delete_planned`: „%d Kochplan-Einträge nutzen Rezepte von hier“
- `space.delete_shared`: „%d Seiten sind freigegeben“
- `space.delete_git`: „Der Inhalt bleibt in der Git-Historie.“
- `space.delete_type_name`: „Zum Bestätigen den Namen eintippen“
- `space.delete_named`: „„%s“ löschen“
- `space.delete_wrong_name`: „Der Name stimmt nicht.“
- `space.delete_locked`: „%s bearbeitet gerade eine Seite in diesem Bereich.“
- `space.deleted`: „Bereich „%s“ gelöscht.“

- [ ] **Step 1: Failing HTTP tests** (mit `setupHaus`):
  - `TestDeleteSpaceRights`: anna (Editor) GET und POST → 403; ben (fremd) → 404.
  - `TestDeleteSpaceConfirmShowsCounts`: Admin legt zwei Seiten an, eine als Rezept, plant sie in einem Kochplan ein → GET enthält „2 Seiten, davon 1 Rezepte, 0 Anhänge“ und „1 Kochplan-Einträge“.
  - `TestDeleteSpaceWrongName`: POST `name=Hau` → 400, Text „Der Name stimmt nicht.“, `/s/haus` weiter 200.
  - `TestDeleteSpaceLocked`: anna öffnet `/p/<id>/edit` (Sperre), Admin POST mit richtigem Namen → 409, enthält „Anna bearbeitet gerade“, `/s/haus` weiter 200.
  - `TestDeleteSpace`: POST `name= Haus ` → 303, `Location` = `/?deleted=Haus`; danach `/s/haus` 404, `/p/<id>` 404, `/search?q=<Seitentitel>` ohne Treffer, Kochplan-Woche enthält den Rezepttitel ohne `href="/p/<id>"`, `/?deleted=Haus` enthält „Bereich „Haus“ gelöscht.“ und kein `href="/s/haus"`.
  - `TestDeleteEmptySpaceNeedsNoName`: neuer Bereich ohne Seiten, POST ohne `name` → 303.
  - `TestSettingsLinkToDelete`: Settings enthält `href="/s/haus/delete"`.
- [ ] **Step 2:** `go test ./internal/web/ -run 'DeleteSpace|DeleteEmptySpace|SettingsLinkToDelete'` → FAIL.
- [ ] **Step 3:** Handler `deleteSpaceForm`, `deleteSpace` implementieren. Reihenfolge im POST: Recht → Stats → Name prüfen (`strings.TrimSpace`, nur wenn `NeedsName`) → erste lebende Sperre unter `Stats.IDs` → `s.Spaces.Delete(author, slug, func() { s.Wiki.ForgetSpace(slug) })` → Redirect. Template wie `page_delete.html`, Knopf `btn btn-destructive`, daneben `btn btn-secondary` „Abbrechen“ zu den Einstellungen; Namensfeld `<input name="name" required autocomplete="off">` nur bei `NeedsName`. Settings: letzter Abschnitt `<h2>` + Hinweis + `<a class="btn btn-quiet tb-danger" href="/s/<slug>/delete">`. Home: `notice` mit `role="status"`, wenn `deleted` gesetzt und es keinen Bereich mit diesem Namen gibt.
- [ ] **Step 4:** `just test` → PASS; `just lint` sauber.
- [ ] **Step 5: Doku:** Feature-Doc (`status: done`, `created`/`updated` 2026-11-16, `## Related` → Spec), CHANGELOG `[Unreleased]` → `### Added`, `docs/design.md`-Zeilen „Space settings“ (letzter Abschnitt „Bereich löschen“) und „Delete space“ (neue Zeile: Job, Zahlen, Name eintippen nur bei Inhalt, Rot = destruktiv), BACKLOG: „Einkaufszeilen ‚für <Rezept>‘ verlinken auch gelöschte Rezepte (404).“
- [ ] **Step 6:** Commit `feat(spaces): delete a space from its settings`.

### Task 5: Playwright

**Files:**
- Modify: `e2e/tests/spaces.spec.js`

- [ ] **Step 1: Test** `owner deletes a space with its pages`: Admin `createSpace(unique('Alt'))`, `createPage('Notiz')`, Einstellungen → Link „Bereich löschen“ → sieht „1 Seiten“ → Knopf ohne Namen → bleibt auf der Seite (HTML-`required`) → Name eintippen → Knopf → Startseite zeigt „Bereich „…“ gelöscht.“ und keinen Link auf den Bereich.
- [ ] **Step 2:** `just e2e` → PASS (alle Projekte).
- [ ] **Step 3:** Commit `test(e2e): delete a space`.
