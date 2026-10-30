// SPDX-License-Identifier: Apache-2.0
// Task capture: a task goes into an outbox on the device first (IndexedDB
// "kanso-tasks") and is uploaded when the network allows; only a confirmed
// upload removes it, so nothing is lost on the way. The device makes the id,
// so sending a task twice after a lost answer stores it once. The boards to
// choose from come with the page, which the service worker keeps for offline.

const SYNC_MS = 30_000;
const ID_ALPHABET = 'abcdefghijklmnopqrstuvwxyz234567';

const form = document.getElementById('task-capture');
const title = form.elements.title;
const status = document.getElementById('capture-status');
const outbox = document.getElementById('outbox');
const msg = form.dataset;

let justSaved = ''; // board name of the task saved last
let moved = false; // a task went to the own board because its board is gone
let needsLogin = false;
let syncing = false;
let syncAgain = false;

// ---- Outbox (IndexedDB) ----------------------------------------------------

const db = new Promise((resolve, reject) => {
  const open = indexedDB.open('kanso-tasks', 1);
  open.onupgradeneeded = () => open.result.createObjectStore('outbox', { keyPath: 'id' });
  open.onsuccess = () => resolve(open.result);
  open.onerror = () => reject(open.error);
});

async function inOutbox(mode, work) {
  const store = (await db).transaction('outbox', mode).objectStore('outbox');
  return new Promise((resolve, reject) => {
    const request = work(store);
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

const allWaiting = () => inOutbox('readonly', (s) => s.getAll());
const keep = (task) => inOutbox('readwrite', (s) => s.put(task));
const drop = (id) => inOutbox('readwrite', (s) => s.delete(id));

function newID() {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return Array.from(bytes, (b) => ID_ALPHABET[b % 32]).join('');
}

// ---- Due dates in the device's own time ------------------------------------

function isoDay(d) {
  const pad = (n) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

// setDueDates gives the quick choices today's dates; the kept page may be old.
function setDueDates() {
  const now = new Date();
  const plus = (days) => new Date(now.getFullYear(), now.getMonth(), now.getDate() + days);
  const offsets = { today: 0, tomorrow: 1, week: (7 - now.getDay()) % 7 }; // week ends on Sunday
  for (const input of form.querySelectorAll('input[data-due]')) {
    input.value = isoDay(plus(offsets[input.dataset.due]));
  }
}

function chosenDue() {
  const due = form.elements.due.value;
  return due === 'date' ? form.elements.due_date.value : due;
}

// ---- Status ----------------------------------------------------------------

function plural(one, many, n) {
  return (n === 1 ? one : many).replace('{n}', String(n));
}

async function render() {
  const tasks = await allWaiting();
  const waiting = tasks.filter((t) => !t.failed);
  const failed = tasks.filter((t) => t.failed);
  const parts = [];
  if (justSaved && waiting.length === 0) parts.push(msg.msgSaved.replace('{board}', justSaved));
  if (moved) parts.push(msg.msgMoved);
  if (waiting.length) parts.push(plural(msg.msgWaitingOne, msg.msgWaitingMany, waiting.length));
  if (failed.length) parts.push(plural(msg.msgFailedOne, msg.msgFailedMany, failed.length));
  status.replaceChildren(parts.join(' · '));
  if (needsLogin && waiting.length) {
    const link = Object.assign(document.createElement('a'), { href: msg.loginUrl, textContent: msg.msgLogin });
    status.append(status.textContent ? ' · ' : '', link);
  }
  outbox.hidden = tasks.length === 0;
  outbox.querySelector('ul').replaceChildren(...tasks.map((t) => {
    const li = document.createElement('li');
    li.textContent = t.failed ? `${t.title} (${msg.msgFailedMark})` : t.title;
    return li;
  }));
}

// ---- Upload ----------------------------------------------------------------

// token returns a fresh CSRF token, null when signed out, undefined offline.
async function token() {
  try {
    const response = await fetch('/notes/token', { credentials: 'same-origin', redirect: 'manual' });
    const json = (response.headers.get('Content-Type') || '').startsWith('application/json');
    if (response.type === 'opaqueredirect' || !response.ok || !json) return null;
    return (await response.json()).csrf;
  } catch {
    return undefined;
  }
}

// upload sends one task; the answer says what to do with it next.
async function upload(task, csrf) {
  const data = new URLSearchParams({ id: task.id, title: task.title, board: task.board, due: task.due });
  if (task.urgent) data.set('urgent', '1');
  if (task.important) data.set('important', '1');
  try {
    const response = await fetch('/tasks', {
      method: 'POST', body: data, credentials: 'same-origin',
      headers: { 'X-CSRF-Token': csrf, Accept: 'application/json' },
    });
    if (response.ok) {
      const answer = await response.json().catch(() => ({}));
      // Only a new task lands elsewhere because its board is gone; an "exists"
      // answer names where someone moved the task meanwhile.
      if (answer.status === 'created' && task.board && answer.board !== task.board) moved = true;
      return 'done';
    }
    if (response.status === 400 || response.status === 413) return 'failed';
    if (response.status === 401 || response.status === 403) return 'login';
    return 'later';
  } catch {
    return 'later';
  }
}

// sync uploads waiting tasks; a call during an upload runs again afterwards.
async function sync() {
  if (syncing) {
    syncAgain = true;
    return;
  }
  syncing = true;
  try {
    do {
      syncAgain = false;
      await uploadWaiting();
    } while (syncAgain);
  } finally {
    syncing = false;
    await render();
  }
}

async function uploadWaiting() {
  const waiting = (await allWaiting()).filter((t) => !t.failed);
  if (waiting.length === 0) return;
  const csrf = await token();
  needsLogin = csrf === null;
  if (!csrf) return;
  for (const task of waiting) {
    const result = await upload(task, csrf);
    if (result === 'done') await drop(task.id);
    else if (result === 'failed') await keep({ ...task, failed: true });
    else {
      needsLogin = result === 'login';
      return;
    }
  }
}

// ---- Writing ---------------------------------------------------------------

async function save(event) {
  event.preventDefault();
  if (!title.value.trim()) return title.focus();
  const board = form.querySelector('input[name="board"]:checked');
  const task = {
    id: newID(), title: title.value.trim(), board: board?.value ?? '', boardName: board?.dataset.name ?? '',
    urgent: form.elements.urgent.checked, important: form.elements.important.checked, due: chosenDue(), failed: false,
  };
  if (!(await stored(task))) {
    status.replaceChildren(msg.msgNotSaved);
    return;
  }
  title.value = '';
  form.elements.urgent.checked = form.elements.important.checked = false;
  form.querySelector('input[name="due"][value=""]').checked = true;
  form.elements.due_date.value = '';
  justSaved = task.boardName;
  title.focus();
  await render();
  await sync();
}

// stored puts the task into the outbox, or uploads it at once if the outbox
// cannot be written; false means it is nowhere yet and stays in the form.
async function stored(task) {
  try {
    await keep(task);
    navigator.storage?.persist?.();
    return true;
  } catch {
    const csrf = await token();
    return Boolean(csrf) && (await upload(task, csrf)) === 'done';
  }
}

// The board link follows the chosen board.
function updateBoardLink() {
  const board = form.querySelector('input[name="board"]:checked')?.value;
  document.getElementById('to-board').href = board ? `/tasks/b/${board}` : '/tasks';
}

// ---- Wiring ----------------------------------------------------------------

// The kept page may have been stored for another board; the address decides.
const wanted = new URLSearchParams(location.search).get('board');
const preset = wanted && form.querySelector(`input[name="board"][value="${CSS.escape(wanted)}"]`);
if (preset) preset.checked = true;
if (!new URLSearchParams(location.search).get('saved')) status.replaceChildren();

setDueDates();
updateBoardLink();
form.addEventListener('submit', save);
form.addEventListener('change', updateBoardLink);
title.addEventListener('input', () => { justSaved = ''; moved = false; });
window.addEventListener('online', sync);
document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible') sync(); });
setInterval(() => { if (document.visibilityState === 'visible') sync(); }, SYNC_MS);
navigator.serviceWorker?.register('/sw.js', { scope: '/' });
render().then(sync);
