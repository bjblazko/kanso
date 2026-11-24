// SPDX-License-Identifier: Apache-2.0
/**
 * When each line of a meditation starts, in seconds after the start: a bell,
 * then line, pause, line, pause, …, and a bell at the end. The fill line's
 * pause grows so that everything takes the meditation's minutes; it never
 * shrinks below the pause it was written with.
 * @returns {{starts: number[], endBell: number, total: number}}
 */
export function plan(durations, pauses, fill, minutes, bell) {
  const own = durations.reduce((a, d) => a + d, 0) + pauses.reduce((a, p) => a + p, 0) + 2 * bell;
  const extra = Math.max(0, minutes * 60 - own);
  const starts = [];
  let at = bell;
  durations.forEach((d, i) => {
    starts.push(at);
    at += d + pauses[i] + (i === fill ? extra : 0);
  });
  return { starts, endBell: at, total: at + bell };
}
