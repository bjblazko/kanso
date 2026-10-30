// SPDX-License-Identifier: Apache-2.0
// Service worker: keeps the capture pages, the to-do page and their files on
// the device, so notes and tasks can be written and lists ticked without network. The
// server puts VERSION and PRECACHE (fingerprinted static URLs) above this
// file. Everything else goes to the network untouched.

const CACHE = `kanso-${VERSION}`;
const CAPTURE = '/notes/new';
const TODOS = '/todos';
const TASK_CAPTURE = '/tasks/new';
const TODO_PAGES = /^\/todos(?:\/[a-z2-7]{12})?$/; // one page draws the overview and every list

self.addEventListener('install', (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(CACHE);
    await cache.addAll(PRECACHE);
    await Promise.all([storePage(cache, CAPTURE), storePage(cache, TODOS), storePage(cache, TASK_CAPTURE)]);
    await self.skipWaiting();
  })());
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    for (const key of await caches.keys()) {
      if (key.startsWith('kanso-') && key !== CACHE) await caches.delete(key);
    }
    await self.clients.claim();
  })());
});

self.addEventListener('fetch', (event) => {
  const url = new URL(event.request.url);
  if (url.origin !== self.location.origin || event.request.method !== 'GET') return;
  if (event.request.mode === 'navigate' && url.pathname === CAPTURE) {
    event.respondWith(networkFirst(event.request, CAPTURE));
  } else if (event.request.mode === 'navigate' && url.pathname === TASK_CAPTURE) {
    event.respondWith(networkFirst(event.request, TASK_CAPTURE));
  } else if (event.request.mode === 'navigate' && TODO_PAGES.test(url.pathname)) {
    event.respondWith(networkFirst(event.request, TODOS));
  } else if (url.pathname.startsWith('/static/')) {
    event.respondWith(caches.match(event.request).then((hit) => hit || fetch(event.request)));
  }
});

// storePage keeps a signed-in page; a redirect to the login is not stored.
async function storePage(cache, path) {
  try {
    const response = await fetch(path, { credentials: 'same-origin' });
    if (response.ok && !response.redirected) await cache.put(path, response);
  } catch {
    // Offline while installing: the next visit stores the page.
  }
}

// networkFirst answers from the network and keeps the answer under key; without
// network it answers with what was kept.
async function networkFirst(request, key) {
  const cache = await caches.open(CACHE);
  try {
    const response = await fetch(request);
    if (response.ok && !response.redirected) await cache.put(key, response.clone());
    return response;
  } catch {
    return (await cache.match(key)) || Response.error();
  }
}
