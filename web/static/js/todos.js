// SPDX-License-Identifier: Apache-2.0
// To-do lists: the page is drawn from a copy on the device (IndexedDB), so it
// opens and works without network. Every change is an operation that is
// applied to the copy at once and kept in an outbox until the server
// confirmed it. The server answers each sync with the current state of all
// lists, on which the still waiting operations are applied again.

const SYNC_MS = 30_000;
const MAX_OPS = 500;
const ID_ALPHABET = 'abcdefghijklmnopqrstuvwxyz234567';
const LIST_PATH = /^\/todos(?:\/([a-z2-7]{12}))?$/;

const root = document.getElementById('todos');
const status = document.getElementById('todos-status');
const msg = JSON.parse(document.getElementById('todo-messages').textContent);
const ME = root.dataset.sub;
const ICONS = [...document.getElementById('todo-icons').content.children];
const titleSuffix = document.title.slice(document.title.indexOf(' · '));

const state = {
  server: [], // lists as the server last sent them
  pending: [], // operations not yet confirmed: { seq, op }
  loaded: false, // a sync has finished (or failed) at least once
  offline: false,
  needsLogin: false,
  editing: null, // { id, text }: the item being edited
  draft: null, // { list, after, text }: a new sub-item being written
  renaming: null, // { id, text }
  newItem: '',
  newList: '',
  focus: null, // data-focus of the field to focus after drawing
};
let drawing = false;
let pointerDown = false;
let drawLater = false;
let syncing = false;
let syncAgain = false;

// ---- Copy and outbox (IndexedDB; in memory where it cannot be opened) ------

const memory = { lists: [], ops: new Map(), seq: 0 };

const db = new Promise((resolve) => {
  try {
    const open = indexedDB.open('kanso-todos', 1);
    open.onupgradeneeded = () => {
      open.result.createObjectStore('lists', { keyPath: 'id' });
      open.result.createObjectStore('ops', { keyPath: 'seq', autoIncrement: true });
    };
    open.onsuccess = () => resolve(open.result);
    open.onerror = () => resolve(null);
  } catch {
    resolve(null);
  }
});

const result = (r) => new Promise((resolve, reject) => {
  r.onsuccess = () => resolve(r.result);
  r.onerror = () => reject(r.error);
});
const finished = (t) => new Promise((resolve, reject) => {
  t.oncomplete = resolve;
  t.onerror = () => reject(t.error);
});

async function objects(name, mode) {
  const d = await db;
  return d && d.transaction(name, mode).objectStore(name);
}

async function loadCopy() {
  const s = await objects('lists', 'readonly');
  return s ? result(s.getAll()) : memory.lists;
}

async function saveCopy(lists) {
  const s = await objects('lists', 'readwrite');
  if (!s) {
    memory.lists = lists;
    return;
  }
  s.clear();
  for (const l of lists) s.put(l);
  await finished(s.transaction);
}

async function loadOps() {
  const s = await objects('ops', 'readonly');
  return s ? result(s.getAll()) : [...memory.ops.values()];
}

async function keepOp(op) {
  const s = await objects('ops', 'readwrite');
  if (s) return { seq: await result(s.add({ op })), op };
  const entry = { seq: ++memory.seq, op };
  memory.ops.set(entry.seq, entry);
  return entry;
}

async function dropOps(seqs) {
  const s = await objects('ops', 'readwrite');
  if (!s) {
    for (const q of seqs) memory.ops.delete(q);
    return;
  }
  for (const q of seqs) s.delete(q);
  await finished(s.transaction);
}

// ---- Operations, the same rules as on the server ----------------------------

function newID() {
  const bytes = crypto.getRandomValues(new Uint8Array(12));
  return Array.from(bytes, (b) => ID_ALPHABET[b % 32]).join('');
}

function applyOp(lists, op) {
  if (op.kind === 'create' && !lists.has(op.list)) {
    lists.set(op.list, { id: op.list, name: op.text, mine: true, items: [], shared: [], sharedNames: [] });
  }
  const l = lists.get(op.list);
  if (!l) return;
  if (op.kind === 'rename') l.name = op.text;
  else if (op.kind === 'add') addItem(l, op);
  else if (op.kind === 'delete') deleteItem(l, op.item);
  else setField(l, op);
  l.updated = new Date(op.at).toISOString();
}

function addItem(l, op) {
  if (l.items.some((it) => it.id === op.item)) return;
  let at = l.items.findIndex((it) => it.id === op.after) + 1;
  if (!op.after || at === 0) at = l.items.length;
  l.items.splice(at, 0, { id: op.item, text: op.text, done: false, sub: Boolean(op.sub) && at > 0 });
}

function setField(l, op) {
  const i = l.items.findIndex((it) => it.id === op.item);
  if (i < 0) return;
  if (op.kind === 'text') l.items[i].text = op.text;
  if (op.kind === 'done') l.items[i].done = op.done;
  if (op.kind === 'sub' && (i > 0 || !op.sub)) l.items[i].sub = op.sub;
}

// deleteItem removes an item; a main item takes its sub-items along.
function deleteItem(l, id) {
  const i = l.items.findIndex((it) => it.id === id);
  if (i < 0) return;
  let end = i + 1;
  while (!l.items[i].sub && l.items[end]?.sub) end++;
  l.items.splice(i, end - i);
}

// lists returns the server state with the waiting operations applied, most
// recently changed first.
function lists() {
  const byID = new Map(structuredClone(state.server).map((l) => [l.id, l]));
  for (const { op } of state.pending) applyOp(byID, op);
  const time = (l) => Date.parse(l.updated) || 0;
  return [...byID.values()].sort((a, b) => time(b) - time(a));
}

async function change(fields) {
  const op = { ...fields, id: newID(), at: Date.now() };
  state.pending.push(await keepOp(op));
  navigator.storage?.persist?.();
  draw();
  sync();
}

// ---- Sync ------------------------------------------------------------------

// token returns a fresh CSRF token, null when signed out, undefined offline.
// A signed-out request is redirected to the sign-in (in production another
// site); the redirect is not followed but read as "signed out".
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

// request answers the server's lists, null when signed out, undefined when
// the server cannot be reached.
async function request(ops) {
  const csrf = ops.length ? await token() : '';
  if (!csrf && ops.length) return csrf;
  const init = ops.length
    ? { method: 'POST', body: JSON.stringify({ ops }), headers: { 'Content-Type': 'application/json', 'X-CSRF-Token': csrf } }
    : {};
  try {
    const response = await fetch(ops.length ? '/api/todos/sync' : '/api/todos', { ...init, credentials: 'same-origin', redirect: 'manual' });
    if (response.type === 'opaqueredirect' || response.status === 401 || response.status === 403) return null;
    return response.ok ? await response.json() : undefined;
  } catch {
    return undefined;
  }
}

async function syncOnce() {
  const sent = state.pending.slice(0, MAX_OPS);
  const answer = await request(sent.map((p) => p.op));
  state.offline = answer === undefined;
  state.needsLogin = answer === null;
  if (!answer) return;
  state.server = answer.lists;
  await saveCopy(answer.lists);
  const done = new Set(sent.map((p) => p.seq));
  await dropOps([...done]);
  state.pending = state.pending.filter((p) => !done.has(p.seq));
  if (state.pending.length > 0 && sent.length === MAX_OPS) syncAgain = true;
}

// sync sends waiting operations; a call during a sync runs again afterwards.
async function sync() {
  if (syncing) {
    syncAgain = true;
    return;
  }
  syncing = true;
  try {
    do {
      syncAgain = false;
      await syncOnce();
    } while (syncAgain);
  } finally {
    syncing = false;
    state.loaded = true;
    draw();
  }
}

// ---- Drawing ---------------------------------------------------------------

function h(tag, attrs = {}, ...children) {
  const el = document.createElement(tag);
  for (const [k, v] of Object.entries(attrs)) {
    if (v === false || v == null) continue;
    if (k.startsWith('on')) el.addEventListener(k.slice(2), v);
    else if (k === 'className') el.className = v;
    else el.setAttribute(k, v === true ? '' : v);
  }
  el.append(...children.flat().filter((c) => c != null && c !== false));
  return el;
}

const fill = (text, vars) => text.replace(/\{(\w+)\}/g, (_, k) => String(vars[k]));
const button = (label, onclick, kind = 'btn-secondary') =>
  h('button', { type: 'button', className: `btn ${kind} btn-sm`, onclick }, label);

function draw() {
  drawing = true;
  const all = lists();
  const id = location.pathname.match(LIST_PATH)?.[1];
  const list = id && all.find((l) => l.id === id);
  root.replaceChildren(...(id ? listPage(list) : overview(all)));
  document.title = (list ? list.name : msg.title) + titleSuffix;
  drawStatus();
  refocus();
  drawing = false;
}

// drawSoon draws now, or, while a finger or mouse button is down, after the
// click it leads to: a field closing on blur must not swap the element under
// the pointer, or the click (on a link, a checkbox) would be lost.
function drawSoon() {
  if (pointerDown) drawLater = true;
  else draw();
}

function refocus() {
  const field = state.focus && root.querySelector(`[data-focus="${state.focus}"]`);
  if (!field) return;
  field.focus();
  field.setSelectionRange?.(field.value.length, field.value.length);
}

function drawStatus() {
  const n = state.pending.length;
  status.replaceChildren(n ? fill(n === 1 ? msg.waiting_one : msg.waiting_many, { n }) : '');
  if (state.needsLogin && n) {
    status.append(' · ', h('a', { href: `/auth/login?next=${encodeURIComponent(location.pathname)}` }, msg.sync_login));
  }
}

// field is a text input that keeps its value in state[key] across redraws.
// Browsers differ in whether and when a field removed by drawing fires blur;
// blur handlers therefore ignore fields that are no longer on the page.
function field(key, attrs) {
  return h('input', {
    autocomplete: 'off', 'data-focus': key, ...attrs,
    oninput: (e) => { setText(key, e.target.value); },
  });
}

function setText(key, value) {
  if (key === 'edit') state.editing.text = value;
  else if (key === 'draft') state.draft.text = value;
  else if (key === 'rename') state.renaming.text = value;
  else state[key === 'new-item' ? 'newItem' : 'newList'] = value;
}

// ---- Overview --------------------------------------------------------------

function overview(all) {
  const form = h('form', { className: 'form form-row todo-new', onsubmit: createList },
    h('div', { className: 'field' }, h('label', { for: 'new-list' }, msg.new_list),
      field('new-list', { id: 'new-list', maxlength: 80, value: state.newList })),
    h('button', { className: 'btn btn-primary', type: 'submit' }, msg.create));
  let body = h('ul', { className: 'list-plain todo-lists' }, all.map(listRow));
  if (all.length === 0) body = h('p', { className: 'empty' }, state.offline ? msg.offline_empty : (state.loaded ? msg.empty : ''));
  return [h('h1', {}, msg.title), form, body];
}

function counts(l) {
  const done = l.items.filter((it) => it.done).length;
  return { done, total: l.items.length, percent: l.items.length ? Math.floor((100 * done) / l.items.length) : 0 };
}

function progress(l) {
  const parts = [fill(msg.progress, counts(l))];
  if (!l.mine) parts.push(fill(msg.from, { name: l.ownerName }));
  return parts.join(' · ');
}

// audience says who sees the list: its owner and everyone it is shared with,
// the viewer as "you"; drawn like the server's line on the other pages. On the
// list itself its owner gets a link to change who sees it.
function audience(l, offerChange = false) {
  const you = (name, i) => (l.shared[i]?.user === ME ? msg.you : name);
  const names = [l.mine ? msg.you : l.ownerName, ...l.sharedNames.map(you)];
  const priv = names.length === 1;
  const icon = ICONS[priv ? 0 : 1].cloneNode(true);
  const change = offerChange && l.mine
    ? [' · ', h('a', { href: `/todos/${l.id}/share` }, msg.change_sharing)] : [];
  return h('p', { className: 'meta audience' }, icon,
    h('span', {}, priv ? msg.only_you : fill(msg.visible_to, { names: names.join(', ') })), ...change);
}

// meter is a bar whose shape alone tells empty, partial and full apart; a
// list without items has nothing to measure and gets none. Rounding down keeps
// 100 for a list that is really done.
function meter(l, attrs = {}) {
  const { done, total, percent } = counts(l);
  if (!total) return null;
  const state = done === 0 ? 'empty' : (done === total ? 'full' : 'partial');
  const bar = h('span', { className: 'todo-meter-fill' });
  bar.style.width = `${percent}%`;
  return h('span', { className: 'todo-meter', 'data-state': state, ...attrs }, bar);
}

function listRow(l) {
  return h('li', {}, h('a', { href: `/todos/${l.id}` }, l.name), h('br'),
    h('span', { className: 'meta todo-summary' }, meter(l, { 'aria-hidden': 'true' }), progress(l)), audience(l));
}

// listProgress is the bar with its percentage between heading and items.
function listProgress(l) {
  const { percent } = counts(l);
  const bar = meter(l, { role: 'progressbar', 'aria-label': msg.progress_label,
    'aria-valuemin': 0, 'aria-valuemax': 100, 'aria-valuenow': percent });
  if (!bar) return null;
  return h('div', { className: 'todo-progress' }, bar, h('span', { className: 'todo-percent' }, fill(msg.percent, { n: percent })));
}

async function createList(event) {
  event.preventDefault();
  const name = state.newList.trim();
  if (!name) return;
  const id = newID();
  state.newList = '';
  history.pushState(null, '', `/todos/${id}`);
  state.focus = 'new-item';
  await change({ kind: 'create', list: id, text: name });
}

// ---- One list --------------------------------------------------------------

function listPage(l) {
  const back = h('p', { className: 'breadcrumb' }, h('a', { href: '/todos' }, msg.all));
  if (!l) return [back, state.loaded ? h('p', { className: 'empty' }, msg.not_found) : ''];
  return [back, heading(l), listProgress(l), h('p', { className: 'meta' }, progress(l)), audience(l, true),
    h('ul', { className: 'todo-items' }, rows(l)), addForm(l), l.mine ? ownerActions(l) : ''];
}

function heading(l) {
  if (state.renaming?.id !== l.id) return h('h1', {}, l.name);
  const save = (event) => {
    event.preventDefault();
    const name = state.renaming.text.trim();
    state.renaming = null;
    if (name && name !== l.name) change({ kind: 'rename', list: l.id, text: name });
    else draw();
  };
  return h('form', { className: 'form form-row', onsubmit: save },
    h('div', { className: 'field' }, h('label', { for: 'rename' }, msg.name),
      field('rename', { id: 'rename', maxlength: 80, value: state.renaming.text })),
    h('button', { className: 'btn btn-secondary', type: 'submit' }, msg.save),
    button(msg.cancel, () => { state.renaming = null; draw(); }, 'btn-quiet'));
}

function rows(l) {
  const out = [];
  l.items.forEach((it, i) => {
    out.push(itemRow(l, it, i));
    if (state.draft?.list === l.id && state.draft.after === it.id) out.push(draftRow(l));
  });
  return out;
}

function itemRow(l, it, i) {
  const editing = state.editing?.id === it.id;
  const cls = ['todo-item', it.sub && 'sub', it.done && 'done', editing && 'editing'].filter(Boolean).join(' ');
  const box = h('label', { className: 'todo-check' }, h('input', {
    type: 'checkbox', 'aria-label': it.text, checked: it.done,
    onchange: (e) => change({ kind: 'done', list: l.id, item: it.id, done: e.target.checked }),
  }));
  if (!editing) {
    return h('li', { className: cls }, box,
      h('button', { type: 'button', className: 'todo-text', onclick: () => startEdit(l, it) }, it.text));
  }
  return h('li', { className: cls }, box, field('edit', {
    className: 'todo-edit', 'aria-label': msg.item, maxlength: 500, value: state.editing.text,
    onkeydown: (e) => editKey(e, l), onblur: (e) => { if (!drawing && e.target.isConnected) { commitEdit(l); drawSoon(); } },
  }), tools(l, it, i));
}

// Tool buttons keep the focus in the text field, so pressing one does not
// first close the editing through the field's blur.
function tools(l, it, i) {
  const keep = (e) => e.preventDefault();
  const act = (fields) => () => {
    commitEdit(l);
    change({ ...fields, list: l.id, item: it.id });
  };
  return h('div', { className: 'todo-tools', onpointerdown: keep, onmousedown: keep },
    it.sub ? button(msg.outdent, act({ kind: 'sub', sub: false })) : (i > 0 && button(msg.indent, act({ kind: 'sub', sub: true }))),
    button(msg.add_sub, () => startDraft(l, it)),
    button(msg.delete_item, act({ kind: 'delete' }), 'btn-destructive'));
}

function startEdit(l, it) {
  commitEdit(l);
  state.draft = null;
  state.editing = { id: it.id, text: it.text };
  state.focus = 'edit';
  draw();
}

// commitEdit saves the edited text; an emptied item is deleted.
function commitEdit(l) {
  const e = state.editing;
  state.editing = null;
  state.focus = null;
  const it = e && l.items.find((x) => x.id === e.id);
  if (!it) return;
  const text = e.text.trim();
  if (!text) change({ kind: 'delete', list: l.id, item: it.id });
  else if (text !== it.text) change({ kind: 'text', list: l.id, item: it.id, text });
}

function editKey(event, l) {
  if (event.key === 'Enter') {
    event.preventDefault();
    commitEdit(l);
    draw();
  } else if (event.key === 'Escape') {
    state.editing = null;
    state.focus = null;
    draw();
  }
}

// startDraft opens a field for a new sub-item below it and its sub-items.
function startDraft(l, it) {
  commitEdit(l);
  let i = l.items.indexOf(it);
  while (!it.sub && l.items[i + 1]?.sub) i++;
  state.draft = { list: l.id, after: l.items[i].id, text: '' };
  state.focus = 'draft';
  draw();
}

function draftRow(l) {
  const add = () => {
    const text = state.draft.text.trim();
    if (!text) return;
    const item = newID();
    const after = state.draft.after;
    state.draft = { list: l.id, after: item, text: '' };
    change({ kind: 'add', list: l.id, item, after, text, sub: true });
  };
  const close = () => { state.draft = null; state.focus = null; drawSoon(); };
  return h('li', { className: 'todo-item sub draft' }, field('draft', {
    className: 'todo-edit', 'aria-label': msg.item, maxlength: 500, value: state.draft.text,
    onkeydown: (e) => {
      if (e.key === 'Enter') { e.preventDefault(); add(); }
      if (e.key === 'Escape') close();
    },
    onblur: (e) => { if (!drawing && e.target.isConnected) { add(); close(); } },
  }));
}

function addForm(l) {
  const submit = (event) => {
    event.preventDefault();
    const text = state.newItem.trim();
    if (!text) return;
    state.newItem = '';
    state.focus = 'new-item';
    change({ kind: 'add', list: l.id, item: newID(), after: '', text, sub: false });
  };
  return h('form', { className: 'form form-row todo-add', onsubmit: submit },
    h('div', { className: 'field' }, h('label', { for: 'new-item' }, msg.new_item),
      field('new-item', { id: 'new-item', maxlength: 500, value: state.newItem })),
    h('button', { className: 'btn btn-primary', type: 'submit' }, msg.add));
}

function ownerActions(l) {
  const rename = () => { state.renaming = { id: l.id, text: l.name }; state.focus = 'rename'; draw(); };
  return h('div', { className: 'actions' },
    button(msg.rename, rename),
    h('a', { className: 'btn btn-secondary btn-sm', href: `/todos/${l.id}/share` }, msg.share),
    h('a', { className: 'btn btn-quiet btn-sm', href: `/todos/${l.id}/delete` }, msg.delete_list));
}

// ---- Wiring ----------------------------------------------------------------

// Links between the overview and a list stay on this page.
root.addEventListener('click', (event) => {
  const link = event.target.closest('a');
  if (!link || event.ctrlKey || event.metaKey || event.shiftKey || !LIST_PATH.test(link.pathname)) return;
  event.preventDefault();
  history.pushState(null, '', link.pathname);
  Object.assign(state, { editing: null, draft: null, renaming: null, focus: null });
  draw();
  window.scrollTo(0, 0);
});
root.addEventListener('focusin', (e) => { state.focus = e.target.dataset?.focus || null; });
root.addEventListener('focusout', () => { if (!drawing) state.focus = null; });
document.addEventListener('pointerdown', () => { pointerDown = true; }, true);
document.addEventListener('pointerup', () => {
  pointerDown = false;
  setTimeout(() => { // after the click this pointer makes
    if (drawLater) {
      drawLater = false;
      draw();
    }
  });
}, true);
window.addEventListener('popstate', draw);
window.addEventListener('online', sync);
document.addEventListener('visibilitychange', () => { if (document.visibilityState === 'visible') sync(); });
setInterval(() => { if (document.visibilityState === 'visible') sync(); }, SYNC_MS);
navigator.serviceWorker?.register('/sw.js', { scope: '/' });

state.server = await loadCopy();
state.pending = await loadOps();
draw();
sync();
