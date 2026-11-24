// SPDX-License-Identifier: Apache-2.0
// After a while at it, Kanso suggests a short step between two things (Zwischendurch).
// Only while a Kanso tab is open (ADR-021): a quiet bar at the top of the page,
// and a notification if the tab is in the background and notifications are
// allowed. Not during a Fokus round or a meditation. Settings and the clock
// live on this device (localStorage), shared by all Kanso tabs.
(function () {
  'use strict';
  const bar = document.getElementById('nudge');
  const user = document.querySelector('.rail')?.dataset.user;
  if (!bar || !user) return;
  const key = `kanso.nudge.${user}`;
  const minute = 60000;
  const later = 15 * minute;

  function read() {
    try {
      const s = JSON.parse(localStorage.getItem(key) || '{}') || {};
      return { every: Number.isFinite(s.every) ? s.every : 60, since: s.since || 0, snooze: s.snooze || 0, told: s.told || 0 };
    } catch { return { every: 60, since: 0, snooze: 0, told: 0 }; }
  }

  function write(s) {
    try { localStorage.setItem(key, JSON.stringify(s)); } catch { /* private mode: nothing to remember */ }
    check();
  }

  /** A Fokus round or a meditation is running in some tab of this device. */
  function busy(now) {
    try {
      const focus = JSON.parse(localStorage.getItem(`kanso.focus.${user}`) || '{}');
      if (focus && focus.endsAt && focus.endsAt > now) return true;
      return Number(localStorage.getItem(`kanso.busy.${user}`) || 0) > now;
    } catch { return false; }
  }

  function due(s, now) {
    return s.every > 0 && s.since > 0 && now - s.since >= s.every * minute && now >= s.snooze && !busy(now);
  }

  function check() {
    const s = read();
    const now = Date.now();
    if (!s.since) { s.since = now; try { localStorage.setItem(key, JSON.stringify(s)); } catch { /* */ } }
    const show = due(s, now);
    bar.hidden = !show;
    if (show && document.hidden && s.told !== s.since && 'Notification' in window && Notification.permission === 'granted') {
      s.told = s.since;
      try { localStorage.setItem(key, JSON.stringify(s)); } catch { /* */ }
      try { new Notification(bar.dataset.title, { body: bar.dataset.body, tag: 'kanso-nudge' }); } catch { /* Android: shown in the page only */ }
    }
  }

  /** Starts the count again: after a break, a step between, or a dismissal. */
  function reset() { write({ ...read(), since: Date.now(), snooze: 0 }); }

  bar.addEventListener('click', (e) => {
    const what = e.target.closest('[data-nudge]')?.dataset.nudge;
    if (what === 'later') write({ ...read(), snooze: Date.now() + later });
    if (what === 'close') reset();
  });
  window.addEventListener('storage', (e) => { if (e.key === key) check(); });

  // The setting on the Selfcare page.
  const every = document.getElementById('nudge-every');
  if (every) {
    every.value = String(read().every);
    every.addEventListener('change', () => write({ ...read(), every: Number(every.value), since: Date.now() }));
  }

  window.KansoNudge = { reset };
  setInterval(check, 30000);
  check();
})();
