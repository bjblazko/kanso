// SPDX-License-Identifier: Apache-2.0
/**
 * A soft gong made in the browser (no sound file): a few partials of a bowl,
 * each fading out on its own, on the AudioContext's clock at `when`.
 */
export function gong(ctx, when, length = 6, level = 1) {
  for (const [freq, peak, decay] of [[432, 0.22, 1.2], [864.7, 0.08, 1.8], [1297, 0.04, 2.4], [1729, 0.02, 3]]) {
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.frequency.value = freq;
    gain.gain.setValueAtTime(0.0001, when);
    gain.gain.exponentialRampToValueAtTime(peak * level, when + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, when + length / decay);
    osc.connect(gain);
    gain.connect(ctx.destination);
    osc.start(when);
    osc.stop(when + length);
  }
}
