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

let lock = null;

async function keepAwake() {
  if (!('wakeLock' in navigator) || document.visibilityState !== 'visible') return;
  try { lock = await navigator.wakeLock.request('screen'); } catch { lock = null; }
}

if (root) {
  const ticked = load();
  for (const box of boxes) {
    box.checked = ticked.has(box.dataset.tick);
    box.addEventListener('change', save);
  }
  keepAwake();
  document.addEventListener('visibilitychange', () => { if (!lock || lock.released) keepAwake(); });
}
