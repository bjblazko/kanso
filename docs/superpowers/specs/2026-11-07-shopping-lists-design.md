# Hüpattl! Kanso – Einkaufslisten (Teil 2)

**Status:** freigegeben („go ahead“, 2026-11-07) · **Created:** 2026-11-07 · **Updated:** 2026-11-07

Zweiter von drei Teilen: [Rezepte](2026-11-06-recipes-design.md) → **Einkaufsliste** → Kochplan.
Baut auf den Todo-Listen auf ([2026-10-27-todos-design.md](2026-10-27-todos-design.md),
ADR-006) und dem Rezept-Parser (ADR-010).

## Warum

Wer einkauft, kocht meistens nicht. Die Einkaufsliste wird heute von Hand aus Rezepten
abgeschrieben; in Apps wie Bring muss man die Menge nach dem Anlegen in einem eigenen Feld
suchen. Wer im Laden steht, weiß nicht, wofür etwas gedacht ist, und muss anrufen, wenn es
das Gewünschte nicht gibt.

## Ziele (vom Nutzer)

- Einkaufsliste aus einem Rezept füllen: Rezept → Portionen → neue oder **bestehende** Liste.
- **Vorschau vor dem Übernehmen, alles angehakt**; man hakt ab, was im Haus ist (Variante A).
- Mengen gleicher Zutaten werden **zusammengerechnet**, auch mit dem, was schon auf der Liste steht.
- **Menge super einfach angeben** – nicht erst Zutat anlegen und dann ein Mengenfeld suchen.
- **Wofür** etwas ist und **Alternativen/Hintergrund** stehen am Eintrag, damit die einkaufende
  Person selbst entscheiden kann.
- Später: Bilder, Zuordnung zu konkreten Produkten je Laden (evtl. per KI) – Format lässt Platz.
- Die Herkunft je Eintrag wird gemerkt, damit Teil 3 (Kochplan) Einträge wieder abziehen kann.

## Entscheidungen (meine, zum Gegenlesen markiert)

- **Einkaufsliste = eine Art der bestehenden Listen** unter „Todos“ (`kind: shopping` in der
  Listendatei), nicht ein zweites System. Offline, Sync als Operationen, Teilen und Löschen
  kommen unverändert aus ADR-006. In der Übersicht mit eigenem Icon; „Neue Einkaufsliste“ neben
  „Neue Liste“. *(Weicht vom früheren Satz „eigener Listentyp, nicht die To-do-Listen“ ab, soweit
  es den Code betrifft: in der Oberfläche ist es eine eigene Art mit eigener Darstellung.)*
- **Die Menge steht im Text** des Eintrags: „2 l Milch“, „Milch 3“ ist auch erlaubt (dann ohne
  Erkennung). Ein Feld, Enter, fertig. Am Eintrag **− / +**, wenn der Text mit einer Menge
  beginnt: g/ml unter 100 in 10er-, ab 100 in 50er-Schritten (Nutzer, 2026-11-07: „bei Gramm muss feiner
  gearbeitet werden“), kg/l in ½, Stückzahlen in 1ern; krumme Werte springen auf die nächste Stufe.
- Jeder Eintrag einer Einkaufsliste hat zusätzlich eine **Notiz** (Alternativen, Hinweise; frei,
  bis 300 Zeichen) und **Herkunft** (`for`: Rezept-ID, Titel, Menge als Text) – nur vom Server
  beim Füllen gesetzt, in der Liste als „für Kartoffelcurry“ gezeigt (Link aufs Rezept).
- **Füllen** ist eine Server-Seite (`/recipes/{id}/shop`), braucht also Netz. Liste offline
  benutzen geht wie bisher.
- **Zusammenrechnen:** gleicher Schlüssel (Name klein, ohne Zusatz) **und** verträgliche Einheit
  (g/kg, ml/cl/l, oder gleiche Zähleinheit) → eine Zeile. Ergebnis in g bzw. ml, ab 1000 in kg
  bzw. l. Ein **abgehakter** (schon gekaufter) Eintrag wird nicht erhöht, sondern ein neuer
  angelegt. Zutaten ohne Menge (Salz) werden nicht doppelt eingetragen, nur die Herkunft ergänzt.
- Der **Zusatz** einer Zutat („festkochend“) kommt in die Notiz des neuen Eintrags.
- Singular/Plural („Zwiebel“/„Zwiebeln“) bleibt getrennt. *(Spätere Verbesserung.)*

## Nicht-Ziele

- Kochplan, „Tage wählen → Liste“, Rückbuchen beim Entfernen aus dem Plan (Teil 3).
- Vorrat/„meist im Haus“, Kategorien/Gänge im Laden, Sortierung nach Laden.
- Bilder, Produkte je Laden, KI.
- Füllen offline.

## Anti-Ziele

- Ein Eintrag geht beim Zusammenrechnen verloren oder eine Menge wird stillschweigend falsch.
- Etwas landet ohne Vorschau auf der Liste.
- Die einkaufende Person sieht nicht, wofür etwas ist.

## Verhalten

### Liste (`/todos/{id}` mit `kind: shopping`)

- Kopfzeile wie bei Todo-Listen. Einträge: Häkchen, Text, darunter klein die Notiz und
  „für <Rezept>, <Rezept>“. Beginnt der Text mit einer ganzen Zahl: − und + am Eintrag.
- Beim Bearbeiten eines Eintrags (antippen): Textfeld und **Notizfeld**; Unterpunkte gibt es bei
  Einkaufslisten nicht.
- Hinzufügen: ein Feld „Was fehlt? z. B. 2 l Milch“.

### Füllen (`GET/POST /recipes/{id}/shop`)

- Erreichbar über „Einkaufen“ in der Werkzeugleiste der Rezeptseite und der Kochansicht
  (dort mit den gerade gewählten Portionen).
- Formular: Portionen (vorbelegt), Ziel: bestehende Einkaufslisten, die die Person sehen darf,
  oder „Neue Einkaufsliste“ mit Namen (vorbelegt „Einkauf“). Darunter die Vorschau: jede Zutat
  angehakt, mit Menge für die Portionen und – wenn sie auf der gewählten Liste schon steht –
  „steht schon drauf: 200 g → 700 g“. Wechsel von Liste oder Portionen lädt die Vorschau neu
  (GET mit Parametern, ohne JS über einen „Aktualisieren“-Knopf).
- „N übernehmen“ (Akzent) wendet alles in einer Änderung an und führt auf die Liste.

### Daten

```json
{ "id": "…", "kind": "shopping", "items": [
  { "id": "…", "text": "700 g Mehl", "note": "Typ 405", "done": false,
    "for": [ { "recipe": "k3x9…", "title": "Brot", "amount": "500 g" } ] } ] }
```

Neue Operation `note` (Text der Notiz) im Sync; `for` wird nur serverseitig gesetzt und mit
der Liste ausgeliefert. Todo-Listen ohne `kind` bleiben unverändert.

## Tests

- Unit: Zusammenrechnen (g+kg, ml+l, Zähleinheit, unverträglich, ohne Menge, abgehakt),
  Formatieren der Summe, Füllen einer Liste (neu, bestehend, Herkunft), Operation `note`,
  `kind` bleibt erhalten.
- HTTP: Füll-Seite (Rechte auf Rezept und Liste, Vorschau, Übernehmen), Liste neu anlegen.
- Playwright: Rezept → Einkaufen → bestehende Liste → Vorschau → übernehmen → Liste zeigt
  Summe, Notiz, „für …“; + erhöht die Zahl; offline abhaken wie bisher.

## Dokumentation

ADR-011 „Einkaufslisten sind eine Art der Listen“; Feature-Doc; arc42; design.md; CHANGELOG;
Gesamtdesign (Phase 5 teilweise).
