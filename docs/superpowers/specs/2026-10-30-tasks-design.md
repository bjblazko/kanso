# Hüpattl! Kanso – Aufgaben (Boards)

**Status:** freigegeben · **Created:** 2026-10-30 · **Updated:** 2026-10-30

Baut auf dem Gesamtdesign auf: [2026-10-26-kanso-design.md](2026-10-26-kanso-design.md) (Phase 4).
Ergänzt Notizen ([2026-10-27-notes-design.md](2026-10-27-notes-design.md)) und Todo-Listen
([2026-10-27-todos-design.md](2026-10-27-todos-design.md)).

## Warum

Vieles fällt zwischen Tür und Angel an und passt auf keine Todo-Liste: „Wir müssen noch Lohnsteuer
machen“, „Kannst du nachher den Hof fegen?“, „Wir müssen das Kind per Mail zur Ferienbetreuung
anmelden“. Das muss **am Handy schnell erfasst und grob eingeordnet** werden, auch ohne Netz. Am
Desktop wird in Ruhe nachgearbeitet: Text ergänzen, in **Kanban** den Stand und in der
**Eisenhower-Matrix** die Dringlichkeit und Wichtigkeit ordnen.

## Ziele (vom Nutzer)

- Schnell mobil erfassen und einordnen; Nacharbeit mit mehr Details am Desktop.
- Eine Aufgabe hat **Betreff** und optional **Text in Markdown**.
- Erfassen am Handy: Betreff, Board (meins/Familie), dringend, wichtig, „bis wann“ (Variante C).
- Erfassen geht **auch offline**.
- Jede Person erfasst für sich; es wird **nicht zugewiesen**.
- Ein **privates Board** pro Person und ein **gemeinsames Familien-Board**, aus dem sich jemand
  eine Aufgabe nehmen kann.
- **Kanban und Eisenhower** sind zwei Ansichten derselben Aufgaben, als **Unter-Tabs** eines
  Boards, nicht gleichzeitig auf einem Bildschirm.
- **Feste Spalten:** Offen · In Arbeit · Wartet · Erledigt. „Wartet“ deckt wartend und blockiert ab.
- Aus einer Notiz wird eine Aufgabe (Notiz bleibt, beide verlinkt).

## Nicht-Ziele

- Wiederkehrende Aufgaben (Nutzer: „brauchen wir nicht“). Falls je, dann später als
  Kalendereinträge (Phase 6).
- Aufgaben anderen zuweisen, Erwähnungen, Erinnerungen, Push.
- Spalten pro Board konfigurieren (Nutzer: feste Spalten).
- Boards offline ansehen oder bearbeiten; offline geht nur das Erfassen. *(meine Entscheidung)*
- Manuelles Umsortieren innerhalb einer Spalte oder eines Quadranten. *(meine Entscheidung)*
- Anhänge an Aufgaben; Fotos bleiben in der verlinkten Notiz. *(meine Entscheidung)*
- Aufgaben in der Suche. *(meine Entscheidung; kann später kommen)*
- Unteraufgaben, Etiketten, Kommentare.

## Anti-Ziele

- Erfassen am Handy dauert länger als: öffnen → Betreff tippen → Speichern. Alles andere ist
  optional und vorbelegt.
- Eine offline erfasste Aufgabe geht still verloren oder kommt doppelt an.
- Jemand sieht ein privates Board einer anderen Person – auch kein Admin.
- Kanban und Matrix zeigen unterschiedliche Wahrheiten (es gibt nur eine Aufgabe, zwei Ansichten).

## Verhalten

### Leiste und Übersicht (`/tasks`)

- Eintrag **„Aufgaben“** in der Leiste der Teile (ADR-005), zwischen Todos und Wiki; eigenes
  Linien-Icon im Kanso-Stil.
- `/tasks` führt direkt auf **Mein Board**. Gibt es weitere sichtbare Boards (Gruppen-Boards),
  stehen sie als Reiter bzw. Auswahl oben („Mein Board · Familie“). Das zuletzt geöffnete Board
  merkt sich das Gerät (localStorage, mit try/catch).
- Hauptaktion auf jedem Board: **Neue Aufgabe** → Erfassen, mit diesem Board vorgewählt.

### Erfassen (`/tasks/new`, auch offline)

- Felder in dieser Reihenfolge, ein Bildschirm, mobil zuerst:
  1. **Betreff** (Pflicht, Fokus beim Öffnen, ≤ 200 Zeichen).
  2. **Board**: Segment „Mein Board“ + jedes sichtbare Gruppen-Board. Vorgabe: Mein Board bzw. das
     Board, von dem man kam.
  3. **dringend**, **wichtig**: zwei Kästchen, Vorgabe aus.
  4. **Bis wann**: Segment „—“ (Vorgabe) · heute · morgen · diese Woche · Datum… (öffnet ein
     Datumsfeld). „Diese Woche“ = kommender Sonntag (heute, wenn Sonntag). *(meine Entscheidung)*
- **Speichern** ist die einzige Akzentfarbe. Danach: leeres Formular für die nächste Aufgabe und
  eine kurze Bestätigung („Gespeichert in Familie“ bzw. „Wartet auf Netz“). Daneben ein Link
  „Zum Board“.
- Neue Aufgaben stehen in **Offen**.
- Kein Textfeld beim Erfassen; Text kommt am Desktop dazu.
- Offline wie bei Notizen (ADR-004): Der Service Worker hält `/tasks/new` und seine Dateien vor.
  Jede Aufgabe bekommt ihre ID auf dem Gerät und wartet in einem **Ausgang** (IndexedDB, eigene
  Datenbank `kanso-tasks`), bis der Server sie bestätigt; doppelt gesendet ergibt eine Aufgabe.
  Hochladen beim Speichern, beim Öffnen, bei `online`, beim Zurückkehren in den Tab und alle 30 s,
  solange die Seite sichtbar ist. Statuszeile „2 Aufgaben warten auf Netz“, ggf. „Zum Abgleichen
  bitte anmelden“.
- Die Board-Auswahl offline nimmt die zuletzt bekannte Liste der Boards (vom letzten Online-Besuch
  in IndexedDB). Kennt das Gerät noch keine, gibt es nur „Mein Board“.
- Ist ein gewähltes Board beim Hochladen nicht mehr sichtbar (gelöscht, aus der Gruppe entfernt),
  landet die Aufgabe auf **Mein Board** und die Statuszeile sagt es einmal. *(meine Entscheidung)*

### Ein Board (`/tasks/b/<board-id>`)

- Überschrift = Board-Name; darunter die Sichtbarkeitszeile (ADR-007): „Nur du“ bzw. „Familie“.
- Unter-Tabs **Kanban** · **Matrix** als Links (`?view=kanban|matrix`, `aria-current`); Vorgabe
  Kanban; das Gerät merkt sich den letzten Tab pro Board.
- **Karte**: Betreff; darunter in klein, nur wenn gesetzt: Fälligkeit („heute“, „morgen“, sonst
  Datum; überfällig mit Wort *und* Signalfarbe Rot, heute/morgen Gelb), „worauf“ (bei Wartet),
  „übernommen von Anna“ bzw. „von dir“ (Gruppen-Board). Kleine Marke **D**/**W** für dringend und
  wichtig im Kanban, damit beides ohne Matrix sichtbar ist (mit `title`/Screenreader-Text).
- Klick auf den Betreff öffnet die Aufgabe (Detailseite).
- **Reihenfolge** in Spalte und Quadrant: Fälligkeit aufsteigend, ohne Fälligkeit danach, dann
  neueste zuerst. *(meine Entscheidung)*
- Leerer Zustand: „Noch keine Aufgaben.“ mit **Neue Aufgabe**.

**Kanban**
- Vier Spalten Offen · In Arbeit · Wartet · Erledigt; breit nebeneinander, am Handy untereinander
  als Abschnitte mit Anzahl.
- **Erledigt** zeigt nur, was in den letzten **14 Tagen** erledigt wurde, plus Link „Ältere
  erledigte anzeigen“ (`?done=all`). Die Dateien bleiben. *(vom Nutzer bestätigt)*
- Am Desktop: Karte per **Ziehen** in eine andere Spalte. Ohne Ziehen (Handy, Tastatur, ohne JS):
  auf der Detailseite die Spalte wählen.

**Matrix**
- Vier Quadranten, Achsen beschriftet: dringend + wichtig „Sofort“, wichtig „Planen“, dringend
  „Kurz erledigen“, weder noch „Später“. Am Handy untereinander.
- Zeigt nur **nicht erledigte** Aufgaben. *(meine Entscheidung)*
- Ziehen in einen anderen Quadranten setzt dringend/wichtig; sonst auf der Detailseite.

### Aufgabe (`/tasks/b/<board-id>/<task-id>`)

- Ansicht: Betreff, Felder, gerenderter Text (gleicher sicherer Markdown-Renderer wie Wiki), Link zur
  Quell-Notiz (nur für deren Besitzer:in sichtbar; ist sie gelöscht: „Notiz gelöscht“), angelegt
  von/am, zuletzt geändert.
- **Bearbeiten** (eine Seite, Formular): Betreff, Text (Textarea mit Vorschau wie der Wiki-Editor,
  ohne Anhänge), Spalte, dringend, wichtig, bis wann (Datum oder leer), worauf (≤ 120 Zeichen; nur
  bei Wartet angezeigt, bleibt gespeichert, wenn die Spalte wechselt), Board verschieben (in ein
  anderes sichtbares Board).
- Gruppen-Board: **Übernehmen** bzw. **Freigeben** (setzt/entfernt `claimed`). Jede:r Sichtbare
  darf übernehmen, auch eine schon übernommene Aufgabe (Haushalt, keine Sperre); die Karte zeigt
  dann den neuen Namen. *(meine Entscheidung)*
- **Löschen …** mit Bestätigungsseite; die Datei verschwindet, der Git-Verlauf behält sie.
- Gleichzeitiges Bearbeiten: kein Sperren; das Formular schickt die gelesene Version mit. Hat sich
  die Datei inzwischen geändert, wird nicht gespeichert (409), der eingegebene Text bleibt im
  Formular und ein Hinweis erscheint (wie bei Notizen). Ziehen ändert nur ein Feld und gewinnt
  immer.

### Aus Notiz wird Aufgabe

- Auf der Notizansicht neben „Als Wiki-Seite …“: **Als Aufgabe …** → Seite mit Board-Auswahl.
- Betreff = erste Zeile der Notiz (ohne `#`, gekürzt auf 200 Zeichen), Text = der Rest; Spalte
  Offen, sonst nichts gesetzt. Danach öffnet sich die Aufgabe zum Bearbeiten.
- Die Notiz bleibt; sie bekommt den Verweis auf die Aufgabe (Frontmatter, als Link angezeigt), die
  Aufgabe den Verweis auf die Notiz (`note`). Anhänge bleiben in der Notiz.
- Nur online.

### Boards anlegen

- **Mein Board** entsteht beim ersten Aufruf von `/tasks` (oder beim ersten Hochladen) von selbst.
- **Gruppen-Boards** legen Admins an: in der Admin-Verwaltung „Boards“ mit Name und Kanso-Gruppe;
  dort auch umbenennen und löschen (Bestätigung; löscht alle Aufgaben darin, Git behält sie).
- Seed (`just seed`): ein Familien-Board mit ein paar erfundenen Aufgaben in allen Spalten.

## Daten

Wie im Gesamtdesign, im Git-Verlauf des Datenordners:

```
boards/<board-id>/
  board.yaml            # name, owner: {user: <sub>} | {group: <name>}, created
  tasks/<task-id>.md    # Frontmatter + Markdown-Text
```

```yaml
---
id: k3x9…
title: Kind zur Ferienbetreuung anmelden
status: waiting          # open | doing | waiting | done
urgent: true
important: true
due: 2026-11-06          # optional, Datum ohne Uhrzeit
waiting_for: Antwort Hort # optional
claimed: <sub>           # optional, nur Gruppen-Boards
note: <note-id>          # optional, Quell-Notiz
author: <sub>
created: 2026-10-30T08:12:00Z
updated: 2026-10-30T09:40:00Z
done_at: 2026-10-31T17:00:00Z  # gesetzt beim Wechsel nach done, entfernt beim Zurück
---
Per Mail an hort@… – Formular hängt an der Notiz.
```

- Das private Board hat eine zufällige ID wie alle anderen; `owner.user` macht es privat. Pro
  Person genau eins (der Store sucht es über `owner`).
- **Ein Commit pro Änderung** (anlegen, bearbeiten, ziehen, übernehmen, löschen) mit der Person als
  Autor, wie im Wiki. Nachricht z. B. `tasks: Hof fegen → done`.
- In der Notiz: Frontmatter-Feld `task: <board-id>/<task-id>`.
- „Heute“, „morgen“, „diese Woche“ rechnet das Gerät beim Erfassen in seiner Ortszeit in ein Datum
  um. „Überfällig“, „heute“ und „morgen“ auf den Karten rechnet der Server in seiner Zeitzone
  (`TZ` des Dienstes). *(meine Entscheidung)*
- Aufgabe in ein anderes Board verschieben = `git mv` der Datei, ein Commit.
- Grenzen: Betreff ≤ 200, Text ≤ 100 KB, worauf ≤ 120 Zeichen, ≤ 2000 Aufgaben pro Board.
- Hauptdesign anpassen: `assignee` und `recur` entfallen, Lanes sind fest (nicht in `board.yaml`).

## Rechte

- **Privates Board:** nur die Besitzer:in. Admins haben **keinen** Zugriff (wie Notizen, Todos).
- **Gruppen-Board:** alle Mitglieder der Gruppe sehen und ändern alle Aufgaben (anlegen, ändern,
  ziehen, übernehmen, löschen). Admins verwalten das Board (anlegen, umbenennen, löschen); sehen
  dessen Aufgaben nur, wenn sie selbst in der Gruppe sind.
- Fremde Boards und Aufgaben antworten mit 404 (nicht 403).
- Aufgabe in anderes Board verschieben: nur, wenn man beide sieht.

## Technik (vorgegeben)

- Neues Paket `internal/tasks` (Store, Board, Task, Rechte, Sortierung, Filter), Handler in
  `internal/web`, servergerenderte Seiten (`html/template`). JavaScript nur für Ziehen und für die
  Offline-Erfassung; ohne JS funktioniert alles außer Ziehen und offline.
- Service Worker (`sw.js`) und Ausgang folgen dem Muster der Notizen; eigener Code, keine neue
  Abhängigkeit.
- Ziehen: natives HTML Drag-and-Drop, schickt `POST` mit CSRF-Token; die Seite aktualisiert die
  Karte ohne Neuladen; Fehler → Karte springt zurück + Hinweis.
- Neue ADR „Aufgaben in Git, offline nur Erfassen“; arc42 (Bausteine, Laufzeit Erfassen offline)
  und `docs/design.md` (Bildschirme Board, Erfassen, Aufgabe) ergänzen.

## Dem Builder überlassen

- Endpunkte und JSON-Format für Hochladen und Ziehen (Vorschlag: `POST /api/tasks` für den
  Ausgang, `POST /tasks/b/<id>/<task>/move` mit `status` bzw. `urgent`/`important`).
- Ob das CSRF-Token für den Ausgang über `/notes/token` oder einen allgemeinen Endpunkt kommt.
- Wie die D/W-Marken und das Aufgaben-Icon genau aussehen (Rams-Regeln, Linien-Icon 1,5 px).
- Wie die Board-Reiter bei vielen Gruppen-Boards umbrechen.

## Tests

- Unit (`internal/tasks`): anlegen mit Geräte-ID (doppelt → eine Aufgabe), privates Board wird
  einmal angelegt, Rechte (fremdes privates Board → nicht gefunden, Admin → nicht gefunden,
  Gruppenmitglied → sichtbar), Sortierung, Erledigt-14-Tage-Filter, `done_at` setzen/entfernen,
  Matrix ohne Erledigte, Versionskonflikt beim Speichern, verschieben zwischen Boards, Notiz →
  Aufgabe (Betreff/Text-Aufteilung), unbekanntes Board beim Hochladen → Mein Board.
- HTTP (`internal/e2e`): Board fremder Person → 404, Ziehen ohne CSRF → abgelehnt, Admin legt
  Gruppen-Board an, Commit mit Autor.
- Playwright: am Handy erfassen (Betreff, Familie, dringend, heute) → erscheint im Board; offline
  erfassen → online → im Board (Chromium); Kanban-Ziehen ändert Spalte und Matrix zeigt dieselbe
  Aufgabe; Matrix-Ziehen setzt dringend/wichtig; übernehmen auf dem Familien-Board durch zweite
  Person; Notiz → Aufgabe.

## Rechtliches

- Keine neue Abhängigkeit, keine neue ausgehende Verbindung.
- Neu im Browser: Aufgaben-Ausgang und Board-Liste in IndexedDB; README-Datenschutz ergänzen.
- Hinweis für die README: Aufgaben stehen im Git-Verlauf; gelöscht heißt nicht spurlos (anders als
  Notizen).

## Meine Entscheidungen (zum Gegenlesen)

- Boards nur online; offline nur Erfassen.
- Kein manuelles Sortieren; Reihenfolge nach Fälligkeit, dann neueste.
- „Diese Woche“ = kommender Sonntag.
- Matrix zeigt keine erledigten Aufgaben.
- Keine Anhänge an Aufgaben; Fotos bleiben in der Notiz.
- Übernehmen ohne Sperre; jede:r darf eine übernommene Aufgabe neu übernehmen.
- Unbekanntes Board beim Hochladen → Mein Board.
- Gruppen-Boards legen nur Admins an; Admins sehen deren Aufgaben nur als Gruppenmitglied.
- Ein Commit pro Änderung, auch beim Ziehen.
- Aufgaben nicht in der Suche.
