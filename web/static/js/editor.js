// SPDX-License-Identifier: Apache-2.0
// Markdown editor: source and preview, edit-lock heartbeat, saving, uploads.

const PREVIEW_DELAY_MS = 400;
const HEARTBEAT_MS = 60_000;

const form = document.getElementById('editor');
const body = document.getElementById('body');
const preview = document.getElementById('preview');
const status = document.getElementById('editor-status');
const panes = form.querySelector('.editor-panes');
const csrf = document.querySelector('meta[name="csrf"]').content;
const pageID = form.dataset.page;
const lock = form.elements.lock.value;

let dirty = false;
let lockLost = false;

function post(url, data, accept = 'text/html') {
  return fetch(url, {
    method: 'POST',
    headers: { 'X-CSRF-Token': csrf, Accept: accept },
    body: data,
    credentials: 'same-origin',
  });
}

function showStatus(message) {
  status.textContent = message;
  status.hidden = !message;
}

// ---- Preview ---------------------------------------------------------------

let previewTimer;

async function refreshPreview() {
  const data = new URLSearchParams({ page: pageID, body: body.value });
  const response = await post('/api/preview', data);
  if (!response.ok) return;
  preview.innerHTML = await response.text(); // server-rendered, sanitised HTML
  document.dispatchEvent(new CustomEvent('kanso:rendered', { detail: preview }));
}

function schedulePreview() {
  clearTimeout(previewTimer);
  previewTimer = setTimeout(refreshPreview, PREVIEW_DELAY_MS);
}

// ---- Tabs (narrow screens) -------------------------------------------------

function selectTab(name) {
  panes.dataset.show = name;
  for (const tab of form.querySelectorAll('[role="tab"]')) {
    const selected = tab.id === `tab-${name}`;
    tab.setAttribute('aria-selected', String(selected));
    tab.setAttribute('aria-pressed', String(selected));
  }
  if (name === 'preview') refreshPreview();
}

// ---- Lock and saving -------------------------------------------------------

function loseLock() {
  lockLost = true;
  showStatus(form.dataset.msgLost);
}

async function heartbeat() {
  if (lockLost) return;
  const response = await post(`/p/${pageID}/lock/heartbeat`, new URLSearchParams({ lock }));
  if (response.status === 409) loseLock();
}

async function save(event) {
  event?.preventDefault();
  if (lockLost) return loseLock();
  const response = await post(`/p/${pageID}`, new URLSearchParams(new FormData(form)), 'application/json');
  if (response.ok) {
    dirty = false;
    window.location.assign(`/p/${pageID}`);
  } else if (response.status === 409) {
    loseLock();
  } else {
    showStatus(form.dataset.msgFailed);
  }
}

// ---- Uploads ---------------------------------------------------------------

function insertAtCursor(text) {
  const { selectionStart: start, selectionEnd: end, value } = body;
  const before = start > 0 && value[start - 1] !== '\n' ? '\n' : '';
  body.value = value.slice(0, start) + before + text + '\n' + value.slice(end);
  body.selectionStart = body.selectionEnd = start + before.length + text.length + 1;
  body.dispatchEvent(new Event('input'));
}

async function uploadFiles(files) {
  for (const file of files) {
    const data = new FormData();
    data.append('file', file);
    const response = await post(`/p/${pageID}/attachments`, data, 'application/json');
    if (!response.ok) {
      showStatus(form.dataset.msgUploadFailed);
      continue;
    }
    insertAtCursor((await response.json()).markdown);
  }
}

function filesFrom(dataTransfer) {
  return [...(dataTransfer?.files ?? [])];
}

// ---- Wiring ----------------------------------------------------------------

body.addEventListener('input', () => { dirty = true; schedulePreview(); });
form.elements.title.addEventListener('input', () => { dirty = true; });
form.addEventListener('submit', save);
for (const tab of form.querySelectorAll('[role="tab"]')) {
  tab.addEventListener('click', () => selectTab(tab.id.replace('tab-', '')));
}
document.getElementById('file').addEventListener('change', (e) => uploadFiles([...e.target.files]));
body.addEventListener('paste', (e) => {
  const files = filesFrom(e.clipboardData);
  if (files.length) { e.preventDefault(); uploadFiles(files); }
});
body.addEventListener('dragover', (e) => e.preventDefault());
body.addEventListener('drop', (e) => {
  const files = filesFrom(e.dataTransfer);
  if (files.length) { e.preventDefault(); uploadFiles(files); }
});
document.addEventListener('keydown', (e) => {
  if ((e.metaKey || e.ctrlKey) && e.key === 's') save(e);
});
window.addEventListener('beforeunload', (e) => {
  if (dirty) { e.preventDefault(); e.returnValue = form.dataset.msgLeave; }
});
setInterval(heartbeat, HEARTBEAT_MS);
refreshPreview();
