# Hüpattl! Kanso – Rezepte

**Status:** freigegeben · **Created:** 2026-11-06 · **Updated:** 2026-11-06

Baut auf dem Gesamtdesign auf: [2026-10-26-kanso-design.md](2026-10-26-kanso-design.md) und dem
Wiki (Phase 1). Erster von drei aufeinander aufbauenden Teilen:

1. **Rezepte** (dieses Dokument): Rezept als Wiki-Seite mit Konvention, Zutaten-Erkennung,
   Rezeptübersicht, Kochansicht. Optional, als eigene Stufe: KI-Erfassung.
2. **Einkaufsliste** (Phase 5, eigene Spec): eigener Listentyp; „Rezept → Einkaufsliste“.
3. **Kochplan** (Phase 8, eigene Spec): Rezepte Tagen zuweisen, Tage wählen → Einkaufsliste.

Teil 1 kommt vor Phase 5 und braucht weder Listen noch Kalender.

## Warum

Rezepte stecken heute in Instagram-Videos, die man trotz Abo nicht wiederfindet, und in denen man
beim Kochen am Handy ständig vor- und zurückspult, weil die Schritte so schnell sind. Oder sie
kommen aus dem Netz. Die Einkaufsliste wird jedes Mal von Hand abgeschrieben. Ziel: Rezepte einmal
sauber ins Wiki übertragen, schnell wiederfinden, am Herd bequem lesen, und – in Teil 2 und 3 –
ohne Abtippen eine Einkaufsliste daraus füllen.

## Ziele (vom Nutzer)

- Rezepte sammeln im **Wiki**; manuell übertragen ist in Ordnung.
- **Mittelweg** zwischen freier Seite und Formular: normale Wiki-Seite, die einer Konvention folgt,
  sodass Kanso Zutaten, Mengen und Portionen versteht.
- Mengen sollen später auf der Einkaufsliste **zusammengerechnet** werden (Teil 2); dafür muss Teil 1
  Menge, Einheit und Name verlässlich erkennen.
- **Wiederfinden per Suche oder Tags** („mit Kartoffeln“, „kinderessen“, „deftig“, „süß“).
- Alles **statisch und deterministisch**, ohne KI benutzbar – Kanso soll als eigenes Projekt auf
  GitHub veröffentlicht werden können.
- **Optional** eine KI über eine **OpenAI-kompatible Schnittstelle** für „advanced stuff“; im eigenen
  Fall gegen den Rechner aus homelab2 mit gutem System-Prompt.

## Nicht-Ziele

- Import von einer URL (Webseite laden, JSON-LD `Recipe` lesen). Später denkbar, braucht
  Legal-Check wegen Abrufen vom Server. *(meine Entscheidung)*
- Vorrat/„meist im Haus“: Der Nutzer hat Variante A gewählt (alles vorgeschlagen, man hakt ab).
  Betrifft Teil 2, hier nur festgehalten.
- Nährwerte berechnen, Einheiten zwischen Masse und Volumen umrechnen (g ↔ EL wäre geraten).
- Bewertungen, Kommentare, Kochhistorie.
- Kochansicht offline. Kommt mit Phase 7 (PWA). *(meine Entscheidung)*
- Ein eigener Speicherort für Rezepte: Rezepte sind Wiki-Seiten, in beliebigen Spaces.

## Anti-Ziele

- Kanso verwirft oder verändert Text, den es nicht versteht. Eine Zeile, die der Parser nicht lesen
  kann, bleibt Wort für Wort stehen und ist trotzdem eine Zutat (ohne Menge).
- Ein Rezept ist nur mit Kanso lesbar. Die Datei bleibt gewöhnliches Markdown, das in jedem Editor
  und auf GitHub gut aussieht.
- Kanso braucht eine KI oder einen fremden Dienst, um Rezepte zu nutzen.
- Text geht unbemerkt an eine KI.
- Die Kochansicht zwingt am Herd zum Scrollen durch Fremdes (Bearbeitungsleiste, Navigation),
  oder der Bildschirm geht mitten im Schritt aus.

## Das Rezeptformat

Ein Rezept ist eine Wiki-Seite (`spaces/<space>/pages/<slug>/index.md`). Zusätzliche Felder im
Frontmatter; alle optional außer `type`:

```markdown
---
id: k3x9…            # wie jede Seite
title: Kartoffelcurry
type: recipe
servings: 4          # ganze Zahl ≥ 1; fehlt sie, gilt 1 und die Kochansicht sagt „ohne Portionsangabe“
tags: [deftig, vegetarisch, kinderessen]
source: https://beispiel.de/kartoffelcurry   # oder Freitext „Instagram @…“
time: 40             # Minuten gesamt, optional
---

Kurz und mild, die Kinder essen es.

## Zutaten

- 800 g Kartoffeln, festkochend
- 2 Zwiebeln
- 1 Dose Kokosmilch
- 1,5 EL Currypaste
- ½ TL Salz
- Koriander zum Bestreuen

### Für den Reis

- 300 g Basmatireis

## Zubereitung

1. Kartoffeln schälen und würfeln.
2. Zwiebeln anschwitzen, Currypaste dazu.
3. …

---

Quelle: [beispiel.de](https://beispiel.de/kartoffelcurry). Für den privaten Gebrauch übertragen.
```

Regeln:

- **Abschnitte** werden über Überschriften der Ebene 2 gefunden: `## Zutaten` (auch `Ingredients`)
  und `## Zubereitung` (auch `Schritte`, `Steps`, `Method`). Groß/klein egal. Ebene-3-Überschriften
  innerhalb der Zutaten sind **Gruppen** („Für den Reis“). Alles andere ist freier Text.
- **Zutatenzeile** = ein Listenpunkt in `## Zutaten`:
  `[Menge] [Einheit] Name[, Zusatz]`.
  - Menge: `2`, `1,5`, `1.5`, `1/2`, `½` (Unicode-Brüche ¼ ½ ¾ ⅓ ⅔), `1 ½`, Spanne `2-3` / `2–3`.
    Fett gesetzte Mengen (`- **200 g** Mehl`, Format des Skills `kanso-rezept`) werden ebenso
    gelesen.
  - Einheit aus einer festen Liste (unten); Einheit ohne Leerzeichen (`200g`) geht auch.
  - Name: der Rest bis zum ersten Komma; danach **Zusatz** („festkochend“, „zum Bestreuen“), der
    angezeigt, aber nicht zum Zusammenrechnen benutzt wird. Ein Klammerzusatz am Ende ebenso.
  - Keine erkennbare Menge → Zutat ohne Menge („Koriander zum Bestreuen“, „Salz“). *(Hinweis
    für Teil 2: kommt auf die Liste ohne Mengenangabe.)*
- **Einheiten** (DE/EN-Schreibweisen, Groß/klein egal, mit/ohne Punkt):
  - Masse: `g`, `kg` · Volumen: `ml`, `cl`, `l` – innerhalb einer Familie umrechenbar.
  - Zähleinheiten, nur mit sich selbst addierbar: `EL`/`tbsp`, `TL`/`tsp`, `Prise`, `Stück`/`Stk`/`pc`,
    `Bund`, `Dose`, `Packung`/`Pck`, `Becher`, `Zehe`, `Scheibe`, `Tasse`/`cup`, `Glas`, `Zweig`,
    `Handvoll`.
  - Keine Einheit, aber Menge (`2 Zwiebeln`) → Einheit „Stück“ implizit.
- **Spanne** `2-3`: angezeigt wie geschrieben, gerechnet wird mit dem oberen Wert. *(meine
  Entscheidung – lieber eine Zwiebel zu viel im Haus.)*
- **Schlüssel zum Zusammenrechnen** (für Teil 2, hier schon erzeugt): Name klein, ohne Zusatz,
  Leerraum normalisiert. Singular/Plural („Zwiebel“/„Zwiebeln“) wird in Teil 1 **nicht**
  zusammengeführt; das entscheidet Teil 2. *(meine Entscheidung)*

## Verhalten

### Paket und Daten

- Neues Paket `internal/recipes`: Parsen (Markdown-Body → `Recipe{Servings, Groups[]{Name,
  Ingredients[]}, Steps[]}`), Skalieren, Formatieren von Mengen (DE: Komma, EN: Punkt). Reine
  Funktionen, kein Dateizugriff; das Wiki liefert die Seite.
- `wiki.PageMeta` bekommt `Type`, `Servings`, `Tags`, `Source`, `Time` (alle `omitempty`), damit sie
  beim Speichern erhalten bleiben. Gewöhnliche Seiten ändern sich dadurch nicht.
- Parser liest Markdown zeilenweise, nicht über den goldmark-AST. *(meine Entscheidung, der Builder
  darf goldmark nehmen, wenn es kürzer wird.)*

### Rezept anlegen und bearbeiten

- **„Neues Rezept“** (Hauptaktion der Rezeptübersicht): wählt Space und optional Elternseite wie
  „Neue Seite“, legt die Seite mit `type: recipe` und dem Gerüst oben an (Kurzbeschreibung,
  `## Zutaten`, `## Zubereitung`, Platzhalterzeilen) und öffnet den Editor.
- Im **Seiteneditor** einer Rezeptseite stehen über dem Text die Felder **Portionen**, **Tags**
  (kommagetrennt, beim Speichern klein geschrieben, getrimmt, doppelte entfernt), **Quelle**, **Zeit
  (Min.)**. Für jede Seite gibt es im Editor ein Häkchen **„Ist ein Rezept“**, mit dem sich eine
  bestehende Seite umwandeln lässt (setzt/entfernt nur `type`, ändert den Text nicht).
- **Rückmeldung zur Erkennung:** Die Vorschau im Editor zeigt bei Rezepten unter den Zutaten, wie
  Kanso sie verstanden hat (Menge · Einheit · Name), und markiert Zeilen ohne erkannte Menge dezent.
  Speichern wird nie blockiert.

### Rezeptseite (Lesen)

- Normale Seitenansicht, gerendert wie jede Wiki-Seite. Zusätzlich unter dem Titel: Tags (als Links
  in die gefilterte Übersicht), Portionen, Zeit, Quelle.
- In der Seiten-Werkzeugleiste: **„Kochen“** als zusätzliche ruhige Aktion (Bearbeiten bleibt
  Akzent).

### Kochansicht (`/recipes/{id}/cook`)

- Eigener Bildschirm fürs Handy am Herd, nach dem Kanso-Bildschirmkopf (sticky Leiste + Titel):
  Leiste mit „Zurück zur Seite“ und dem **Portionen-Wähler** (− Zahl +).
- Darunter Zutaten (nach Gruppen) mit umgerechneten Mengen, jede zum Abhaken; dann die Schritte,
  groß und mit viel Abstand, jeder zum Abhaken. Freier Text des Rezepts erscheint hier nicht.
- Abgehakte Zutaten/Schritte und gewählte Portionen sind Gerätezustand (localStorage, try/catch,
  pro Rezept), kein gespeicherter Inhalt.
- **Bildschirm bleibt an** über die Screen Wake Lock API, solange die Kochansicht offen und sichtbar
  ist; ohne Unterstützung still weglassen.
- Mengen-Umrechnung: Faktor = gewählte / `servings`. Darstellung auf sinnvolle Genauigkeit:
  g/ml ab 100 auf 5 gerundet, darunter ganz; kg/l auf 2 Nachkommastellen ohne Nullen; Zähleinheiten
  als Bruch auf ¼ genau (`1½ Dosen`). Zutaten ohne Menge bleiben unverändert. *(meine Entscheidung)*
- Nur Lesen erfordert nur Leserecht auf die Seite.

### Rezeptübersicht (`/recipes`)

- Neuer Eintrag **„Küche“** in der Leiste der Teile (ADR-005), nach dem Wiki, mit eigenem
  Linien-Icon im Kanso-Stil; öffnet die Rezeptübersicht. Der Kochplan (Teil 3) kommt später unter
  denselben Eintrag, nicht als eigener (Nutzer, 2026-11-06). Modul-Name `kitchen`.
- Zeigt alle Rezeptseiten aller Spaces, die die Person lesen darf (dieselbe Rechteprüfung wie Suche
  und Seitenansicht), alphabetisch, je Zeile: Titel, Tags, Zeit, Space.
- Oben ein **Suchfeld** (Titel, Tags und Zutatennamen, Präfix-Treffer wie die bestehende Suche) und
  die **Tags** aller sichtbaren Rezepte als Filter zum An-/Abwählen (mehrere = alle müssen passen).
  Filter stehen in der URL (`/recipes?tag=deftig&q=kartoffel`), damit sie teilbar und mit Zurück
  bedienbar sind; funktioniert ohne JS als Formular.
- „mit Kartoffeln“ braucht kein Tag: Die Suche trifft erkannte Zutatennamen.
- Leerer Zustand: kurzer Satz, was ein Rezept ist, und „Neues Rezept“.
- Die globale Suche findet Rezepte wie bisher als Seiten; Tags kommen mit in den Suchindex.

### Sprache

Alle neuen Texte in `internal/i18n/de.yaml` und `en.yaml`. Der Parser erkennt deutsche und
englische Abschnittsnamen und Einheiten unabhängig von der Sprache der Person.

## Stufe 2: KI-Erfassung (optional)

*Ersetzt durch [KI-Anbindung und Rezept von Webseite](2026-11-07-ai-recipe-import-design.md).*

Wird erst nach Teil 1 gebaut und nach einem **huepattl-legal-check** (Inhalte verlassen den
Server).

- **Konfiguration** (aus, wenn nicht gesetzt): `KANSO_AI_URL` (Basis einer OpenAI-kompatiblen
  Schnittstelle, z. B. `http://pc.lan:8080/v1`), `KANSO_AI_MODEL`, `KANSO_AI_KEY_FILE` (optional).
  Aufruf `POST {url}/chat/completions` mit Go-Standardbibliothek, keine neue Abhängigkeit,
  Zeitlimit, kein Streaming nötig.
- **Erster Fall: Text → Rezept.** Bei „Neues Rezept“ ein zweiter Weg **„Aus Text erstellen“**: Feld
  zum Einfügen (Instagram-Beschreibung, Text einer Webseite). Kanso schickt den Text mit festem
  System-Prompt (im Repo, beschreibt genau das Format oben) an die KI und legt das Ergebnis als
  **Entwurf im Editor** an – gespeichert wird erst, wenn die Person speichert. Das Ergebnis läuft
  durch denselben Parser; die Erkennungs-Rückmeldung zeigt, was nicht passt.
- Der Weg ist nur sichtbar, wenn die KI konfiguriert ist; der Knopf sagt, wohin der Text geht
  („Text geht an: pc.lan“).
- Weitere Fälle (Zutatennamen vereinheitlichen, Tags vorschlagen) sind nicht Teil dieser Spec.

## Tests

- Unit: Parser mit Tabelle aller Mengen-, Einheiten- und Abschnittsformen, unerkannte Zeilen,
  Gruppen, fett gesetzte Mengen, Skill-Format; Skalieren und Runden; Tags normalisieren;
  `PageMeta` behält neue Felder beim Lesen/Schreiben, alte Seiten unverändert.
- Go-E2E (`internal/e2e`): Rezept anlegen, Felder speichern, Übersicht filtert nach Tag und Zutat,
  Rechte (Rezept in fremdem Space unsichtbar), Kochansicht rendert umgerechnet.
- Playwright (`e2e/`): Neues Rezept → speichern → in Übersicht finden → Kochen → Portionen ändern
  → abhaken.
- Stufe 2: Fake-Server im Test für `/chat/completions`; aus, wenn nicht konfiguriert.

## Dokumentation

Feature-Doc `docs/features/2026-11-06 recipes.md`, ADR „Rezepte sind Wiki-Seiten mit Konvention“
(Alternativen: eigener Datentyp/Formular, freie Seiten ohne Struktur), ADR für die KI-Anbindung in
Stufe 2, arc42 (Baustein `internal/recipes`), `docs/design.md` (Bildschirme Rezeptübersicht und
Kochansicht), CHANGELOG, Gesamtdesign (Phasenliste: Rezepte vor Phase 5).

## Dem Builder überlassen

Genaue Aufteilung der Handler, Icon-Zeichnung (im bestehenden Stil), Wortlaut der Texte, ob der
Parser zeilenweise oder per goldmark arbeitet, Aufbau des Suchindex für Tags/Zutaten.
