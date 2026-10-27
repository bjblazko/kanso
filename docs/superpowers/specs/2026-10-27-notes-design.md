# Hüpattl! Kanso – Notizen (Phase 2)

**Status:** freigegeben · **Created:** 2026-10-27 · **Updated:** 2026-10-27

Baut auf dem Gesamtdesign auf: [2026-10-26-kanso-design.md](2026-10-26-kanso-design.md).

## Warum

Unterwegs fällt etwas ein, und es landet heute irgendwo: Notiz-App, Papier, VS Code am Desktop. Das
Einsammeln ist mühsam. Kanso soll **die eine Stelle** werden, an der alles Unsortierte landet, und am
Handy so schnell sein, dass sich der Griff zu etwas anderem nicht lohnt. Sortiert wird später, von Hand.

## Ziele (vom Nutzer)

- **Schnell erfassen am Handy, unstrukturiert:** kein Titel, kein Bereich, keine Ordner.
- **Offline schon in dieser Phase:** neue Notizen ohne Netz schreiben; sie werden hochgeladen, sobald
  wieder Netz da ist.
- **Fotos:** direkt von der Kamera oder aus der Fotomediathek, auch offline.
- **Einsortieren von Hand:** löschen, oder in ein Wiki-Dokument überführen. To-do-Listen gibt es erst
  in Phase 4.

## Nicht-Ziele

- Ältere Notizen offline lesen oder bearbeiten (vielleicht später, Nutzer: „alte ggf. später“).
- Notiz → Aufgabe (kommt mit den Tasks in Phase 4).
- Automatisches Sortieren, Tags, Ordner, Teilen von Notizen.
- Volle PWA mit Offline-Listen und Push (Phase 7); hier nur das Stück, das Notizen brauchen.

## Anti-Ziele (wäre ein Fehlschlag, auch wenn es technisch läuft)

- Erfassen braucht mehr als: Symbol antippen → tippen → Speichern.
- Eine offline geschriebene Notiz oder ein Foto geht still verloren.
- Jemand anderes als die Verfasserin sieht eine Notiz, auch kein Admin in der App.
- Ein Foto trägt den Aufnahmeort in eine Wiki-Seite, die die Familie sieht.

## Verhalten

### Erfassen (`/notes/new`)

- Eine Seite mit genau einer Aufgabe: großes Textfeld, darunter **Kamera** (öffnet die Kamera),
  **Foto wählen** (Mediathek oder Datei, mehrere auf einmal) und **Speichern** (Hauptaktion).
  Angehängte Fotos erscheinen als kleine Vorschauen mit „Entfernen“.
- Nach dem Speichern bleibt man auf der Seite, das Feld ist leer, eine ruhige Zeile sagt
  „Gespeichert“ bzw. „Gespeichert · 2 warten auf Netz“. Die nächste Notiz kann sofort folgen.
- Die erste nicht-leere Zeile ist der Titel (in Listen gekürzt). Notizen ohne Text, nur mit Fotos,
  heißen „Foto vom 27.10.2026, 14:03“.
- Kanso lässt sich auf den Homescreen legen (Web-App-Manifest). Das Symbol öffnet direkt
  `/notes/new`. Hinweis: iOS öffnet die Tastatur nicht von selbst; ein Tipp ins Feld bleibt nötig.

### Offline

- Ein Service Worker (eigener Code, keine Bibliothek) hält die Erfassungsseite und ihre Dateien
  (CSS, JS, Schrift, Icons) vor. Ohne Netz öffnet `/notes/new` trotzdem.
- **Ausgang auf dem Gerät:** Jede Notiz geht zuerst in IndexedDB (Text + Fotos als Blobs), mit einer
  auf dem Gerät erzeugten ID. Hochgeladen wird beim Speichern, beim Öffnen der Seite, beim Ereignis
  `online` und alle 30 s, solange die Seite offen ist (Safari kennt kein Background Sync).
- Erst wenn der Server die Notiz bestätigt hat, verschwindet sie aus dem Ausgang. Das Hochladen ist
  **idempotent**: Kennt der Server die ID schon, antwortet er „schon da“; nichts wird doppelt angelegt.
- Ist die Sitzung abgelaufen, bleibt alles im Ausgang, die Seite sagt „Zum Hochladen bitte anmelden“
  mit Link; nach der Anmeldung geht es weiter.
- Die Liste der wartenden Notizen ist auf der Erfassungsseite sichtbar (Anzahl, auf Tipp die Titel).

### Liste und Einsortieren (`/notes`)

- Eigene Notizen, neueste zuerst: Titel, Anfang des Textes, Zeit, Anzahl Fotos. Eintrag „Notizen“ in
  der Navigation über den Bereichen.
- Eine Notiz öffnen zeigt sie gerendert (Markdown, Fotos inline); **Bearbeiten** nutzt denselben
  Editor wie das Wiki (Quelltext/Vorschau, Anhänge).
- **Löschen** mit Bestätigungsseite (wie bei Wiki-Seiten).
- **Als Wiki-Seite anlegen:** Bereich (nur solche mit Rolle ≥ Editor) und übergeordnete Seite wählen,
  Titel vorbelegt aus der ersten Zeile und änderbar. Text und Anhänge werden zur Seite; die Notiz ist
  danach weg. Die Seite behält die ID der Notiz nicht zwingend (siehe Builder).
- Die Suche findet auch eigene Notizen (eigene Ergebnisgruppe), nie fremde.

### Fotos

- Originalgröße, kein Verkleinern (Typenschilder müssen lesbar bleiben). Grenze wie bei Anhängen:
  50 MiB pro Datei.
- **Aufnahmeort entfernen:** Bei JPEG entfernt der Server beim Speichern die GPS-Angaben aus den
  EXIF-Daten; die Ausrichtung (Orientation) bleibt, sonst stünden Fotos auf dem Kopf. Andere Formate
  bleiben unverändert (iOS liefert über das Datei-Feld in der Regel JPEG). Selbst gebaut mit der
  Go-Standardbibliothek.

## Daten

```
notes/<sub>/<note-id>/
  index.md            # Frontmatter: id, created, updated; Body = Markdown
  attachments/…       # Fotos, Dateien; im Text relativ als attachments/<name> verlinkt
```

- Neue Notiz: Fotos werden als Anhänge gespeichert und ans Ende des Textes als Bilder angefügt.
- **Nicht im Git-Verlauf (Nutzer, 2026-10-27):** `notes/` steht in der `.gitignore` des Datenordners
  (Kanso legt sie an bzw. ergänzt sie beim Start). Speichern schreibt atomar, ohne Commit; Löschen
  entfernt den Ordner endgültig. Wird eine Notiz zur Wiki-Seite, wird sie dorthin verschoben und
  dort erstmals committet („Seite aus Notiz angelegt“); ihr Verlauf beginnt dann.
- Backups des Datenordners erfassen Notizen trotzdem (sie liegen als Dateien dort).

## Rechte

- Nur die Verfasserin sieht, bearbeitet, löscht ihre Notizen. Admins in der App nicht.
- Wer Zugriff auf den Datenordner auf dem Server hat, kann sie lesen (steht im README).

## Meine Entscheidungen (zum Gegenlesen)

- Erfassen bleibt nach dem Speichern auf der Seite (schnelle Folge-Notizen).
- Titel = erste Zeile, kein eigenes Titelfeld.
- Keine Bearbeitungssperre für Notizen (nur eine Person). Gegen zwei Geräte schützt eine
  Versionsprüfung: Speichern wird abgelehnt, wenn sich die Notiz seit dem Öffnen geändert hat; der
  Text bleibt im Editor.
- GPS wird serverseitig entfernt (nicht im Browser), damit auch Uploads ohne JS geschützt sind.
- Kein neues Drittanbieter-Paket. Service Worker, IndexedDB, Manifest, EXIF: eigener Code.
- Das Homescreen-Symbol wird selbst gezeichnet (Lizenz: Apache-2.0 wie das Projekt).

## Dem Builder überlassen

- Endpunkte und JSON-Formate (Hochladen als ein Multipart-Request pro Notiz mit allen Fotos).
- Wie die Erfassungsseite ihr CSRF-Token bekommt, obwohl sie aus dem Cache kommt (z. B. frisches
  Token per GET vor dem Hochladen).
- Ob eine Notiz beim Überführen ihre ID als Seiten-ID behält (Kollisionen prüfen).
- Wie der Suchindex Notizen nach Besitzerin filtert.
- Zuschnitt des Service-Worker-Caches und seine Versionierung (am Asset-Fingerprint).

## Tests

- Unit: Notiz-Speicher (anlegen, idempotent per ID, Rechte), Titel aus erster Zeile, EXIF-GPS-Entfernen
  (JPEG mit GPS + Orientation → ohne GPS, mit Orientation; kaputte Dateien bleiben unverändert),
  Überführen ins Wiki.
- HTTP: fremde Notizen → 404, Admin → 404, Hochladen zweimal → eine Notiz.
- Playwright: Erfassen online; Erfassen offline (`context.setOffline`) → wartet → online → in der
  Liste; Foto anhängen; Notiz → Wiki-Seite; Löschen. Offline-Tests mindestens in Chromium (WebKit in
  Playwright unterstützt Service Worker nur eingeschränkt).

## Rechtliches

- Keine neue Abhängigkeit, keine neue ausgehende Verbindung.
- Neu: Daten im Browser (IndexedDB, Service-Worker-Cache) bis zum Hochladen. README-Datenschutz
  ergänzen.
