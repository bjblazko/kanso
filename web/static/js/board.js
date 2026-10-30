// SPDX-License-Identifier: Apache-2.0
// Board: drag a card into another column (Kanban) or quadrant (matrix). The
// server answers first; only then the board is redrawn from the server, so the
// order and the marks stay right. A failed move leaves the card where it was.
// Without dragging (phone, keyboard) the task's own page changes the same fields.
// The device also remembers the last board and the last tab per board.

const page = document.querySelector('.board-page');
const status = document.getElementById('board-status');
const csrf = document.querySelector('meta[name="csrf"]').content;
const { board, view } = page.dataset;

// A quadrant is a pair of urgent and important.
const QUADRANTS = {
  do_now: { urgent: '1', important: '1' },
  plan: { urgent: '0', important: '1' },
  quick: { urgent: '1', important: '0' },
  later: { urgent: '0', important: '0' },
};

let dragged = null; // the card being dragged

// ---- Remembering (per device; works without storage too) ------------------

function remembered(key) {
  try { return localStorage.getItem(key); } catch { return null; }
}

function remember(key, value) {
  try { localStorage.setItem(key, value); } catch { /* private mode: nothing to remember */ }
}

// returnToLast opens the remembered board and tab; true when it navigates away.
function returnToLast() {
  const params = new URLSearchParams(location.search);
  const last = remembered('kanso.tasks.board');
  if (location.pathname === '/tasks' && last && last !== board &&
      document.querySelector(`.board-switch a[href="/tasks/b/${CSS.escape(last)}"]`)) {
    location.replace(`/tasks/b/${last}`);
    return true;
  }
  const lastView = remembered(`kanso.tasks.view.${board}`);
  if (!params.has('view') && lastView === 'matrix' && view !== 'matrix') {
    location.replace(`/tasks/b/${board}?view=matrix`);
    return true;
  }
  return false;
}

// ---- Moving ----------------------------------------------------------------

function fieldsFor(key) {
  return view === 'matrix' ? QUADRANTS[key] : { status: key };
}

function say(message) {
  status.textContent = message;
  status.hidden = !message;
}

async function move(card, key) {
  const response = await fetch(`/tasks/b/${board}/${card.dataset.task}/move`, {
    method: 'POST', credentials: 'same-origin',
    headers: { 'X-CSRF-Token': csrf, Accept: 'application/json' },
    body: new URLSearchParams(fieldsFor(key)),
  }).catch(() => null);
  if (!response?.ok) {
    say(response?.status === 404 ? page.dataset.msgMoveGone : page.dataset.msgMoveFailed);
    return;
  }
  say('');
  await redraw();
}

// redraw replaces the columns with the server's current board.
async function redraw() {
  const response = await fetch(location.href, { credentials: 'same-origin' }).catch(() => null);
  if (!response?.ok) return;
  const html = new DOMParser().parseFromString(await response.text(), 'text/html');
  const fresh = html.querySelector('.board');
  if (fresh) document.querySelector('.board').replaceWith(fresh);
}

// ---- Wiring (delegated, so a redrawn board needs nothing) ------------------

function column(event) {
  return event.target.closest?.('.board-col');
}

document.addEventListener('dragstart', (e) => {
  dragged = e.target.closest?.('.card');
  if (!dragged) return;
  dragged.classList.add('dragging');
  e.dataTransfer.effectAllowed = 'move';
  e.dataTransfer.setData('text/plain', dragged.dataset.task);
});
document.addEventListener('dragend', () => {
  dragged?.classList.remove('dragging');
  dragged = null;
  for (const c of document.querySelectorAll('.drop-target')) c.classList.remove('drop-target');
});
document.addEventListener('dragover', (e) => {
  const col = column(e);
  if (!dragged || !col) return;
  e.preventDefault();
  col.classList.add('drop-target');
});
document.addEventListener('dragleave', (e) => {
  const col = column(e);
  if (col && !col.contains(e.relatedTarget)) col.classList.remove('drop-target');
});
document.addEventListener('drop', (e) => {
  const col = column(e);
  if (!dragged || !col) return;
  e.preventDefault();
  col.classList.remove('drop-target');
  if (!col.contains(dragged)) move(dragged, col.dataset.key);
});

if (!returnToLast()) {
  remember('kanso.tasks.board', board);
  remember(`kanso.tasks.view.${board}`, view);
}
