// SPDX-License-Identifier: Apache-2.0
// A guided meditation: one small clip per line, played with the pauses
// between them. Every clip and both gongs are put on the AudioContext's clock
// at the start, so the timing holds even when the browser slows timers in a
// background tab. The gong is made in the browser (gong.js). While it plays the
// screen stays on (Wake Lock), since phones may stop sound when they lock.
import { gong } from './gong.js';
import { plan } from './meditation-plan.js';

const root = document.getElementById('meditation');
const toggle = document.getElementById('med-toggle');
const timeEl = document.getElementById('med-time');
const nowEl = document.getElementById('med-now');
const lines = [...document.querySelectorAll('#med-lines > li')];
const minutes = Number(root.dataset.minutes);
const bellLength = Number(root.dataset.bell);
let session = null; // {ctx, schedule: {t0, p}, tick, lock}

const clock = (s) => {
  const t = Math.max(0, Math.ceil(s));
  return `${String(Math.floor(t / 60)).padStart(2, '0')}:${String(t % 60).padStart(2, '0')}`;
};

const bar = document.getElementById('med-meter');

/** The bar under the time: share of the meditation done (0…1). */
function meter(share, mode) {
  const pct = Math.round(Math.min(Math.max(share, 0), 1) * 100);
  bar.firstElementChild.style.width = `${pct}%`;
  bar.setAttribute('aria-valuenow', String(pct));
  bar.dataset.state = mode;
}

function label(text) {
  toggle.querySelector(':scope > span:last-child').textContent = text;
  toggle.dataset.running = String(text === toggle.dataset.stop);
}

function voice() { return root.querySelector('input[name="voice"]:checked').value; }

async function load(ctx, who) {
  const data = await Promise.all(lines.map(async (li) => {
    const response = await fetch(li.dataset[`clip${who[0].toUpperCase()}${who.slice(1)}`]);
    if (!response.ok) throw new Error(response.statusText);
    return ctx.decodeAudioData(await response.arrayBuffer());
  }));
  return data;
}

function schedule(ctx, buffers) {
  const pauses = lines.map((li) => Number(li.dataset.pause));
  const fill = lines.findIndex((li) => 'fill' in li.dataset);
  const p = plan(buffers.map((b) => b.duration), pauses, fill, minutes, bellLength);
  const t0 = ctx.currentTime + 0.3;
  gong(ctx, t0, bellLength);
  buffers.forEach((buffer, i) => {
    const src = ctx.createBufferSource();
    src.buffer = buffer;
    src.connect(ctx.destination);
    src.start(t0 + p.starts[i]);
  });
  gong(ctx, t0 + p.endBell, bellLength);
  return { t0, p };
}

function show() {
  const { t0, p } = session.schedule;
  const elapsed = session.ctx.currentTime - t0;
  timeEl.textContent = clock(p.total - elapsed);
  meter(elapsed / p.total, 'running');
  let current = -1;
  p.starts.forEach((s, i) => { if (elapsed >= s) current = i; });
  lines.forEach((li, i) => li.classList.toggle('med-current', i === current));
  nowEl.textContent = current >= 0 ? lines[current].textContent : '';
  if (elapsed >= p.total) stop(root.dataset.done);
}

async function start() {
  const Ctx = window.AudioContext || window.webkitAudioContext;
  const ctx = new Ctx();
  session = { ctx };
  label(toggle.dataset.stop);
  root.querySelectorAll('input[name="voice"]').forEach((r) => { r.disabled = true; });
  nowEl.textContent = root.dataset.loading;
  try {
    const buffers = await load(ctx, voice());
    if (session?.ctx !== ctx) return; // stopped while loading
    session.schedule = schedule(ctx, buffers);
    busy(Date.now() + session.schedule.p.total * 1000);
    session.lock = await navigator.wakeLock?.request('screen').catch(() => null);
    if ('mediaSession' in navigator) navigator.mediaSession.metadata = new MediaMetadata({ title: document.title });
    session.tick = setInterval(show, 250);
    show();
  } catch {
    stop(root.dataset.failed);
  }
}

/** Tells the reminder (nudge.js) in every tab to wait until the meditation is over. */
function busy(until) {
  const key = `kanso.busy.${document.querySelector('.rail')?.dataset.user}`;
  try { if (until) localStorage.setItem(key, String(until)); else localStorage.removeItem(key); } catch { /* private mode */ }
}

function stop(message = '') {
  if (!session) return;
  busy(0);
  clearInterval(session.tick);
  session.ctx.close();
  session.lock?.release?.();
  session = null;
  label(toggle.dataset.start);
  root.querySelectorAll('input[name="voice"]').forEach((r) => { r.disabled = false; });
  lines.forEach((li) => li.classList.remove('med-current'));
  timeEl.textContent = clock(minutes * 60);
  meter(0, 'idle');
  nowEl.textContent = message;
}

toggle.addEventListener('click', () => (session ? stop() : start()));
