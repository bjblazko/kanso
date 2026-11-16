# Bereich löschen — Kurzdesign

**Created:** 2026-11-16

## Warum

Der Owner hat Inhalte zwischen Bereichen verschoben; jetzt stehen Bereiche
herum, die weg sollen. Heute lässt sich ein Bereich nicht löschen.

## Ziele

- Ein Bereich lässt sich mit allem Inhalt löschen: Seiten, Unterseiten,
  Anhänge, Rezepte.
- Vorher sagt Kanso, was verloren geht.

## Keine Ziele

- Kein Papierkorb, kein Archiv, kein Wiederherstellen in Kanso. Git behält
  alles; zurückholen geht von Hand.

## Anti-Ziele

- Ein voller Bereich verschwindet nicht durch einen versehentlichen Klick.
- Andere Teile (Kochpläne, Einkaufslisten, Suche) zeigen danach keine
  kaputten Links und keine Fehlerseiten.

## Ansätze

| | Ansatz | Abwägung |
|---|---|---|
| 1 | Nur leere Bereiche löschen | am sichersten, bei vollen Bereichen mühsam |
| **2 (gewählt)** | Löschen mit allem Inhalt, Bestätigung durch Eintippen des Namens | ein Schritt; Sicherheitsnetz ist Git |
| 3 | Archivieren | umkehrbar, aber die Bereiche bleiben herumstehen |

Der Owner hat Ansatz 2 gewählt (2026-11-16).

## Design

- **Wer:** wer den Bereich verwalten darf (`access.ManageSpace`), also
  dieselben Leute wie für die Einstellungen.
- **Wo:** in den Einstellungen des Bereichs als letzter Abschnitt
  „Bereich löschen“: ein Satz und ein Link (roter Text) zur Bestätigung
  `/s/{space}/delete`.
- **Bestätigung:**
  - Titel „„<Name>“ löschen?“, dann was verloren geht: „12 Seiten, davon
    3 Rezepte, 8 Anhänge“. Dazu die Zahl der Kochplan-Einträge mit Rezepten
    aus dem Bereich und die Zahl der freigegebenen Seiten, wenn es welche gibt.
  - Ein Hinweis, dass der Inhalt in der Git-Historie bleibt.
  - Ein voller Bereich verlangt den Namen im Eingabefeld. Bei einem leeren
    Bereich reicht der Knopf. „„<Name>“ löschen“ ist rot (destruktiv), daneben
    „Abbrechen“.
- **Wird gerade eine Seite des Bereichs bearbeitet** (Sperre lebt), wird das
  Löschen abgelehnt und Kanso nennt, wer gerade arbeitet.
- **Wirkung:** ein Git-Commit „Bereich gelöscht: <Name>“, der
  `spaces/<slug>/` entfernt. Danach ist der Bereich aus dem Speicher weg, die
  Seiten sind aus dem Wiki-Index und der Suche entfernt (`OnChange` je Seite).
  Weiterleitung auf die Startseite mit dem Hinweis „Bereich „<Name>“
  gelöscht.“
- **Was bleibt:** Kochplan-Einträge zeigen den Rezepttitel ohne Link, so wie
  heute bei Rezepten, die man nicht lesen darf. Notizen und Aufgaben sind nicht
  betroffen. Einkaufszeilen „für <Rezept>“ setzen den Link im Browser; er
  führt danach auf „nicht gefunden“, so wie heute schon nach dem Löschen einer
  einzelnen Rezeptseite. Das kommt ins Backlog, nicht in diese Änderung.

## Dem Builder überlassen

- Wie Wiki und Bereiche zusammen löschen, ohne dass ein Commit halb gelingt
  (ein Commit unter `repo.Locked`).
- Wortlaut der Texte (de/en wie die übrigen).
- Tests: Unit-Tests in `spaces` und `wiki`, Go-HTTP-e2e (Rechte, Name falsch,
  Sperre, Erfolg, Kochplan danach), Playwright für den Ablauf.
