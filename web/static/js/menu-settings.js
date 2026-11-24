// SPDX-License-Identifier: Apache-2.0
// Menu settings: show or hide each part and move it up or down. Every change
// is saved at once on this device and the rail follows it (menu.js, ADR-020).
const menu = window.KansoMenu;
const list = document.getElementById('menu-parts');

function render() {
  const prefs = menu.read();
  const rows = new Map([...list.children].map((li) => [li.dataset.key, li]));
  prefs.order.forEach((key, i) => {
    const li = rows.get(key);
    list.append(li);
    li.querySelector('input').checked = !prefs.hidden.includes(key);
    li.querySelector('[data-move="-1"]').disabled = i === 0;
    li.querySelector('[data-move="1"]').disabled = i === prefs.order.length - 1;
  });
}

function move(key, step) {
  const prefs = menu.read();
  const i = prefs.order.indexOf(key);
  const j = i + step;
  if (j < 0 || j >= prefs.order.length) return;
  [prefs.order[i], prefs.order[j]] = [prefs.order[j], prefs.order[i]];
  menu.write(prefs);
}

function show(key, visible) {
  const prefs = menu.read();
  prefs.hidden = prefs.hidden.filter((k) => k !== key);
  if (!visible) prefs.hidden.push(key);
  menu.write(prefs);
}

list.addEventListener('change', (e) => {
  show(e.target.closest('li').dataset.key, e.target.checked);
  render();
});
list.addEventListener('click', (e) => {
  const button = e.target.closest('[data-move]');
  if (!button) return;
  move(button.closest('li').dataset.key, Number(button.dataset.move));
  render();
  button.closest('li').querySelector(`[data-move="${button.dataset.move}"]:not(:disabled)`)?.focus();
});
document.getElementById('menu-reset').addEventListener('click', () => { menu.reset(); render(); });
render();
