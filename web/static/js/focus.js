// SPDX-License-Identifier: Apache-2.0
// Fokus, a Pomodoro timer: 25 minutes focus, 5 minutes break, a long break
// after four rounds. It counts against an end time, so a throttled background
// tab and a reload stay exact. Settings and the running phase stay on this
// device (localStorage); the end rings in this tab only (ADR-021).
const root = document.getElementById('focus');
const user = document.querySelector('.rail')?.dataset.user ?? '';
const key = `kanso.focus.${user}`;
const rounds = 4;
const phases = ['focus', 'short', 'long'];
const defaults = { minutes: { focus: 25, short: 5, long: 15 }, phase: 'focus', round: 1, endsAt: null, left: null };
const text = root.dataset;
const el = (id) => document.getElementById(id);
const pageTitle = document.title;
let state = load();
let audio = null;

function load() {
  try {
    const saved = JSON.parse(localStorage.getItem(key) || '{}') || {};
    const s = { ...defaults, ...saved, minutes: { ...defaults.minutes, ...saved.minutes } };
    if (!phases.includes(s.phase)) s.phase = 'focus';
    s.round = Math.min(Math.max(Number(s.round) || 1, 1), rounds);
    return s;
  } catch {
    return structuredClone(defaults);
  }
}

function save() {
  try { localStorage.setItem(key, JSON.stringify(state)); } catch { /* private mode: nothing to remember */ }
}

const full = (phase) => state.minutes[phase] * 60000;
const running = () => state.endsAt !== null;
const remaining = () => (running() ? state.endsAt - Date.now() : state.left ?? full(state.phase));

function clock(ms) {
  const s = Math.max(0, Math.ceil(ms / 1000));
  return `${String(Math.floor(s / 60)).padStart(2, '0')}:${String(s % 60).padStart(2, '0')}`;
}

// ---- What the person hears and sees at the end -------------------------------

/** Three short tones from the browser itself; no sound file. */
function ring() {
  if (!audio) return;
  const t = audio.currentTime;
  for (let i = 0; i < 3; i += 1) {
    const osc = audio.createOscillator();
    const gain = audio.createGain();
    osc.type = 'sine';
    osc.frequency.value = 880;
    gain.gain.setValueAtTime(0.0001, t + i * 0.35);
    gain.gain.exponentialRampToValueAtTime(0.3, t + i * 0.35 + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, t + i * 0.35 + 0.25);
    osc.connect(gain);
    gain.connect(audio.destination);
    osc.start(t + i * 0.35);
    osc.stop(t + i * 0.35 + 0.3);
  }
}

function notify(body) {
  if (!('Notification' in window) || Notification.permission !== 'granted') return;
  try {
    new Notification(text.phaseFocus, { body, tag: 'kanso-focus' });
  } catch { // Android Chrome: only through the service worker
    navigator.serviceWorker?.getRegistration().then((r) => r?.showNotification(text.phaseFocus, { body, tag: 'kanso-focus' }));
  }
}

/** Asks once, on the person's own tap, and unlocks sound for this tab. */
function prepare() {
  const Ctx = window.AudioContext || window.webkitAudioContext;
  if (!audio && Ctx) audio = new Ctx();
  audio?.resume?.();
  if ('Notification' in window && Notification.permission === 'default') {
    Notification.requestPermission().then(render, render);
  }
}

// ---- Phases -----------------------------------------------------------------

function next() {
  if (state.phase === 'focus') {
    state.phase = state.round >= rounds ? 'long' : 'short';
  } else {
    state.round = state.phase === 'long' ? 1 : state.round + 1;
    state.phase = 'focus';
  }
  state.endsAt = null;
  state.left = null;
}

function finish() {
  const body = state.phase === 'focus' ? text.doneFocus : text.doneBreak;
  next();
  save();
  ring();
  notify(body);
}

function toggle() {
  if (running()) {
    state.left = remaining();
    state.endsAt = null;
  } else {
    prepare();
    state.endsAt = Date.now() + remaining();
    state.left = null;
  }
  save();
  render();
}

function setMinutes(phase, input) {
  const n = Math.min(Math.max(Math.round(Number(input.value)) || defaults.minutes[phase], 1), 120);
  input.value = n;
  state.minutes[phase] = n;
  if (phase === state.phase && !running() && state.left === null) render();
  save();
  render();
}

// ---- Drawing ----------------------------------------------------------------

function notifyText() {
  if (!('Notification' in window)) return text.notifyOff;
  return { granted: text.notifyOn, denied: text.notifyOff }[Notification.permission] ?? text.notifyAsk;
}

function render() {
  const phaseName = text[`phase${state.phase[0].toUpperCase()}${state.phase.slice(1)}`];
  const left = clock(remaining());
  el('focus-phase').textContent = phaseName;
  el('focus-round').textContent = text.round.replace('%d', state.round);
  el('focus-time').textContent = left;
  const paused = !running() && state.left !== null;
  const label = running() ? text.pause : paused ? text.resume : text.start;
  const toggleButton = el('focus-toggle');
  toggleButton.querySelector('span').textContent = label;
  toggleButton.querySelector('use, svg')?.classList.toggle('is-running', running());
  document.title = running() || paused ? `${left} · ${phaseName}` : pageTitle;
  el('focus-notify').textContent = notifyText();
  for (const p of phases) el(`focus-minutes-${p}`).value = state.minutes[p];
}

function tick() {
  if (running() && remaining() <= 0) finish();
  render();
}

el('focus-toggle').addEventListener('click', toggle);
el('focus-skip').addEventListener('click', () => { next(); save(); render(); });
el('focus-reset').addEventListener('click', () => {
  state = { ...structuredClone(defaults), minutes: state.minutes };
  save();
  render();
});
for (const p of phases) el(`focus-minutes-${p}`).addEventListener('change', (e) => setMinutes(p, e.target));
window.addEventListener('storage', (e) => { if (e.key === key) { state = load(); render(); } });
setInterval(tick, 250);
tick();
