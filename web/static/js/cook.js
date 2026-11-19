// SPDX-License-Identifier: Apache-2.0
// Cook view: remembers ticked ingredients and steps on this device, per
// recipe, and keeps the screen on while the page is visible. Without
// storage or Wake Lock support the page simply works without them.

const root = document.querySelector('.cook');
const key = `kanso.cook.${root?.dataset.recipe}`;
const boxes = root ? [...root.querySelectorAll('input[data-tick]')] : [];

function load() {
  try { return new Set(JSON.parse(localStorage.getItem(key) || '[]')); } catch { return new Set(); }
}

function save() {
  const ticked = boxes.filter((b) => b.checked).map((b) => b.dataset.tick);
  try { localStorage.setItem(key, JSON.stringify(ticked)); } catch { /* private mode: nothing to remember */ }
}

// progress fills the bar over the steps, the same as a to-do list's.
function progress() {
  const bar = root.querySelector('.todo-progress');
  if (!bar) return;
  const steps = boxes.filter((b) => b.closest('.cook-steps'));
  const done = steps.filter((b) => b.checked).length;
  const percent = steps.length ? Math.floor((100 * done) / steps.length) : 0;
  const meter = bar.querySelector('.todo-meter');
  meter.dataset.state = done === 0 ? 'empty' : (done === steps.length ? 'full' : 'partial');
  meter.setAttribute('aria-valuenow', percent);
  meter.querySelector('.todo-meter-fill').style.width = `${percent}%`;
  bar.querySelector('.todo-percent').textContent = bar.dataset.percent.replace('{n}', percent);
}

let lock = null;

async function keepAwake() {
  if (!('wakeLock' in navigator) || document.visibilityState !== 'visible') return;
  try { lock = await navigator.wakeLock.request('screen'); } catch { lock = null; }
}

if (root) {
  const ticked = load();
  for (const box of boxes) {
    box.checked = ticked.has(box.dataset.tick);
    box.addEventListener('change', () => { save(); progress(); });
  }
  progress();
  const reset = root.querySelector('[data-reset]');
  reset.hidden = false;
  reset.addEventListener('click', () => {
    for (const box of boxes) box.checked = false;
    save();
    progress();
  });
  keepAwake();
  document.addEventListener('visibilitychange', () => { if (!lock || lock.released) keepAwake(); });
}
