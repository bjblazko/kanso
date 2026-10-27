# Hüpattl! Kanso – Todo-Listen

**Status:** freigegeben · **Created:** 2026-10-27 · **Updated:** 2026-10-27

Baut auf dem Gesamtdesign auf: [2026-10-26-kanso-design.md](2026-10-26-kanso-design.md). Ergänzt die
Notizen ([2026-10-27-notes-design.md](2026-10-27-notes-design.md)) um einen zweiten Teil fürs Handy.

## Warum

Kleine Vorhaben wie „Gartenarbeit“ oder „Weihnachtsvorbereitungen“ brauchen eine schlichte Liste zum
Abhaken, am Handy, auch ohne Netz. Ein Kanban-Board (Phase 4) ist dafür zu schwer, ein Häkchen in
einer Wiki-Seite (Git-Commit pro Klick) zu umständlich. Todo-Listen sind deshalb **ein eigenes Ding**,
neben den Boards.

## Ziele (vom Nutzer)

- Einfache Todo-Listen, mobil zuerst, ad hoc wie Notizen.
- **Offline abhaken** (und bearbeiten); später landet alles dauerhaft auf dem Server.
- **Kein Git:** Listen stehen nicht im Verlauf.
- Abgehakte Punkte **bleiben an ihrer Stelle**, nichts wird umsortiert oder versteckt; sie werden nur
  etwas blasser (Deckkraft).
- Hauptpunkte mit **Unterpunkten** (eine Ebene), z. B. „Laub wegfegen“ → „vor der Haustür“, „Garten“.
- Listen sind meist persönlich, **können aber geteilt werden**.
- Die Übersicht ist **nach Aktualität sortiert**, zuletzt bearbeitete Liste oben. Abhaken zählt als
  Bearbeitung.
- Listen sind **flach**: keine Ordner, keine Unterlisten.
- Erledigte Listen bleiben stehen, bis man sie **von Hand löscht**.
- Ein Kanso-Symbol auf dem Startbildschirm für alles; es startet weiter in den Notizen.

## Nicht-Ziele

- Erwähnungen, Zuständigkeiten, Fälligkeiten, Erinnerungen, Push.
- Punkte umsortieren (Ziehen) – vorerst nicht; neue Punkte kommen ans Ende bzw. ans Ende ihres
  Hauptpunkts. *(meine Entscheidung)*
- Suche über Todo-Listen. *(meine Entscheidung; kann später kommen)*
- Mehr als eine Ebene Unterpunkte.
- Nur-Lesen-Freigaben.
- Verlauf, Wiederherstellen gelöschter Listen.
- Funktion ohne JavaScript (Todos sind eine Web-App-Seite; ohne JS ein Hinweis). *(meine Entscheidung)*

## Anti-Ziele

- Ein offline gesetztes Häkchen oder ein offline geschriebener Punkt geht still verloren.
- Abhaken verschiebt Punkte oder blendet sie aus.
- Jemand, mit dem die Liste nicht geteilt ist, sieht sie – auch kein Admin.
- Am Handy braucht ein neuer Punkt mehr als: Liste öffnen → tippen → Enter.

## Verhalten

### Übersicht (`/todos`)

- Eintrag **„Todos“** in der Leiste der Teile (ADR-005), zwischen Notizen und Wiki.
- Alle Listen, die ich sehe (eigene und mit mir geteilte), gemischt, **zuletzt bearbeitete zuerst**.
  Pro Zeile: Name, „3 von 7 erledigt“; bei fremden Listen „von Anna“, bei eigenen geteilten
  „geteilt mit Familie, Ben“.
- Oben ein Feld „Neue Liste“ + Knopf **Anlegen** (Hauptaktion); die neue Liste öffnet sich sofort.
  Geht auch offline.
- Leerer Zustand: „Noch keine Listen.“

### Eine Liste (`/todos/<id>`)

- Überschrift = Name. Darunter die Punkte: Kästchen + Text. Unterpunkte eingerückt.
- **Antippen des Kästchens** hakt ab bzw. wieder auf. Abgehakt: Kästchen gefüllt, ganze Zeile
  mit verringerter Deckkraft; kein Durchstreichen, kein Verschieben.
- Ein Hauptpunkt wird **nicht** automatisch abgehakt, wenn alle Unterpunkte erledigt sind.
- **Neuer Punkt:** Feld am Ende („Neuer Punkt“), Enter fügt hinzu und lässt das Feld offen für den
  nächsten.
- **Punkt bearbeiten:** Antippen des Textes macht ihn zum Eingabefeld; darunter erscheinen
  **Einrücken/Ausrücken**, **Unterpunkt hinzufügen** und **Löschen**. Enter oder Verlassen speichert.
  Einen Punkt leer speichern löscht ihn.
- Einrücken geht nur, wenn ein Hauptpunkt davor steht. Ein Hauptpunkt, der eingerückt wird, nimmt
  seine Unterpunkte nicht mit, sie hängen dann am neuen Hauptpunkt (eine Ebene bleibt eine Ebene).
- Einen **Hauptpunkt löschen** löscht seine Unterpunkte mit. *(meine Entscheidung)*
- **Besitzer:in** sieht zusätzlich **Umbenennen**, **Teilen …** und **Liste löschen …**.
- Statuszeile wie bei Notizen: „3 Änderungen warten auf Netz“, ggf. „Zum Abgleichen bitte anmelden“.

### Teilen (`/todos/<id>/share`, nur online, nur Besitzer:in)

- Personen oder Kanso-Gruppen hinzufügen bzw. entfernen (wie die Mitglieder eines Wiki-Bereichs,
  nur ohne Rolle).
- Wer eine Liste sieht, darf: abhaken, Punkte anlegen, ändern, einrücken, löschen.
- Nur Besitzer:in darf: umbenennen, teilen, die Liste löschen.
- Wird man aus einer Freigabe entfernt, verschwindet die Liste beim nächsten Abgleich vom Gerät;
  noch wartende Änderungen daran werden verworfen.

### Liste löschen (`/todos/<id>/delete`, nur online, nur Besitzer:in)

- Bestätigungsseite wie bei Notizen; danach ist die Liste endgültig weg (für alle).

### Offline und Abgleich

- Die Todo-Seiten werden im Browser aus einer **Kopie auf dem Gerät** gezeichnet (IndexedDB, eigene
  Datenbank `kanso-todos`, getrennt von den Notizen). Der Service Worker hält die Seite `/todos` vor
  und liefert sie offline auch für `/todos/<id>`.
- Jede Änderung ist eine **Operation** mit einer auf dem Gerät erzeugten ID und Uhrzeit (z. B.
  „Punkt X abhaken“, „Punkt Y anlegen nach X“). Sie wird sofort auf der Kopie ausgeführt und in einen
  **Ausgang** gelegt.
- Abgleich wie bei Notizen: beim Öffnen, bei jeder Änderung, bei `online`, beim Zurückkehren in den
  Tab und alle 30 s, solange die Seite sichtbar ist. Ein Abgleich schickt alle wartenden Operationen
  in einem Request; die Antwort ist der aktuelle Stand aller sichtbaren Listen. Die Kopie wird damit
  ersetzt, noch nicht bestätigte Operationen werden darauf erneut ausgeführt.
- Eine Operation verschwindet erst aus dem Ausgang, wenn der Server sie bestätigt hat. Doppelt
  gesendete Operationen schaden nicht.
- **Konflikte:** pro Feld eines Punkts (Text, Häkchen, Einrückung) gewinnt die spätere Änderung
  (Uhrzeit des Geräts). Löschen gewinnt immer; ein gelöschter Punkt kommt nicht durch eine späte
  Änderung zurück.
- Öffnet man Todos zum ersten Mal ohne Netz, steht dort „Offline – noch keine Listen auf diesem
  Gerät.“

### Startbildschirm

- Bleibt ein Symbol, Start in `/notes/new`. Todos sind über die Leiste einen Tipp entfernt.
- Kein eigener Installieren-Knopf in dieser Phase. *(meine Entscheidung; iPhone und Firefox bieten
  ohnehin keinen)*

## Daten

```
todos/<list-id>.json
```

```json
{
  "id": "…", "name": "Gartenarbeit", "owner": "<sub>",
  "shared": [{"user": "<sub>"}, {"group": "Familie"}],
  "created": "…", "updated": "…",
  "items": [
    {"id": "…", "text": "Laub wegfegen", "done": false, "sub": false,
     "at": {"text": 1761571200000, "done": 0, "sub": 0}}
  ],
  "deleted": {"<item-id>": 1761571200000}
}
```

- Eine Datei pro Liste, flach in `todos/`. Reihenfolge der Punkte = Reihenfolge im Array. `sub` heißt
  „Unterpunkt des nächsten Hauptpunkts davor“.
- **Nicht im Git-Verlauf:** `/todos/` kommt in die `.gitignore` des Datenordners (wie Notizen).
  Schreiben atomar, ohne Commit. Backups des Datenordners erfassen die Dateien trotzdem.
- `updated` setzt der Server bei jeder angewandten Änderung (Serverzeit); danach sortiert die
  Übersicht.
- `deleted` merkt sich gelöschte Punkt-IDs (Grabsteine), damit späte Operationen sie nicht
  wiederbeleben; älter als 30 Tage wird entfernt.
- Grenzen: Name ≤ 80 Zeichen, Punkt ≤ 500 Zeichen, ≤ 1000 Punkte pro Liste, ≤ 500 Operationen pro
  Abgleich.

## Rechte

- Sehen und bearbeiten: Besitzer:in, Personen und Mitglieder von Gruppen in `shared`.
- Admins haben **keinen** Sonderzugriff (wie bei Notizen).
- Wer Zugriff auf den Datenordner des Servers hat, kann die Dateien lesen (README).

## Meine Entscheidungen (zum Gegenlesen)

- Teilen: Personen **und** Kanso-Gruppen; keine Nur-Lesen-Freigabe.
- Besitzer:in behält umbenennen, teilen, löschen.
- Übersicht mischt eigene und geteilte Listen, mit „von …“ bzw. „geteilt mit …“.
- Punkte: reiner Text, eine Ebene Unterpunkte, kein Umsortieren in dieser Phase.
- Hauptpunkt löschen löscht seine Unterpunkte.
- Konflikte: spätere Änderung gewinnt pro Feld, Löschen gewinnt immer.
- Die Seiten werden im Browser gezeichnet (für offline); Teilen und Löschen der Liste sind
  servergerenderte Seiten und nur online.
- Keine neue Abhängigkeit, eigener Code für Abgleich und Service Worker.

## Dem Builder überlassen

- Endpunkte und JSON-Formate (Vorschlag: `GET /api/todos`, `POST /api/todos/sync`).
- Ob das CSRF-Token wie bei Notizen über `/notes/token` kommt oder über einen allgemeinen Endpunkt.
- Genaue Deckkraft abgehakter Punkte (Text muss lesbar bleiben).
- Wie die Seite zwischen Übersicht und Liste wechselt (History API).

## Tests

- Unit: Operationen anwenden (anlegen nach, abhaken, Text, einrücken nur mit Hauptpunkt davor,
  löschen mit Unterpunkten, Grabsteine, späte Änderung verliert, doppelte Operation), Rechte (fremd →
  nicht gefunden, Admin → nicht gefunden, geteilt per Gruppe), Sortierung nach `updated`,
  `.gitignore`.
- HTTP: Abgleich mit fremder Liste → Operation verworfen; Teilen/Löschen nur Besitzer:in.
- Playwright: Liste anlegen, Punkte und Unterpunkt, abhaken bleibt an der Stelle und wird blasser;
  teilen mit zweiter Person, die abhakt; offline abhaken → online → beim anderen sichtbar (Chromium);
  löschen.

## Rechtliches

- Keine neue Abhängigkeit, keine neue ausgehende Verbindung.
- Neu im Browser: Todo-Kopie und Ausgang in IndexedDB. README-Datenschutz ergänzen.
