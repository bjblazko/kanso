# Hüpattl! Kanso – Design (Brainstorming-Ergebnis)

## Context

Docmost (0.96, Postgres + Redis, auf delorean/RPi5) soll abgelöst werden – und Kanso soll mehr werden: ein
Familien-Allrounder für Wissen, Termine, Haus, Kind, Einkauf, Wünsche und Links. Kleines System, wenige
Nutzer → keine Datenbank, alles im Dateisystem, lesbar auch ohne Kanso. Go-Backend, reines HTML/CSS/JS.
Leitbild: Dieter Rams, *weniger, aber besser*.

Ist-Stand (aus `../homelab2`): Docmost hat 6 Spaces (General, Ideen Timo, Homelab, Familie, Haus,
Gesundheit Timo), ~46 Seiten, 74 Anhänge; eigener Login, alle sehen alles. Authelia (file backend
`users.yml`, OIDC aktiv, Grafana nutzt `groups`-Scope). Eigene Apps laufen nach ADR-017 als Flake-Input,
systemd + `DynamicUser` + `StateDirectory`, hinter Caddy. Familienkalender heute in iCloud.

## Entscheidungen (vom Nutzer)

| Thema | Entscheidung |
|---|---|
| Name | „Hüpattl! Kanso“, Repo `kanso`, hier begonnen, behandelt wie ein öffentliches GitHub-Repo |
| Lizenz | Apache-2.0 |
| Nutzer | Alle (auch Großeltern/Gäste) bekommen ein Authelia-Konto; keine öffentlichen Links |
| Gruppen | in Kanso gepflegt; Authelia liefert Identität + Admin-Flag (Gruppe `kanso-admin`) |
| Gleichzeitiges Bearbeiten | Bearbeitungssperre pro Dokument |
| Verlauf | Datenordner ist ein Git-Repo, jedes Speichern = Commit mit Autor (git-CLI, keine Lib) |
| Bibliotheken | so wenig wie möglich, jede einzeln nach strenger Prüfung vom Nutzer freigegeben |
| Kalender | Start mit eigenem leeren Kalender im Datenordner; CalDAV/extern später |
| Mobil | mobil zuerst, Offline-Listen (PWA) |
| Benachrichtigungen | Web-Push |
| Sprache | Deutsch + Englisch, pro Person umschaltbar |
| Extras | wiederkehrende Aufgaben, Essensplan → Einkauf, Haus-Inventar, Kind & Kontakte |

## Meine Entscheidungen (zum Gegenlesen markiert)

- **Rollen**: global `admin` (aus Authelia-Gruppe `kanso-admin`); pro Space `owner` / `editor` / `viewer`,
  vergeben an Personen oder Kanso-Gruppen; höchste Rolle gewinnt. Kein Zugriff = Space unsichtbar.
- **Auth**: OIDC Authorization Code + PKCE gegen Authelia, implementiert mit Go-Stdlib
  (`net/http`, `crypto/*`, JWKS-Abruf, ID-Token-Prüfung). Session-Cookie (HttpOnly, Secure, SameSite=Lax),
  Sessions in einer Datei. Neuer Authelia-Client `kanso` in `homelab2/nas-app/modules/authelia.nix`
  (scopes `openid profile email groups`). Dev-Modus mit lokalem Fake-Login nur per Flag.
- **Sperre**: 5 min ohne Heartbeat → frei; Owner/Admin kann brechen; nur im Speicher.
- **Listen, Boards, Tasks**: eine Datei pro Eintrag statt einer großen Datei → keine Sperren nötig,
  wenig Konflikte, Offline-Abgleich pro Eintrag (last-write-wins pro Eintrag + Revision).
- **Suche**: In-Memory-Index beim Start, aktualisiert beim Speichern (kein externer Index).
- **Docmost-Import**: einmaliges Kommando `kanso import-docmost`, liest direkt aus Postgres
  (ProseMirror-JSON → Markdown, Hierarchie, Zeitstempel, Autoren, Anhänge aus dem NAS-Ordner).
  Fallback: Markdown-ZIP-Export von Docmost. Braucht einen Postgres-Treiber → nur in einem separaten
  Import-Binary (`cmd/kanso-import`), damit der Server frei davon bleibt; Freigabe nötig.
- **Aus Notiz wird …**: Task (Notiz bleibt, verlinkt) oder Wiki-Artikel (Notiz wird verschoben,
  Anhänge wandern mit, Git-Verlauf bleibt via `git mv`).
- **Task-Ansichten**: ein Task hat `status` (Lane) **und** `urgent`/`important`; Kanban und
  Eisenhower sind nur Ansichten derselben Dateien. Lanes pro Board konfigurierbar.
- **Wunschliste**: „Ich besorge das“ ist für die Person, der die Liste gehört, unsichtbar.
- Telemetrie/Update-Check: keine. Einzige ausgehende Verbindungen: Authelia, Web-Push-Dienste
  (Apple/Google/Mozilla, verschlüsselte Payload), später CalDAV/ICS. Wird im Legal-Check dokumentiert.

## Datenlayout (Datenordner = Git-Repo)

```
data/
  kanso.yaml                     # Gruppen, Space-Rollen, Einstellungen
  users/<sub>.yaml               # Profil: Name, Sprache, Push-Abos (beim ersten Login angelegt)
  spaces/<space-slug>/
    space.yaml                   # Name, Icon, Beschreibung, Rollen
    pages/<page-slug>/
      index.md                   # YAML-Frontmatter (id, title, order, created, author) + Markdown
      attachments/…              # Bilder, PDFs; Markdown referenziert relativ
      <child-slug>/index.md      # Hierarchie = Ordnerhierarchie
  notes/<sub>/<note-id>/index.md (+ attachments/)
  boards/<board-id>/
    board.yaml                   # Name, Besitzer (Person oder Gruppe), Lanes
    tasks/<task-id>.md           # Frontmatter: status, urgent, important, due, assignee, recur, source
  lists/<list-id>/
    list.yaml                    # Art: shopping|wish|links|…, Besitzer, Freigaben
    items/<item-id>.md
  calendars/<cal-id>/
    calendar.yaml
    events/<uid>.ics             # RFC 5545 – CalDAV-tauglich
```

IDs: kurze, zufällige, URL-sichere IDs; Slugs nur für lesbare Ordnernamen bei Wiki-Seiten (Umbenennen =
`git mv`, interne Links per ID, damit sie nicht brechen).

## Architektur

- Ein Binary `kanso`, Frontend per `embed` eingebettet, Konfiguration per Env/Flags, lauscht auf localhost.
- Domänen-first Pakete (huepattl-code-quality): `internal/auth`, `internal/access` (Rollen/Rechte),
  `internal/store` (Dateien + Git + Sperren), `internal/wiki`, `internal/notes`, `internal/tasks`,
  `internal/lists`, `internal/calendar`, `internal/push`, `internal/search`, `internal/i18n`, `web/`.
- Server rendert Markdown → HTML (Sanitizing!); Editor = Textarea mit Quelltext/Vorschau-Umschaltung
  (Mobil) bzw. nebeneinander (breit). Vorschau über denselben Server-Renderer.
- Frontend: Seiten servergerendert (Go `html/template`), JS nur für Editor, Drag & Drop,
  Offline/Service Worker, Push. Design-Tokens aus huepattl-rams-design (`tokens.css`), IBM Plex selbst
  gehostet (OFL → Legal-Check).
- Sicherheit: CSP ohne Inline-Skripte, CSRF-Token, alle Pfade gegen Path-Traversal prüfen,
  Anhänge mit `Content-Disposition` und festem Content-Type, Upload-Limits.

## Fremdabhängigkeiten – Kandidaten (jede einzeln zur Freigabe vorlegen)

| Kandidat | Zweck | Alternative ohne Lib |
|---|---|---|
| `github.com/yuin/goldmark` (MIT) + Erweiterungen (Tabellen, Tasklisten) | Markdown → HTML | eigener Parser (großer Aufwand, Fehlerquelle) |
| `mermaid.js` (MIT), lokal, nur bei Bedarf geladen | Diagramme | nur Quelltext anzeigen |
| `gopkg.in/yaml.v3` (MIT/Apache) | Frontmatter/Config | JSON statt YAML (Stdlib) – prüfen! |
| Postgres-Treiber `pgx` (MIT), nur im Import-Binary | Docmost-Import | ZIP-Export-Import |
| IBM Plex (OFL) | Schrift | Systemschrift |

OIDC, Web-Push (VAPID, RFC 8291 via `crypto/ecdh`, `crypto/hkdf`), iCalendar, Git (CLI): selbst gebaut.

## Phasen (je Phase eigener Implementierungsplan via superpowers:writing-plans)

1. **Fundament + Wiki**: Repo-Gerüst (LICENSE Apache-2.0, README, CHANGELOG, `docs/adr/`, CLAUDE.md),
   Auth/OIDC, Rollen/Gruppen, Spaces, Seiten-Baum, Editor Quelltext/Vorschau, Anhänge, Mermaid,
   Sperren, Git-Verlauf + Vergleich, Suche, i18n DE/EN, Design-System.
2. **Docmost-Import** + Deployment in homelab2 (Flake-Input, Modul, Authelia-Client, Caddy, DNS) und Parallelbetrieb.
3. **Notizen** (privat) → Task / Wiki-Artikel.
4. **Tasks**: Boards (privat/Gruppe), Kanban + Eisenhower, Fälligkeiten, wiederkehrende Aufgaben.
5. **Listen**: Einkauf, Wunschlisten (Freigabe, „Ich besorge das“), Links.
6. **Kalender** nativ (.ics), Tasks mit Datum im Kalender, Geburtstage.
7. **Mobil/Offline (PWA)** + **Web-Push**.
8. **Extras**: Essensplan → Einkauf, Haus-Inventar (Wartung erzeugt Tasks), Kind & Kontakte.
9. **Kalender extern**: ICS-Abo, CalDAV-Client (iCloud), später CalDAV-Server.

Docmost wird erst abgeschaltet, wenn Phase 2 im Alltag besteht.

## Verifikation

- `go test ./...`, `go vet`, `staticcheck`/`golangci-lint`, `gocyclo` gemäß huepattl-code-quality.
- Integrationstests mit temporärem Datenordner (Git-Commits, Rechte pro Rolle, Sperren, Path-Traversal).
- OIDC lokal gegen Authelia-Testinstanz oder Fake-Provider im Test.
- UI: Review nach huepattl-rams-design (`review.md`), Kontrast-Skript, Zustände leer/Fehler/offline.
- Docmost-Import: Probelauf gegen Kopie des Dumps, Seiten-/Anhangzahl (46/74) vergleichen, Stichproben.
- Vor jedem Release/Deploy: huepattl-legal-check (Lizenzliste „Licenses and thanks“ in der App,
  ausgehende Verbindungen, Haftungsausschluss).

## Nächste Schritte nach Freigabe

1. `git init` in `kanso/`, Design als `docs/superpowers/specs/2026-10-26-kanso-design.md` committen.
2. Dependency-Prüfung der Kandidaten → Freigabe pro Kandidat einholen.
3. superpowers:writing-plans für Phase 1.

Hinweis: Der Update-Check der huepattl-Skills wurde im Plan-Modus übersprungen (schreibt eine Datei).
