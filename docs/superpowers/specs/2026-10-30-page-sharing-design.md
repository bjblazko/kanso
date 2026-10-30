# Hüpattl! Kanso – Seiten einzeln freigeben

**Status:** Entwurf · **Created:** 2026-10-30 · **Updated:** 2026-10-30

Baut auf dem Gesamtdesign auf: [2026-10-26-kanso-design.md](2026-10-26-kanso-design.md) (Rollen, Spaces)
und ADR-007 (jede Seite sagt, wer sie sieht).

## Warum

Ein Space gehört meist einer Person oder Gruppe. Manchmal soll genau eine Seite daraus jemand anderem
zugänglich sein, ohne den Rest des Spaces: „Timos Wunschliste 2026“ aus „Timos Space“ für Christina.
Bisher geht das nur über die Freigabe des ganzen Spaces. Dazu kommt: „Freigabe ändern“ auf einer Seite
öffnet heute die Space-Einstellungen, obwohl man an die Seite denkt.

## Ziele (vom Nutzer)

- Eine Wiki-Seite lässt sich **einzeln** an Personen oder Kanso-Gruppen freigeben, als **Ausnahme** vom
  Space; der Rest des Spaces bleibt für diese Personen unsichtbar.
- Eine Seitenfreigabe hat eine **Rolle** wie bei Spaces: lesen, bearbeiten, verwalten.
- **Unterseiten erben** die Freigabe; beim Freigeben steht ein Hinweis darauf.
- Wer etwas einzeln freigegeben bekommt, findet es an einem Ort **„Für dich freigegeben“**.

## Nicht-Ziele

- Rechte entziehen: Eine Seitenfreigabe gibt nur Rechte dazu, sie nimmt nie welche weg. Wer den Space
  lesen darf, liest auch die Seite. *(meine Entscheidung)*
- Freigaben über Links ohne Anmeldung (öffentliche Links).
- Seiten in andere Spaces verschieben oder kopieren (eigenes Projekt, danach).
- Ein gemeinsamer „Für dich freigegeben“-Ort über alle Teile: Todo-Listen zeigen geteilte Listen schon
  in ihrer Übersicht („von Anna“). *(meine Entscheidung, Nutzer hat sie mir überlassen)*

## Anti-Ziele

- Wer nur eine Seite freigegeben hat, erfährt etwas über den Rest des Spaces: andere Seitentitel,
  den Seitenbaum, Treffer in der Suche, Anhänge oder den Verlauf anderer Seiten.
- „Freigabe ändern“ auf einer Seite ändert stillschweigend den ganzen Space.

## Verhalten

### Rechte

- Die Rolle einer Person auf einer Seite ist die **höchste** aus: ihrer Space-Rolle und jeder
  Seitenfreigabe auf dieser Seite **oder einer ihrer Oberseiten**. Admins bleiben Owner überall.
- Rollen: **Lesen** (viewer), **Bearbeiten** (editor: Text, Anhänge, Unterseiten anlegen, verschieben
  innerhalb dessen, was sie bearbeiten darf, löschen), **Verwalten** (owner: zusätzlich die Freigaben
  dieser Seite ändern und Sperren brechen). Gleiche Bedeutung wie bei Spaces.
- Freigaben einer Seite ändern dürfen: Space-Owner, Admins, und wer auf der Seite „Verwalten“ hat.
- Alles, was heute „darf den Space lesen“ prüft, prüft künftig „darf diese Seite lesen“: Seite,
  Anhänge, Verlauf und Vergleich, Vorschau, Suche, Seitenbaum, Verschieben-Ziele, Notiz → Wiki-Seite.

### Freigabe-Seite (`/p/<id>/share`)

- Aufbau wie die Mitglieder in den Space-Einstellungen: Liste der Freigaben (Person oder Gruppe, Rolle,
  Entfernen), darunter ein Formular Art · Wer · Rolle · Hinzufügen.
- Oben ein Hinweis, wenn die Seite Unterseiten hat: „Gilt auch für 3 Unterseiten.“ (Anzahl aller
  Nachfahren.)
- Darunter, nur lesend: „Über den Space sehen sie außerdem: …“ mit Link zu den Space-Einstellungen für
  Space-Owner. Und geerbte Freigaben von Oberseiten: „Von „Wunschlisten“ geerbt: Christina (Lesen)“ mit
  Link zu dieser Oberseite.
- Personen erscheinen erst nach ihrer ersten Anmeldung (wie bei Spaces); Gruppen immer.

### Sichtbarkeitszeile (ADR-007) und Link

- Auf einer Seite nennt die Zeile alle, die sie lesen dürfen, wie bisher über den Space, plus die über
  Seitenfreigaben (eigene und geerbte), ohne Doppelte: „Sichtbar für dich, Christina und Admins“.
- „Freigabe ändern“ führt auf einer Seite zur **Freigabe dieser Seite**, für alle, die sie ändern
  dürfen. Auf der Space-Startseite bleibt es die Space-Freigabe. *(behebt die Verwechslung)*

### „Für dich freigegeben“

- Neuer Abschnitt in der Wiki-Navigation unter den Wiki-Bereichen, nur sichtbar, wenn es etwas gibt.
- Er zeigt jede Seite, die dir über eine Seitenfreigabe zugänglich ist und deren Space du **nicht**
  ohnehin lesen darfst, als Baum mit ihren Unterseiten; je Eintrag klein der Space-Name und
  „von <Space-Owner>“ *(meine Entscheidung: Name der ersten Owner-Person des Spaces; Gruppe, wenn es
  nur Gruppen gibt; ohne Owner entfällt „von …“)*.
- Die Wiki-Startseite listet dieselben Seiten unter der Überschrift „Für dich freigegeben“.
- Öffnest du eine solche Seite, zeigt die Navigation nicht den Space-Baum, sondern diesen Abschnitt mit
  der aktuellen Seite markiert; die Brotkrumen beginnen bei der obersten Seite, die du sehen darfst
  (nicht beim Space).

### Bearbeiten durch Freigabe-Empfänger

- Wer Bearbeiten hat, legt Unterseiten unter freigegebenen Seiten an (sie erben) und verschiebt Seiten
  nur innerhalb des freigegebenen Teilbaums. Ziele außerhalb werden nicht angeboten.
- Notiz → Wiki-Seite bietet freigegebene Seiten mit Bearbeiten-Recht als Ort an.

## Daten

- Freigaben stehen im Frontmatter der Seite (`index.md`), im Git-Verlauf:

  ```yaml
  members:
    - {user: christina-sub, role: viewer}
    - {group: grosseltern, role: editor}
  ```

  Gleiches Format wie `access.Member` in `space.yaml`. Beim Verschieben einer Seite wandern ihre
  Freigaben mit; eine verschobene Unterseite verliert, was sie von der alten Oberseite geerbt hat.
- Jede Änderung ein Commit mit der Person als Autor: `Freigabe von „<Titel>“ geändert`.

## Technik (vorgegeben)

- `internal/access`: eine Funktion, die aus Space-Mitgliedern und den Mitgliedern der Seite samt
  Oberseiten die Rolle bestimmt (höchste gewinnt; Admin = Owner).
- `internal/wiki` hält die Mitglieder pro Seite im Index (wie Titel und Reihenfolge), damit
  Navigation und Suche ohne Dateizugriffe pro Anfrage auskommen.
- `pageFor` und die Suche benutzen die neue Rolle; der Seitenbaum eines Spaces wird für Personen ohne
  Space-Recht auf ihre freigegebenen Teilbäume beschnitten.
- Keine neue Abhängigkeit, keine neue ausgehende Verbindung.

## Dem Builder überlassen

- Genaue Darstellung der geerbten Freigaben auf der Freigabe-Seite.
- Ob „Für dich freigegeben“ auf der Startseite als Liste oder Kacheln erscheint (Rams-Regeln).
- Reihenfolge der Einträge (Vorschlag: Space-Name, dann Seitentitel).

## Tests

- Unit (`access`, `wiki`): höchste Rolle aus Space und Seiten-Kette; Gruppe; Admin; Vererbung an
  Unterseiten; verschobene Unterseite verliert Geerbtes; Freigabe wandert mit.
- HTTP: Christina mit Lesen auf einer Seite → Seite 200, Unterseite 200, Nachbarseite 404, Space-Seite
  404, Anhang der Seite 200, Anhang der Nachbarseite 404, Verlauf 200/404 entsprechend, Suche findet nur
  die freigegebenen Seiten, Bearbeiten 403; mit Bearbeiten → speichern geht, Verschieben-Ziele nur im
  Teilbaum; Freigabe ändern nur für Space-Owner, Admins und „Verwalten“; Sichtbarkeitszeile nennt
  Christina; „Freigabe ändern“ auf der Seite → `/p/<id>/share`.
- Playwright: Timo gibt „Wunschliste“ für Christina frei (Hinweis auf Unterseiten sichtbar); Christina
  findet sie unter „Für dich freigegeben“, sieht die Unterseite, nicht aber andere Seiten oder den Space.

## Meine Entscheidungen (zum Gegenlesen)

- Seitenfreigaben geben nur Rechte dazu, nehmen nie welche weg.
- Drei Rollen wie bei Spaces; „Verwalten“ darf die Freigabe der Seite ändern.
- „Für dich freigegeben“ als Abschnitt der Wiki-Navigation und auf der Wiki-Startseite, kein Eintrag in
  der Leiste; geteilte Todo-Listen bleiben in der Todo-Übersicht.
- „von …“ nennt die erste Owner-Person des Spaces.
- Freigaben im Frontmatter der Seite, damit sie beim Verschieben mitwandern und im Verlauf stehen.
- Wer bearbeiten darf, verschiebt nur innerhalb des freigegebenen Teilbaums.
- Wer über eine Seitenfreigabe bearbeiten darf, darf die Seite auch löschen, wie Editoren eines Spaces.
