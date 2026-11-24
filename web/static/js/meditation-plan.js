// SPDX-License-Identifier: Apache-2.0
/**
 * When each line of a meditation starts, in seconds after the start: a bell,
 * then line, pause, line, pause, …, and a bell at the end. The time the
 * minutes leave over is shared out among the pauses in proportion to their
 * written length, so long pauses grow most and no single silence gets huge.
 * @returns {{starts: number[], endBell: number, total: number}}
 */
export function plan(durations, pauses, minutes, bell) {
  const written = pauses.reduce((a, p) => a + p, 0);
  const own = durations.reduce((a, d) => a + d, 0) + written + 2 * bell;
  const stretch = written > 0 ? 1 + Math.max(0, minutes * 60 - own) / written : 1;
  const starts = [];
  let at = bell;
  durations.forEach((d, i) => {
    starts.push(at);
    at += d + pauses[i] * stretch;
  });
  return { starts, endBell: at, total: at + bell };
}
