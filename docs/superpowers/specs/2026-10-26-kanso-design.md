# Hüpattl! Kanso – Design

**Status:** freigegeben · **Created:** 2026-10-26 · **Updated:** 2026-10-27

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
- **Docmost-Migration**: nicht Teil des Projekts (Nutzer). Kanso startet normalerweise leer; die
  eigene Migration wird später einmalig und separat gemacht. Das Spaces/Seiten-Modell orientiert sich
  grob an Docmost, damit das dann leicht fällt.
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
  kanso.yaml                     # Gruppen, Einstellungen (Space-Rollen stehen in space.yaml)
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

## Fremdabhängigkeiten (geprüft 2026-10-26, vom Nutzer einzeln freigegeben)

| Abhängigkeit | Version | Lizenz | Zweck | Auflagen |
|---|---|---|---|---|
| `github.com/yuin/goldmark/v2` | v2.1.9 | MIT | Markdown → HTML | rohes HTML aus, Ausgabe zusätzlich sanitizen |
| `go.yaml.in/yaml/v3` | v3 (frozen, Security-Fixes) | Apache-2.0/MIT | Frontmatter, Konfiguration | auf v4 wechseln, wenn stabil |
| `mermaid` (ESM-Build) | 12.2.0 | MIT (+ MIT/ISC/BSD/Apache-Abhängigkeiten, DOMPurify unter Apache-2.0 gewählt, robust-predicates Unlicense) | Diagramme, lokal, lazy | **ohne ELK-Chunk** (elkjs = EPL-2.0), Layout fest `dagre` |
| IBM Plex Sans/Mono | aktuell | OFL-1.1 | Schrift | selbst gehostet, in „Lizenzen und Dank“ |

Abgelehnt: `pgx` (Docmost-Migration ist kein Projektteil). Neue Abhängigkeiten nur nach gleicher Prüfung
und Freigabe. OIDC, Web-Push (VAPID, RFC 8291 via `crypto/ecdh`, `crypto/hkdf`), iCalendar, Git (CLI):
selbst gebaut. Alle Lizenztexte kommen in die App („Lizenzen und Dank“) mit Test auf Vollständigkeit.

## Phasen (je Phase eigener Implementierungsplan via superpowers:writing-plans)

1. **Fundament + Wiki**: Repo-Gerüst (LICENSE Apache-2.0, README, CHANGELOG, `docs/adr/`, CLAUDE.md),
   Auth/OIDC, Rollen/Gruppen, Spaces, Seiten-Baum, Editor Quelltext/Vorschau, Anhänge, Mermaid,
   Sperren, Git-Verlauf + Vergleich, Suche, i18n DE/EN, Design-System. *(fertig)*
2. **Notizen** (privat) → Wiki-Artikel; schnelles Erfassen am Handy, auch offline, mit Fotos. *(fertig)*
   Dazwischen: **To-do-Listen** (offline, teilbar; eigene Spec). *(fertig)*
3. **Deployment** in homelab2 (Flake-Input, Modul, Authelia-Client, Caddy) und Parallelbetrieb zu Docmost.
   *(fertig)*
4. **Tasks**: Boards (privat/Gruppe), Kanban + Eisenhower, Fälligkeiten, wiederkehrende Aufgaben; Notiz → Task.
5. **Listen**: Einkauf (eigener Listentyp, nicht die To-do-Listen), Wunschlisten (Freigabe, „Ich besorge das“), Links.
6. **Kalender** nativ (.ics), Tasks mit Datum im Kalender, Geburtstage.
7. **Mobil/Offline (PWA)** + **Web-Push**.
8. **Extras**: Essensplan → Einkauf, Haus-Inventar (Wartung erzeugt Tasks), Kind & Kontakte.
9. **Kalender extern**: ICS-Abo, CalDAV-Client (iCloud), später CalDAV-Server.

Docmost wird erst abgeschaltet, wenn Kanso im Alltag besteht und die separate Migration gelaufen ist.

## Verifikation

- `go test ./...`, `go vet`, `staticcheck`/`golangci-lint`, `gocyclo` gemäß huepattl-code-quality.
- Integrationstests mit temporärem Datenordner (Git-Commits, Rechte pro Rolle, Sperren, Path-Traversal).
- OIDC lokal gegen Authelia-Testinstanz oder Fake-Provider im Test.
- UI: Review nach huepattl-rams-design (`review.md`), Kontrast-Skript, Zustände leer/Fehler/offline.
- Vor jedem Release/Deploy: huepattl-legal-check (Lizenzliste „Licenses and thanks“ in der App,
  ausgehende Verbindungen, Haftungsausschluss).
