// SPDX-License-Identifier: Apache-2.0
// Klänge: noise and a soft chord, endless or with a sleep timer that fades
// out over the last minute. The choices stay on this device; while it plays
// the reminder (nudge.js) keeps quiet.
import { drone, noise } from './soundscape.js';

const $ = (id) => document.getElementById(id);
const root = $('sounds');
const toggle = $('sounds-toggle');
const user = document.querySelector('.rail')?.dataset.user ?? '';
const key = `kanso.sounds.${user}`;
const fade = 60000;
let play = null; // {ctx, master, noiseGain, droneGain, noise, drone, endsAt, tick, lock, fading}

function prefs() {
  const noiseKind = document.querySelector('input[name="noise"]:checked')?.value ?? '';
  return { noise: noiseKind, noiseVol: Number($('noise-volume').value), drone: $('drone').checked,
    droneVol: Number($('drone-volume').value), timer: Number($('sleep-timer').value), awake: $('awake').checked };
}

function restore() {
  let p = {};
  try { p = JSON.parse(localStorage.getItem(key) || '{}') || {}; } catch { p = {}; }
  const radio = document.querySelector(`input[name="noise"][value="${CSS.escape(p.noise ?? 'pink')}"]`);
  if (radio) radio.checked = true;
  if (Number.isFinite(p.noiseVol)) $('noise-volume').value = p.noiseVol;
  if (typeof p.drone === 'boolean') $('drone').checked = p.drone;
  if (Number.isFinite(p.droneVol)) $('drone-volume').value = p.droneVol;
  if ([0, 15, 30, 45, 60, 90].includes(p.timer)) $('sleep-timer').value = String(p.timer);
  if (typeof p.awake === 'boolean') $('awake').checked = p.awake;
}

function save() { try { localStorage.setItem(key, JSON.stringify(prefs())); } catch { /* private mode */ } }

function busy(until) {
  try {
    if (until) localStorage.setItem(`kanso.busy.${user}`, String(until));
    else localStorage.removeItem(`kanso.busy.${user}`);
  } catch { /* private mode */ }
}

const clock = (ms) => {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
};

function meter(share, mode) {
  const bar = $('sounds-meter');
  const pct = Math.round(Math.min(Math.max(share, 0), 1) * 100);
  bar.firstElementChild.style.width = `${pct}%`;
  bar.setAttribute('aria-valuenow', String(pct));
  bar.dataset.state = mode;
  bar.hidden = prefs().timer === 0;
}

/** (Re)starts the layers to match the choices, keeping what did not change. */
function layers() {
  const p = prefs();
  play.noiseGain.gain.value = p.noiseVol;
  play.droneGain.gain.value = p.droneVol * 1.6;
  if (play.noiseKind !== p.noise) {
    play.noise?.stop();
    play.noise = p.noise ? noise(play.ctx, play.noiseGain, p.noise) : null;
    play.noiseKind = p.noise;
  }
  if (Boolean(play.drone) !== p.drone) {
    play.drone?.stop();
    play.drone = p.drone ? drone(play.ctx, play.droneGain) : null;
  }
}

function setTimer() {
  const minutes = prefs().timer;
  play.endsAt = minutes ? Date.now() + minutes * 60000 : null;
  play.fading = false;
  play.master.gain.cancelScheduledValues(play.ctx.currentTime);
  play.master.gain.setValueAtTime(1, play.ctx.currentTime);
  busy(play.endsAt ?? Date.now() + 12 * 3600000);
}

function show() {
  const minutes = prefs().timer;
  if (!play) {
    $('sounds-time').textContent = minutes ? clock(minutes * 60000) : root.dataset.endless;
    meter(0, 'idle');
    return;
  }
  if (!play.endsAt) {
    $('sounds-time').textContent = root.dataset.endless;
    meter(0, 'running');
    return;
  }
  const left = play.endsAt - Date.now();
  $('sounds-time').textContent = clock(left);
  meter(1 - left / (minutes * 60000), 'running');
  if (left <= fade && !play.fading) { // a slow fade, so the end does not wake anyone
    play.fading = true;
    const t = play.ctx.currentTime;
    play.master.gain.setValueAtTime(1, t);
    play.master.gain.linearRampToValueAtTime(0.0001, t + Math.max(left, 0) / 1000);
  }
  if (left <= 0) stop();
}

async function start() {
  const Ctx = window.AudioContext || window.webkitAudioContext;
  const ctx = new Ctx();
  const master = ctx.createGain();
  const limit = ctx.createDynamicsCompressor(); // noise and chord together never clip
  limit.threshold.value = -12;
  limit.ratio.value = 4;
  master.connect(limit);
  limit.connect(ctx.destination);
  const noiseGain = ctx.createGain();
  const droneGain = ctx.createGain();
  noiseGain.connect(master);
  droneGain.connect(master);
  play = { ctx, master, noiseGain, droneGain, noise: null, drone: null, noiseKind: '' };
  layers();
  setTimer();
  toggle.dataset.running = 'true';
  toggle.querySelector(':scope > span:last-child').textContent = toggle.dataset.stop;
  if (prefs().awake) play.lock = await navigator.wakeLock?.request('screen').catch(() => null);
  if ('mediaSession' in navigator) navigator.mediaSession.metadata = new MediaMetadata({ title: document.title });
  play.tick = setInterval(show, 500);
  show();
}

function stop() {
  if (!play) return;
  clearInterval(play.tick);
  play.noise?.stop();
  play.drone?.stop();
  play.ctx.close();
  play.lock?.release?.();
  play = null;
  busy(0);
  toggle.dataset.running = 'false';
  toggle.querySelector(':scope > span:last-child').textContent = toggle.dataset.start;
  show();
}

restore();
show();
toggle.addEventListener('click', () => (play ? stop() : start()));
root.addEventListener('input', (e) => {
  save();
  if (!play) { show(); return; }
  if (e.target.id === 'sleep-timer') setTimer();
  else layers();
  show();
});
root.addEventListener('change', save);
