# Hüpattl! Kanso – Kochplan (Teil 3)

**Status:** freigegeben (Nutzer: „merge und weiter“, Planart 1) · **Created:** 2026-11-07 · **Updated:** 2026-11-07

Dritter Teil nach [Rezepten](2026-11-06-recipes-design.md) und
[Einkaufslisten](2026-11-07-shopping-lists-design.md).

## Warum

Donnerstags wird die nächste Woche geplant, freitags eingekauft. Gerichte sollen Tagen zugewiesen
werden; aus ausgewählten Tagen entsteht die Einkaufsliste. Fliegt ein Gericht wieder raus, soll
Kanso anbieten, seine Zutaten von der Liste zu nehmen.

## Ziele (vom Nutzer)

- Rezept einem Tag zuweisen; Tage auswählen → Einkaufsliste (neu oder bestehend), wie beim
  Rezept mit Vorschau, alles angehakt.
- Gericht aus dem Plan nehmen → anbieten, die Zutaten wieder abzuziehen.
- Eigene Wochenansicht unter „Küche“, nicht der Kalender.
- **Ein gemeinsamer Plan pro Gruppe**, von einem Admin angelegt (wie Gruppen-Boards).

## Entscheidungen (meine, zum Gegenlesen)

- **Speicher:** in Git (Verlauf), `mealplans/<plan-id>/plan.yaml` (Name, Gruppe) und eine Datei
  pro ISO-Woche `mealplans/<plan-id>/<JJJJ-Www>.yaml` mit den Einträgen je Tag. Ein Commit pro
  Änderung. Kein Offline (die Seite braucht Netz).
- **Eintrag:** `id`, `recipe` (Seiten-ID), `title` (zur Anzeige, falls das Rezept nicht lesbar
  oder gelöscht ist), `servings`, nach dem Einkaufen `fill` und `list` (Fill-ID und Liste aus
  ADR-011).
- **Rechte:** Mitglieder der Gruppe sehen und ändern den Plan. Rezepte, die eine Person nicht
  lesen darf, erscheinen nur mit Titel, ohne Link; einkaufen kann sie die nicht (sie werden in
  der Vorschau übersprungen und genannt).
- **Woche:** Montag bis Sonntag, Blättern mit „‹ Vorige“ / „Nächste ›“, Start: aktuelle Woche.
- **Hinzufügen:** je Tag „+ Gericht“ (Rezept aus einer Auswahl aller lesbaren Rezepte, Portionen
  vorbelegt mit denen des Rezepts) und auf der Rezeptseite „Einplanen“ (Plan, Tag, Portionen).
- **Einkaufen:** Tage per Checkbox wählen → „Einkaufen“. Vorschau je Gericht als Abschnitt
  („Di · Kartoffelcurry · 4 Portionen“), die Zeilen berücksichtigen die Gerichte davor (Mengen
  laufen zusammen). Schon eingekaufte Gerichte (mit `fill`) werden nicht angeboten, sondern als
  „schon auf <Liste>“ genannt. Jedes Gericht bekommt eine eigene Fill-ID; danach stehen `fill`
  und `list` im Eintrag.
- **Entfernen:** Ohne `fill` sofort (mit Rückfrage-freiem POST). Mit `fill`: Bestätigungsseite
  „Zutaten wieder von <Liste> nehmen?“ mit den betroffenen Einträgen; „Entfernen und abziehen“
  (Akzent) oder „Nur aus dem Plan“. Abziehen: Herkunft mit dieser Fill-ID entfernen; Menge um
  den Anteil verringern (verträgliche Einheit), Eintrag löschen, wenn nichts übrig bleibt oder
  das Gericht die einzige Herkunft war; **abgehakte** (gekaufte) Einträge bleiben unverändert.
- **Admin:** `/admin/mealplans` – Plan für eine Gruppe anlegen, löschen (mit Bestätigung).
  Umbenennen später.
- **Küche:** oben zwei Reiter „Rezepte“ (`/recipes`) und „Kochplan“ (`/mealplan`, führt auf den
  ersten Plan der Person; mehrere Pläne als Auswahl).

## Nicht-Ziele

Offline, Drag & Drop, Kalender-Ebene, Vorlagen/wiederkehrende Wochen, Umbenennen von Plänen,
Pläne pro Person.

## Anti-Ziele

- Abziehen nimmt etwas von der Liste, das schon gekauft ist, oder mehr, als das Gericht brachte.
- Ein Gericht landet doppelt auf der Liste, weil es zweimal eingekauft wurde.
- Jemand außerhalb der Gruppe sieht den Plan.

## Tests

Unit (`internal/mealplan`: Wochen, Einträge, Rechte; `internal/shopping.Remove`), HTTP
(Woche, Hinzufügen, Einplanen, Einkaufen mehrerer Tage, Entfernen mit/ohne Abziehen, Rechte,
Admin), Playwright (Admin legt Plan an → zwei Gerichte einplanen → Tage einkaufen → Liste →
ein Gericht entfernen und abziehen).

## Dokumentation

ADR-012 „Kochpläne gehören Gruppen und stehen in Git“; Feature-Doc; arc42; design.md; CHANGELOG;
Gesamtdesign Phase 8 (Essensplan fertig).
