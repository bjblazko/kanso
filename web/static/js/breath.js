// SPDX-License-Identifier: Apache-2.0
// Breathing 4–6: counts six rounds of four seconds in and six out, as words
// and numbers; no animation, so it suits reduced motion too.
const out = document.getElementById('breath');
const button = document.getElementById('breath-toggle');
const rounds = 6;
const phases = [[out.dataset.in, 4], [out.dataset.out, 6]];
let timer = null;

function label(text) { button.querySelector('span').textContent = text; }

function stop(message = '') {
  clearTimeout(timer);
  timer = null;
  out.textContent = message;
  label(button.dataset.start);
}

/** Shows one second of one phase, then schedules the next. */
function step(round, phase, second) {
  if (round > rounds) { stop(out.dataset.done); return; }
  const [word, length] = phases[phase];
  out.textContent = `${word} ${second} · ${out.dataset.round.replace('%d', round)}`;
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
