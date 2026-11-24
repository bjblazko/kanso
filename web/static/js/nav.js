// SPDX-License-Identifier: Apache-2.0
// On narrow screens the navigation starts folded, so the page comes first.
// On wide screens it stays as the person left it: a cookie tells the server,
// so the next page is drawn folded or open without a jump.
const nav = document.querySelector('.sidenav > details');
const wide = window.matchMedia('(min-width: 60rem)');
if (nav && !wide.matches) nav.open = false;
nav?.addEventListener('toggle', () => {
  if (!wide.matches) return;
  nav.closest('.shell')?.classList.toggle('shell-folded', !nav.open);
  const value = nav.open ? 'open' : 'closed';
  document.cookie = `kanso_sidenav=${value}; Path=/; Max-Age=31536000; SameSite=Lax`;
});

// "Drucken" in a toolbar: the browser's own print dialog, which can also save a PDF.
document.addEventListener('click', (e) => { if (e.target.closest('[data-print]')) window.print(); });
