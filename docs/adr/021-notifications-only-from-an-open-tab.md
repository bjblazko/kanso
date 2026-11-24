# 021 — Notifications Only from an Open Tab

**Status:** Accepted
**Created:** 2026-11-24
**Updated:** 2026-11-24

## Context

The first design planned Web Push ("Offline and Web Push"). Fokus, the
Pomodoro timer, is the first function that needs to alert someone. Web Push
needs VAPID keys, encrypted messages and the browser vendors' push services
(Google, Apple, Mozilla), so data would leave the house.

## Decision

Kanso alerts only from a tab that is open: a sound made with Web Audio (no
file) and, if the person allowed it, a `Notification`. There is no Web Push,
now or later (owner, 2026-11-24). Timers count against an end time stored on
the device, so a throttled background tab and a reload stay exact.

## Alternatives Considered

- **Web Push:** reaches a locked phone, but it adds a third party and server
  keys, and it means scheduling on the server.

## Consequences

- A closed tab or a sleeping phone does not ring. Fokus says so on its page.
- The "Offline and Web Push" feature shrinks to offline use.
