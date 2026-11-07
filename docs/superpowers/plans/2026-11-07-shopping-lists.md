# Einkaufslisten (Teil 2) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Einkaufslisten als Art der Listen (offline, geteilt) mit Notiz und Herkunft je Eintrag, − / + an der Menge, und „Einkaufen“ aus einem Rezept mit Vorschau und Zusammenrechnen.

**Architecture:** `internal/todos` bekommt `List.Kind`, `Item.Note`, `Item.For`, die Operation `note` und serverseitige `Create`/`Change`. Ein neues reines Paket `internal/shopping` plant und wendet das Füllen an (Zusammenrechnen über `internal/recipes`). `internal/web` bekommt `/recipes/{id}/shop`; `todos.js` zeichnet Einkaufslisten anders.

**Tech Stack:** Go, html/template, plain JS (todos.js), Playwright.

**Spec:** `docs/superpowers/specs/2026-11-07-shopping-lists-design.md`

## Global Constraints

- SPDX-Kopf in neuen Go/JS-Dateien; keine neue Abhängigkeit; Funktionen ~30 Zeilen, zyklomatisch ≤ 10.
- TDD; Doku im selben Commit (CHANGELOG, design.md, Feature-Doc, ADR-011, arc42).
- Todo-Listen ohne `kind` verhalten und speichern sich wie bisher (alle neuen JSON-Felder `omitempty`).
- Notiz ≤ 300 Zeichen, darf leer sein. `for` setzt nur der Server.
- Arbeiten auf Branch `shopping`.

## Review Focus

1. Eine alte Liste ohne `kind`/`note`/`for` wird gelesen und unverändert zurückgeschrieben – Task 1.
2. Abgehakter Eintrag „200 g Mehl“ + Rezept „500 g Mehl“ → neuer Eintrag, der alte bleibt abgehakt – Task 2.
3. Unverträgliche Einheiten („2 EL Mehl“ + „200 g Mehl“) → zwei Einträge, keine Summe – Task 2.
4. Füllen in eine Liste, die die Person nicht sieht, oder aus einem Rezept, das sie nicht lesen darf → 404 – Task 3.
5. − bei „1 Zitrone“ geht nicht unter 1; Text ohne führende Zahl hat kein − / + – Task 4.

---

### Task 1: Listenart, Notiz, Herkunft, serverseitiges Anlegen und Ändern

**Files:** Modify `internal/todos/todos.go`, `internal/todos/ops.go`; Test `internal/todos/shopping_test.go`; Create `docs/adr/011-shopping-lists-are-lists.md`, `docs/features/2026-11-07 shopping-lists.md`; Modify arc42.

**Interfaces – Produces:**
```go
const KindShopping = "shopping"
type Source struct { Recipe, Title, Amount string } // json: recipe,title,amount
// List: Kind string `json:"kind,omitempty"`
// Item: Note string `json:"note,omitempty"`; For []Source `json:"for,omitempty"`; Stamps.Note int64 `json:"note,omitempty"`
// Op: ListKind string `json:"listKind,omitempty"` (nur create; "" oder KindShopping)
const MaxNote = 300
func (s *Store) Create(owner, name, kind string) (List, error)               // neue ID (store.NewID), validiert Name
func (s *Store) Change(sub string, g access.Groups, id string, fn func(*List) error) error // unter Sperre; ErrNotFound wenn unsichtbar
func NewItem(text, note string, now time.Time) Item                         // neue ID, Stamps = now
```
- [ ] Tests: `TestCreateShoppingByOp` (create mit listKind → Kind gesetzt; unbekannte listKind → abgelehnt), `TestNoteOp` (setzt, leert, > 300 abgelehnt, ältere Op verliert), `TestOldListRoundTrip` (JSON ohne neue Felder → nach Apply einer done-Op keine Felder `kind`/`note`/`for` in der Datei), `TestStoreCreateAndChange` (Create → Change fügt Item mit For hinzu → Get liefert es; Change auf fremde Liste → ErrNotFound).
- [ ] Rot sehen, umsetzen, grün, ADR-011 + Feature-Doc + arc42, Commit `feat(todos): shopping lists with notes and where items come from`.

### Task 2: Füllen planen und anwenden (`internal/shopping`)

**Files:** Create `internal/shopping/fill.go`, `internal/shopping/fill_test.go`.

**Interfaces – Produces:**
```go
type Line struct {
    Name   string // Zutat, wie sie auf der Liste heißt
    Text   string // neuer Text des Eintrags
    Old    string // bisheriger Text, wenn ein Eintrag ergänzt wird; "" = neuer Eintrag
    Item   string // ID des ergänzten Eintrags; "" = neu
    Note   string // Zusatz der Zutat (nur neue Einträge)
    Amount string // Menge aus diesem Rezept, z. B. "500 g"; "" ohne Menge
}
func Plan(items []todos.Item, ings []recipes.Ingredient, lang string) []Line
func Apply(l *todos.List, lines []Line, take func(i int) bool, src todos.Source, now time.Time) int // Anzahl übernommener Zeilen
```
Regeln (Spec „Zusammenrechnen“): Match über `Key`, nicht abgehakt; beide mit Menge und verträglich (Masse: g/kg; Volumen: ml/cl/l; sonst gleiche kanonische Einheit) → Summe der oberen Werte, Basis g/ml, ab 1000 kg/l, `recipes.FormatAmount`; eine Seite ohne Menge → Text bleibt, nur Herkunft; sonst neue Zeile. Zeilen werden nacheinander gegen eine Arbeitskopie geplant, sodass doppelte Zutaten im selben Rezept zusammenlaufen. Neuer Text: `Menge [Einheit] Name` (implizites Stück ohne Einheit).
- [ ] Tests (Tabellen): `200 g Mehl` + `0,5 kg Mehl` → `700 g Mehl`, Old `200 g Mehl`; `800 ml Milch` + `0,5 l Milch` → `1,3 l Milch`; `2 Zwiebeln` + `3 Zwiebeln` → `5 Zwiebeln`; `2 EL Mehl` + `200 g Mehl` → neue Zeile (Focus 3); abgehakt `200 g Mehl` + `500 g Mehl` → neue Zeile (Focus 2); `Salz` + `1 TL Salz` → Text `Salz`, Item gesetzt; Rezept mit zweimal `100 g Zucker` → eine Zeile `200 g Zucker`; Zusatz `festkochend` → Note; `Apply` mit take nur Zeile 0 → 1 übernommen, Herkunft an ergänztem und neuem Eintrag, abgehakter bleibt.
- [ ] Rot, umsetzen, grün, Commit `feat(shopping): plan and add a recipe's ingredients to a list`.

### Task 3: „Einkaufen“ – Füll-Seite

**Files:** Create `internal/web/handlers_shop.go`, `web/templates/recipe_shop.html`; Modify `routes.go` (`GET/POST /recipes/{id}/shop`), `server.go` (pageNames), `page_view.html` + `recipe_cook.html` (Aktion „Einkaufen“, Icon `cart`), `icons.html`, i18n, CSS, design.md, CHANGELOG; Test `internal/web/handlers_shop_test.go`.

**Interfaces – Consumes:** `pageFor(…, access.Read)`, `recipes.Parse`, `Scale`, `ClampServings`, `shopping.Plan/Apply`, `todos.Store.Visible/Create/Change`.
Formular-Parameter: `servings`, `list` (ID oder `new`), `name` (bei new), `take` (wiederholt, Zeilenindex).
- [ ] Tests: `TestShopPreview` (GET mit servings=6 → „1200 g Kartoffeln“, alle Checkboxen `checked`, bestehende Einkaufsliste wählbar, Todo-Listen nicht); `TestShopMergePreview` (Liste mit `200 g Kartoffeln` gewählt → „200 g Kartoffeln → 1000 g Kartoffeln“ bzw. formatiert); `TestShopApplyNewList` (POST list=new&name=Wocheneinkauf, take=0,1 → Redirect `/todos/<id>`, Liste Kind shopping mit 2 Einträgen, Herkunft Titel); `TestShopApplyExisting` (Summe im bestehenden Eintrag); `TestShopAccess` (Rezept ohne Leserecht → 404; fremde Liste → 404; Nicht-Rezept → 404); `TestRecipeToolbarHasShop`.
- [ ] Rot, umsetzen, grün, Commit `feat(shopping): fill a shopping list from a recipe with a preview`.

### Task 4: Einkaufsliste im Browser

**Files:** Modify `web/static/js/todos.js`, `internal/web/handlers_todos.go` (Messages), `web/templates/todos.html` (Icon cart in `#todo-icons`), i18n, CSS, design.md, CHANGELOG; Test `e2e/tests/shopping.spec.js`.
- Übersicht: Formular mit zwei Knöpfen „Liste anlegen“ / „Einkaufsliste anlegen“ (`create` mit `listKind`). Zeilen von Einkaufslisten mit Icon cart.
- Eintrag (Einkaufsliste): Text, darunter `meta`: Notiz · „für A, B“ (Links `/p/<recipe>`); − / + nur bei `/^(\d+)(\s.*)$/`, − nicht unter 1 (Op `text`). Bearbeiten: Text + Notiz (Op `note`) + Löschen; keine Unterpunkte. Hinzufügen-Label „Was fehlt? z. B. 2 l Milch“.
- [ ] Playwright zuerst: Einkaufsliste anlegen, „2 l Milch“ hinzufügen, + → „3 l Milch“, − zweimal → „1 l Milch“ und − deaktiviert (Focus 5); Notiz „laktosefrei“ setzen, neu laden, sichtbar; Rezept → Einkaufen → diese Liste → übernehmen → „für <Rezept>“ sichtbar.
- [ ] Rot, umsetzen, grün; alle e2e; Commit `feat(shopping): shopping lists in the browser with notes, amounts and what for`.

### Task 5: Abschluss

- [ ] `just lint`, `go test ./...`, `just e2e`, Rams-Review mit Screenshots (Liste, Füll-Seite, Handy), Funde beheben.
- [ ] Feature-Doc `done`, Gesamtdesign Phase 5 („Einkauf fertig; Wunsch-/Linklisten offen“), Commit.
