# KI-Anbindung und Rezept von Webseite – Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Admins tragen eine OpenAI-kompatible KI ein; „Von Webseite“ macht aus einer URL einen Rezept-Entwurf (JSON-LD, sonst KI), der nach Bestätigung angelegt wird.

**Spec:** `docs/superpowers/specs/2026-11-07-ai-recipe-import-design.md`

## Global Constraints

SPDX-Kopf; keine neue Abhängigkeit; Funktionen ~30 Zeilen, zyklomatisch ≤ 10; TDD; Doku im selben Commit; Branch `ai-import`. Token nie in Git, Log oder HTML. Keine Abrufe ins eigene Netz. Nichts wird ohne Bestätigung angelegt.

## Review Focus

1. Weiterleitung von einer öffentlichen auf eine interne Adresse wird abgelehnt – Task 2.
2. KI antwortet mit `<think>…</think>` oder ```-Zäunen – wird entfernt – Task 3.
3. Token bleibt beim Speichern ohne neues Token erhalten, erscheint nie im HTML – Task 1/4.
4. JSON-LD mit HowToSection, `recipeYield` als Liste/Zahl/„4 Portionen“, HTML-Entities – Task 2.
5. KI-Antwort ohne Zutaten → Fehlermeldung, kein Entwurf – Task 3.

### Task 1: `internal/ai` – Konfiguration und Client
`type Config struct{Enabled bool; BaseURL, Model, Token string}`; `NewStore(stateDir) *Store`; `Load() Config`; `Save(Config) error` (0600); `(Config) Host() string`; `(Config) Ready() bool`; `type Message struct{Role, Content string}`; `Client{HTTP *http.Client}`; `(c Client) Complete(ctx, cfg, msgs) (string, error)` mit Fehlerarten `ErrUnreachable, ErrAuth, ErrModel, ErrTimeout, ErrBadAnswer`. Tests gegen httptest. ADR-013, Feature-Doc, arc42.

### Task 2: `internal/recipeimport` – Abruf und JSON-LD
`type Fetcher struct{...}`; `NewFetcher() *Fetcher` (sperrt private Ziele); `NewTestFetcher()` (erlaubt Loopback, nur Tests); `(f *Fetcher) Fetch(ctx, url) (Page{URL, HTML string}, error)`; `type Draft struct{Title, Description string; Servings, Time int; Source string; Ingredients, Steps []string}`; `FromJSONLD(html, url) (Draft, bool)`; `(Draft) Markdown(lang) string`; `Text(html) string`.

### Task 3: KI-Weg
`Prompt` (embed `prompt.md`); `FromAI(ctx, client, cfg, page) (Draft, error)`: Text → Anfrage → `<think>`/Zäune weg → `# Titel`, `Portionen:`, `Zeit:`, Abschnitte → Draft; ohne Zutaten `ErrNoRecipe`.

### Task 4: Web
`/admin/ai` (GET/POST: speichern, testen), Küche „Von Webseite“ (`GET /recipes/import`, `POST /recipes/import` → Vorschau, `POST /recipes/import/create` → Seite + Editor), Admin-Menü-Eintrag, i18n, design.md, CHANGELOG.

### Task 5: Playwright, Rams-Review, README-Hinweis, Abschluss
