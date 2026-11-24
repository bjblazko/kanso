# 023 — Meditations as Spoken Clips, Timed by the Page

**Status:** Accepted
**Created:** 2026-11-24
**Updated:** 2026-11-24

## Context

Selfcare gets two guided meditations (5 and 10 minutes). Each one is spoken
in a voice the person chooses, in the UI language. Speech comes from the
household's AI PC (Qwen3-TTS), which Kanso must not depend on at run time
(no network calls, ADR-013 keeps AI optional). The owner wants no large
audio files.

## Decision

Each line of a meditation is spoken once per voice and language ahead of
time (`tools/meditation`). The result is one small MP3 clip per line: mono,
24 kHz, VBR, trimmed and normalized to −16 LUFS. The clips are served from
`web/static/audio/meditation/<voice>/<lang>/`. The page loads the clips of
the chosen voice, works out the pauses (one "fill" pause makes up the
minutes) and puts every clip and the two gongs on the AudioContext's clock
at once. The browser makes the gong itself (no file). While a meditation
plays, a Wake Lock keeps the screen on. The voices are Lyra and Orion,
designed voices (Qwen3-TTS VoiceDesign) rather than clones of real people.

## Alternatives Considered

- **One mixed file per meditation, voice and language:** plays on with a
  locked phone, but about 20 MB in all, and every text change means
  re-mixing everything.
- **Speech at run time through the AI connection:** needs the PC to be awake
  and on the network, and adds latency.

## Consequences

- About 3–4 MB of clips in the repository and the binary. A test checks
  that every clip exists and stays small.
- A locked phone may stop the sound (iOS). The page says so and keeps the
  screen on.
- Changing a line means speaking that line again on the PC
  (`tools/meditation/README.md`).
