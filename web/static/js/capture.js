// SPDX-License-Identifier: Apache-2.0
// Capture: a note goes into an outbox on the device first (IndexedDB, with
// its photos as bytes) and is uploaded when the network allows. Only a
// confirmed upload removes it from the outbox, so nothing is lost on the way.
// Where the outbox cannot be written (e.g. private browsing), the note is
// uploaded at once, and the text stays in the field until that worked.

const SYNC_MS = 30_000;
const ID_ALPHABET = 'abcdefghijklmnopqrstuvwxyz234567';

const form = document.getElementById('capture');
const body = document.getElementById('note-body');
const previews = document.getElementById('previews');
const status = document.getElementById('capture-status');
const outbox = document.getElementById('outbox');
const msg = form.dataset;

let chosen = []; // files picked for the note being written: { file, url }
let justSaved = false;
let needsLogin = false;
let syncing = false;
let syncAgain = false;

// ---- Outbox (IndexedDB) ----------------------------------------------------

const db = new Promise((resolve, reject) => {
  const open = indexedDB.open('kanso', 1);
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
const keep = (note) => inOutbox('readwrite', (s) => s.put(note));
const drop = (id) => inOutbox('readwrite', (s) => s.delete(id));

function newID() {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return Array.from(bytes, (b) => ID_ALPHABET[b % 32]).join('');
}

function titleOf(note) {
  const line = note.body.split('\n').map((l) => l.trim()).find((l) => l !== '');
  return line || msg.msgPhoto;
}

// ---- Status ----------------------------------------------------------------

function plural(one, many, n) {
  return (n === 1 ? one : many).replace('{n}', String(n));
}

async function render() {
  const notes = await allWaiting();
  const waiting = notes.filter((n) => !n.failed);
  const failed = notes.filter((n) => n.failed);
  const parts = [];
  if (justSaved) parts.push(msg.msgSaved);
  if (waiting.length) parts.push(plural(msg.msgWaitingOne, msg.msgWaitingMany, waiting.length));
  if (failed.length) parts.push(plural(msg.msgFailedOne, msg.msgFailedMany, failed.length));
  status.replaceChildren(parts.join(' · '));
  if (needsLogin && waiting.length) {
    const link = Object.assign(document.createElement('a'), { href: msg.loginUrl, textContent: msg.msgLogin });
    status.append(status.textContent ? ' · ' : '', link);
  }
  outbox.hidden = notes.length === 0;
  outbox.querySelector('ul').replaceChildren(...notes.map((n) => {
    const li = document.createElement('li');
    li.textContent = n.failed ? `${titleOf(n)} (${msg.msgFailedMark})` : titleOf(n);
    return li;
  }));
}

// ---- Upload ----------------------------------------------------------------

// token returns a fresh CSRF token, null when signed out, undefined offline.
// A signed-out request is redirected to the sign-in, which in production is
// another site (the OIDC provider); the redirect is not followed but read as
// "signed out".
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

// upload sends one note; the answer says what to do with it next.
async function upload(note, csrf) {
  const data = new FormData();
  data.append('id', note.id);
  data.append('body', note.body);
  for (const f of note.files) data.append('files', new Blob([f.data], { type: f.type }), f.name);
  try {
    const response = await fetch('/notes', {
      method: 'POST', body: data, credentials: 'same-origin',
      headers: { 'X-CSRF-Token': csrf, Accept: 'application/json' },
    });
    if (response.ok) return 'done';
    if (response.status === 400 || response.status === 413) return 'failed';
    if (response.status === 401 || response.status === 403) return 'login';
    return 'later';
  } catch {
    return 'later';
  }
}

// sync uploads waiting notes; a call during an upload runs again afterwards,
// so a network change in the middle is not lost.
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
  const waiting = (await allWaiting()).filter((n) => !n.failed);
  if (waiting.length === 0) return;
  const csrf = await token();
  needsLogin = csrf === null;
  if (!csrf) return;
  for (const note of waiting) {
    const result = await upload(note, csrf);
    if (result === 'done') await drop(note.id);
    else if (result === 'failed') await keep({ ...note, failed: true });
    else {
      needsLogin = result === 'login';
      return;
    }
  }
}

// ---- Writing ---------------------------------------------------------------

function showPreviews() {
  previews.replaceChildren(...chosen.map((c, i) => {
    const li = document.createElement('li');
    if (c.file.type.startsWith('image/')) {
      li.append(Object.assign(document.createElement('img'), { src: c.url, alt: c.file.name }));
    } else {
      li.append(c.file.name);
    }
    const remove = Object.assign(document.createElement('button'), {
      type: 'button', className: 'btn btn-quiet btn-sm', textContent: msg.msgRemove,
    });
    remove.addEventListener('click', () => {
      URL.revokeObjectURL(c.url);
      chosen.splice(i, 1);
      showPreviews();
    });
    li.append(remove);
    return li;
  }));
}

function addFiles(input) {
  for (const file of input.files) chosen.push({ file, url: URL.createObjectURL(file) });
  input.value = ''; // the files now live in `chosen`; the same photo may be picked again
  showPreviews();
}

async function save(event) {
  event.preventDefault();
  if (body.value.trim() === '' && chosen.length === 0) {
    status.replaceChildren(msg.msgEmpty);
    return;
  }
  // Bytes, not Blobs: Safari cannot store Blobs in IndexedDB in private windows.
  const files = await Promise.all(chosen.map(async (c) => ({
    name: c.file.name, type: c.file.type, data: await c.file.arrayBuffer(),
  })));
  const note = { id: newID(), body: body.value, files, created: Date.now(), failed: false };
  if (!(await stored(note))) {
    status.replaceChildren(msg.msgNotSaved);
    return;
  }
  for (const c of chosen) URL.revokeObjectURL(c.url);
  chosen = [];
  body.value = '';
  showPreviews();
  justSaved = true;
  await render();
  await sync();
}

// stored puts the note into the outbox, or uploads it right away if the
// outbox cannot be written; false means it is nowhere yet.
async function stored(note) {
  try {
    await keep(note);
    navigator.storage?.persist?.();
    return true;
  } catch {
    const csrf = await token();
    return Boolean(csrf) && (await upload(note, csrf)) === 'done';
  }
}

// ---- Wiring ----------------------------------------------------------------

form.addEventListener('submit', save);
body.addEventListener('input', () => { justSaved = false; });
for (const id of ['camera', 'pick']) {
  const input = document.getElementById(id);
  input.addEventListener('change', () => addFiles(input));
}
window.addEventListener('online', sync);
document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible') sync(); });
setInterval(() => { if (document.visibilityState === 'visible') sync(); }, SYNC_MS);
navigator.serviceWorker?.register('/sw.js', { scope: '/' });
render().then(sync);
