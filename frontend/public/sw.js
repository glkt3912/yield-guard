const API_CACHE = 'api-cache';
const STATIC_CACHE = 'static-cache-v1';
const MAX_ENTRIES = 64;
const MAX_AGE_SECONDS = 300;

const PRECACHE_URLS = ['/', '/icons/icon-192x192.png', '/icons/icon-512x512.png'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => cache.addAll(PRECACHE_URLS))
      .then(() => self.skipWaiting())
  );
});

self.addEventListener('activate', (event) => {
  const known = new Set([API_CACHE, STATIC_CACHE]);
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => !known.has(k)).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  if (request.method !== 'GET') return;
  if (url.pathname.startsWith('/_next/static/') || url.pathname.startsWith('/icons/')) {
    event.respondWith(cacheFirst(request));
  } else if (url.pathname === '/') {
    event.respondWith(networkFirstShell(request));
  } else if (/\/api\/.*/i.test(url.pathname)) {
    event.respondWith(networkFirst(request));
  }
});

async function cacheFirst(request) {
  const cache = await caches.open(STATIC_CACHE);
  const cached = await cache.match(request);
  if (cached) return cached;
  try {
    const response = await fetch(request);
    if (response.ok) await cache.put(request, response.clone());
    return response;
  } catch {
    return new Response('Network error', { status: 503 });
  }
}

async function networkFirstShell(request) {
  const cache = await caches.open(STATIC_CACHE);
  try {
    const response = await fetch(request);
    if (response.ok) await cache.put(request, response.clone());
    return response;
  } catch {
    const cached = await cache.match(request);
    return cached ?? new Response('Network error', { status: 503 });
  }
}

async function networkFirst(request) {
  const cache = await caches.open(API_CACHE);
  try {
    const response = await fetch(request);
    if (response.ok) {
      await trimCache(cache, MAX_ENTRIES - 1);
      await cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await cache.match(request);
    if (cached) {
      const date = cached.headers.get('date');
      if (!date || (Date.now() - new Date(date).getTime()) / 1000 <= MAX_AGE_SECONDS) {
        return cached;
      }
    }
    return new Response('Network error', { status: 503 });
  }
}

async function trimCache(cache, max) {
  const keys = await cache.keys();
  for (let i = 0; i < keys.length - max; i++) {
    await cache.delete(keys[i]);
  }
}
