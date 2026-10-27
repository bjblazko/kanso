// SPDX-License-Identifier: Apache-2.0
// Service worker: keeps the capture page and its files on the device, so a
// note can be written without network. The server puts VERSION and PRECACHE
// (fingerprinted static URLs) above this file. Everything else goes to the
// network untouched.

const CACHE = `kanso-${VERSION}`;
const CAPTURE = '/notes/new';

self.addEventListener('install', (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(CACHE);
    await cache.addAll(PRECACHE);
    await storeCapturePage(cache);
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
    event.respondWith(captureNetworkFirst(event.request));
  } else if (url.pathname.startsWith('/static/')) {
    event.respondWith(caches.match(event.request).then((hit) => hit || fetch(event.request)));
  }
});

// storeCapturePage keeps the signed-in capture page; a redirect to the login
// is not stored.
async function storeCapturePage(cache) {
  try {
    const response = await fetch(CAPTURE, { credentials: 'same-origin' });
    if (response.ok && !response.redirected) await cache.put(CAPTURE, response);
  } catch {
    // Offline while installing: the next visit stores the page.
  }
}

async function captureNetworkFirst(request) {
  const cache = await caches.open(CACHE);
  try {
    const response = await fetch(request);
    if (response.ok && !response.redirected) await cache.put(CAPTURE, response.clone());
    return response;
  } catch {
    return (await cache.match(CAPTURE)) || Response.error();
  }
}
