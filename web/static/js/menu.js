// SPDX-License-Identifier: Apache-2.0
// The rail as this person wants it on this device (ADR-020): order and hidden
// parts live in localStorage, not in a cookie and not on the server. A plain
// script right after the rail, so the order is right before the page is drawn.
// On phones more than five parts become four and "Mehr".
(function () {
  'use strict';
  const nav = document.querySelector('.rail');
  if (!nav) return;
  const key = 'kanso.menu.' + nav.dataset.user;
  const bar = nav.querySelector('.rail-bar');
  const moreList = nav.querySelector('.rail-more-list');
  const more = nav.querySelector('.rail-more');
  const defaults = [...bar.querySelectorAll(':scope > li[data-key]')].map((li) => li.dataset.key);
  const fixed = 'start'; // home stays: it can be moved, not hidden
  const inBar = 4; // on phones, with "Mehr" as the fifth

  const known = (list) => (Array.isArray(list) ? list.filter((k) => defaults.includes(k)) : []);

  /** The saved settings, cleaned: unknown parts dropped, new ones appended. */
  function read() {
    let saved = {};
    try { saved = JSON.parse(localStorage.getItem(key) || '{}') || {}; } catch { saved = {}; }
    const order = [...new Set(known(saved.order))];
    for (const k of defaults) if (!order.includes(k)) order.push(k);
    const hidden = [...new Set(known(saved.hidden))].filter((k) => k !== fixed);
    return { order, hidden };
  }

  function write(prefs) {
    try { localStorage.setItem(key, JSON.stringify(prefs)); } catch { /* private mode: nothing to remember */ }
    apply(prefs);
  }

  function reset() {
    try { localStorage.removeItem(key); } catch { /* nothing stored */ }
    apply(read());
  }

  /** Puts list's parts in order, hides the hidden ones and says where each goes on phones. */
  function arrange(list, prefs, visible, overflow) {
    const anchor = list.querySelector(':scope > li:not([data-key])');
    const items = new Map([...list.querySelectorAll(':scope > li[data-key]')].map((li) => [li.dataset.key, li]));
    for (const k of prefs.order) {
      const li = items.get(k);
      list.insertBefore(li, anchor);
      li.hidden = prefs.hidden.includes(k);
      li.dataset.slot = overflow && visible.indexOf(k) >= inBar ? 'more' : 'bar';
    }
  }

  function apply(prefs) {
    const visible = prefs.order.filter((k) => !prefs.hidden.includes(k));
    const overflow = visible.length > inBar + 1;
    arrange(bar, prefs, visible, overflow);
    arrange(moreList, prefs, visible, overflow);
    nav.classList.toggle('rail-has-more', overflow);
    const current = moreList.querySelector('li[data-slot="more"]:not([hidden]) a[aria-current]');
    more.classList.toggle('rail-more-current', Boolean(current));
  }

  // "Mehr" closes on a tap elsewhere and on Escape, like the user menu.
  const details = more.querySelector('details');
  document.addEventListener('click', (e) => { if (!more.contains(e.target)) details.open = false; });
  document.addEventListener('keydown', (e) => { if (e.key === 'Escape') details.open = false; });

  window.KansoMenu = { read, write, reset, defaults, fixed };
  apply(read());
})();
