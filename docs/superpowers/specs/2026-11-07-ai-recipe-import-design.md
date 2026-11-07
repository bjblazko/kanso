# Hüpattl! Kanso – KI-Anbindung und Rezept von Webseite

**Status:** freigegeben · **Created:** 2026-11-07 · **Updated:** 2026-11-07

Baut auf [Rezepte](2026-11-06-recipes-design.md) (ADR-010) auf; ersetzt dort „Stufe 2: KI-Erfassung“.

## Warum

Rezepte abtippen kostet Zeit. Die meisten Rezeptseiten liefern ihre Daten maschinenlesbar mit;
für den Rest kann eine KI den Text in Kansos Rezeptformat bringen. Die KI soll später auch im
Wiki helfen (aufräumen, zusammenfassen, finden). Kanso soll als öffentliches Projekt mit jeder
üblichen KI laufen: lokal (Ollama) oder gehostet (OpenRouter, OpenAI …).

## Ziele (vom Nutzer)

- Admins registrieren **zentral** eine KI (LLM). Protokoll: Empfehlung erwünscht; bei ihm läuft es
  über die lokale KI aus homelab2 (Ollama hinter einem Wake-Proxy, **ohne Auth**); andere sollen
  z. B. **OpenRouter** mit Token nutzen können.
- Erstes Feature: in der Küche ein Knopf, **URL eingeben → Rezept in unserer Struktur**.
  Foto/Instagram später.
- Zugriff der KI auf Kanso-Inhalte **über eine API statt direkt aufs Dateisystem**, damit
  „nur lesen“ sauber geht.
- **KI an = für alle an.**

## Entscheidungen (meine, zum Gegenlesen)

### Protokoll und Anbindung

- **OpenAI-kompatible Chat Completions:** `POST {base}/chat/completions`, JSON, optional
  `Authorization: Bearer <token>`, kein Streaming. Läuft mit Ollama (`…/v1`), OpenRouter,
  OpenAI, LM Studio, llama.cpp, vLLM, LiteLLM. Nur Go-Standardbibliothek.
- **Admin › KI-Anbindung** (`/admin/ai`): Basis-URL, Modell, Token (optional), an/aus,
  „Verbindung testen“. Gespeichert in `<StateDir>/ai.json` (Datei 0600, **nicht im Daten-Git**);
  der Token wird nie wieder angezeigt, nur „gesetzt“ – leer lassen behält ihn, „entfernen“ löscht.
- Ein Paket `internal/ai`: `Client.Complete(ctx, messages) (string, error)` mit Zeitlimit
  **180 s** (Aufwachen + Laden + Rechnen), Antwort-Grenze 1 MB, verständliche Fehler
  („KI nicht erreichbar“, „Modell unbekannt“, „Zeit abgelaufen“, „Token abgelehnt“).
- Jede KI-Funktion sagt **vor** dem Absenden, wohin Text geht (Host der Basis-URL).
- Hinweis auf der Admin-Seite (Legal-Check 2026-11-07): Gehostete Anbieter verarbeiten den
  gesendeten Text; dafür kann ein Auftragsverarbeitungsvertrag mit dem Anbieter nötig sein.
  Kanso schreibt keine gesendeten oder empfangenen Inhalte ins Log.

### Zugriff auf Inhalte (für später, Grundstein jetzt)

- Die KI greift nie auf Dateien zu. Sie bekommt Inhalte nur über Kansos Go-Dienste **mit den
  Rechten der Person**, die die Funktion auslöst (gleiche Prüfungen wie die Weboberfläche).
- Ergebnisse werden **Entwürfe**, die die Person bestätigt; die KI schreibt nie selbst. Für die
  spätere Wiki-Hilfe werden diese Dienste als lesende „Werkzeuge“ (Function Calling) angeboten.
- Rezept-Import braucht noch keine Werkzeuge: eine einzelne Anfrage.

### Rezept von Webseite

- Küche: „Von Webseite“ neben „Neues Rezept“ → Formular: URL, Wiki-Bereich (schreibbar).
- **Abruf** vom Server: nur `http`/`https`, Ports 80/443, höchstens 5 Weiterleitungen,
  10 s Zeitlimit, höchstens 3 MB, nur `text/html`; **keine Ziele im eigenen Netz** (Loopback,
  private, link-local, ULA – geprüft für jede aufgelöste IP und jede Weiterleitung, gegen SSRF).
  User-Agent `Kanso (+https://github.com/bjblazko/kanso)` ohne persönliche Daten.
- **Zuerst ohne KI:** JSON-LD `Recipe` (auch in `@graph`, Listen) → Titel, Beschreibung,
  `recipeYield` → Portionen, `totalTime`/`cookTime`+`prepTime` (ISO-8601-Dauer) → Minuten,
  `recipeIngredient` → Zutaten, `recipeInstructions` (Text, HowToStep, HowToSection) → Schritte,
  `keywords`/`recipeCategory` → keine Tags (zu laut; *meine Entscheidung*), Quelle = URL.
- **Sonst KI** (wenn an): Seitentext ohne Skripte/Stile/Navigation, gekürzt auf 30 000 Zeichen,
  mit festem System-Prompt (im Repo, beschreibt das Format aus ADR-010 genau) → Markdown.
  Antwort durch denselben Parser; ohne erkennbare Zutaten → Fehlermeldung statt Entwurf.
- Ohne JSON-LD und ohne KI: klare Meldung „Diese Seite hat keine Rezeptdaten; mit einer
  KI-Anbindung ginge es“.
- **Ergebnis = Entwurf:** Kanso legt die Seite erst an, wenn die Person auf der Vorschau
  „Rezept anlegen“ drückt: Titel, Portionen, Zutaten, Schritte und Quelle sichtbar, dazu
  „So versteht Kanso die Zutaten“. Danach Editor wie gewohnt.
- Am Ende des Rezepttexts: `Quelle: [Seite](URL). Für den privaten Gebrauch übernommen.`
- Lange Wartezeit: Der Knopf zeigt „Wird abgerufen … (mit KI bis zu 3 Minuten)“ und wird
  gesperrt; ohne JS lädt die Seite einfach länger.

## Nachtrag 2026-11-07: Fotos (Nutzer)

„Mindestens ein Foto übernehmen, auch als Vorschau in der Liste; mehrere gern, Schrittbilder in
die Schritte.“ Umgesetzt: Hauptfoto (JSON-LD `image`, sonst `og:image`/`twitter:image`) oben,
Schrittbilder (HowToStep `image`) unter ihrem Schritt, weitere Fotos der Seite wählbar (Oben, nach
Schritt n, am Ende, nicht übernehmen). Fotos holt der Server mit denselben Schutzregeln (nur
JPEG/PNG/WebP/GIF, ≤ 8 MB); die Vorschau zeigt sie als `data:`-Adressen (≤ 2 MB), der Browser lädt
nichts von fremden Seiten. Gespeichert als Anhänge, JPEG ohne GPS, `image` im Frontmatter.

## Nicht-Ziele

Instagram, Foto als Eingabe (statt URL), mehrere Anbindungen, Rechte pro Gruppe, Streaming, Kosten-Anzeige,
Wiki-Werkzeuge (nur der Grundstein), Bilder des Rezepts übernehmen.

## Anti-Ziele

- Kanso ruft interne Adressen ab, weil jemand eine URL eintippt.
- Ein Token landet im Git, im Log oder wieder auf einer Seite.
- Etwas wird ohne Bestätigung im Wiki angelegt.
- Text geht an eine KI, ohne dass die Seite das vorher sagt.

## Tests

Unit: JSON-LD-Varianten (einfach, `@graph`, Liste, HowToSection, Dauer, Yield „4 Portionen“),
HTML→Text, Adressprüfung (127.0.0.1, 10.x, 192.168.x, ::1, fd00::, Weiterleitung auf intern),
KI-Client gegen Fake-Server (ok, 401, 404 Modell, Zeitlimit, kaputtes JSON), Konfig ohne Token
im Klartext auf Seiten. HTTP: Admin-Seite (nur Admins), Import mit Fake-Webseite (JSON-LD),
mit Fake-KI, ohne KI, Vorschau → Anlegen. Playwright: Admin trägt Fake-KI ein → Import.

## Dokumentation

ADR-013 „KI über OpenAI-kompatible Schnittstelle, Zugriff nur über Kanso-Dienste“; Feature-Doc;
arc42 (Baustein, Risiko ausgehende Verbindungen); Legal-Check vor dem Bauen (neue ausgehende
Verbindungen: Rezeptseiten, KI-Anbieter) und Hinweis in „Lizenzen und Dank“/Datenschutz-Text,
was wann wohin geht.
