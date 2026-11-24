// SPDX-License-Identifier: Apache-2.0
// Breathing 4–6: counts six rounds of four seconds in and six out, as words
// and numbers; no animation, so it suits reduced motion too.
const out = document.getElementById('breath');
const button = document.getElementById('breath-toggle');
const rounds = 6;
const phases = [[out.dataset.in, 4], [out.dataset.out, 6]];
let timer = null;

const bar = document.getElementById('breath-meter');

/** In-breath fills the bar, out-breath empties it, one second at a time. */
function meter(share, mode) {
  bar.firstElementChild.style.width = `${Math.round(share * 100)}%`;
  bar.dataset.state = mode;
}

function label(text) {
  button.querySelector(':scope > span:last-child').textContent = text;
  button.dataset.running = String(text === button.dataset.stop);
}

function stop(message = '') {
  clearTimeout(timer);
  timer = null;
  out.textContent = message;
  meter(0, 'idle');
  label(button.dataset.start);
}

/** Shows one second of one phase, then schedules the next. */
function step(round, phase, second) {
  if (round > rounds) { stop(out.dataset.done); return; }
  const [word, length] = phases[phase];
  out.textContent = `${word} ${second} · ${out.dataset.round.replace('%d', round)}`;
  meter(phase === 0 ? second / length : 1 - second / length, 'running');
  timer = setTimeout(() => {
    if (second < length) step(round, phase, second + 1);
    else if (phase === 0) step(round, 1, 1);
    else step(round + 1, 0, 1);
  }, 1000);
}

button.addEventListener('click', () => {
  if (timer) { stop(); return; }
  label(button.dataset.stop);
  step(1, 0, 1);
});
