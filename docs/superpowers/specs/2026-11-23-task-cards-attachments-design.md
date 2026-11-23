# Aufgaben: Karten, Etiketten, Avatar, Anhänge

**Status:** freigegeben (Owner, 2026-11-23)

## Warum

Der Owner hat eine Notiz mit Foto in eine Aufgabe umgewandelt, und das Foto
fehlte: Aufgaben haben bisher keine Anhänge (Spec 2026-10-30, „Fotos bleiben
in der Notiz“). Das Board wirkt außerdem wie „umrahmte Überschriften“ statt
Karten. `D`/`W` sagen nicht, was sie bedeuten. Und der Avatar aus 0.11.3
taucht bei seinen Daten nie auf, weil er nur bei übernommenen Aufgaben auf
Gruppen-Boards erscheint.

## Ziele

1. **Karten**, die wie Kanban-Karten aussehen. Die ganze Karte ist ein Link,
   der Titel ist nicht unterstrichen. Oben stehen die Etiketten, dann der
   Titel. Unten ist eine Fußzeile: Fälligkeitstermin bzw. „wartet auf …“
   links, Avatar rechts. Mehr Innenabstand, Radius 4 px, Fläche `--bg` auf
   der Spalte (`--bg-2`). Beim Überfahren wechselt nur der Hintergrund. Ein
   Schatten nur beim Ziehen (schwebende Ebene).
2. **Etiketten** „dringend“ (Rot = braucht Aufmerksamkeit, Text in
   `--warning-ink` auf zarter Fläche) und „wichtig“ (neutral kräftig). Es
   sind Wörter, also funktioniert es auch in Graustufen. Sie ersetzen `D`/`W`.
3. **Avatar** auf jeder Karte und auf der Aufgabenseite: die Person, die
   übernommen hat, sonst der Autor. Das gilt auf allen Boards, auch auf
   privaten. Übernehmen und Freigeben bleiben unverändert.
4. **Anhänge an Aufgaben** wie im Wiki: Hochladen im Bearbeiten-Formular per
   Knopf, Einfügen und Ziehen. Markdown-Link `attachments/<name>` relativ zur
   Aufgabe.
   - Ablage: `boards/<board>/attachments/<aufgabe>/<name>`, im Git-Verlauf.
   - Ausliefern: `GET /tasks/b/{board}/{task}/attachments/{name}`, nur für
     wer das Board sieht.
   - Verschieben auf ein anderes Board verschiebt den Ordner mit. Löschen
     entfernt ihn, der Verlauf behält ihn.
   - **Notiz → Aufgabe:** Die Fotos der Notiz werden in die Aufgabe
     *kopiert*, die Bildzeilen bleiben im Text. Die Notiz bleibt unverändert.
5. Offline-Erfassen bleibt ohne Fotos.

## Nicht-Ziele

Zuweisen an andere, Anhänge löschen aus der UI (wie bei Notizen: Link aus dem
Text nehmen genügt vorerst; aufgeräumt wird über die Berichte „Verwaiste
Medien“, sofern die Aufgaben-Anhänge dort erfasst werden, sonst
Backlog), Vorschaubilder auf Karten.

## Ansätze

- **Anhänge pro Aufgabe in einem Ordner neben dem Board (gewählt).**
  `tasks/<id>.md` bleibt wie es ist, keine Migration.
- *Aufgabe als Ordner `tasks/<id>/index.md` + `attachments/`:* Das wäre
  sauberer, braucht aber eine Migration aller Aufgaben und Änderungen an
  jedem Lesepfad. Verworfen.
- *Fotos in der Notiz lassen und nur verlinken:* Das ist der heutige Stand.
  Andere Board-Mitglieder sehen die private Notiz nicht. Verworfen.

## Entscheidungen des Builders

Die Größengrenze übernimmt `maxAttachment` aus dem Wiki. Die Commit-Texte
folgen den Aufgaben (`tasks: „Titel“ Anhang …`). Das ADR 019 kehrt die
Anhang-Entscheidung aus Spec 2026-10-30 um.
