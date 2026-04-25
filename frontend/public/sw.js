const API_CACHE = 'api-cache';
const STATIC_CACHE = 'static-cache-v1'; // new deploy → bump to v2, v3, ... to purge old entries
const MAX_ENTRIES = 64;
const MAX_AGE_SECONDS = 300;

const PRECACHE_URLS = ['/', '/manifest.webmanifest', '/icons/icon-192x192.png', '/icons/icon-512x512.png'];

self.addEventListener('install', (event) => {
  event.waitUntil(
    caches.open(STATIC_CACHE)
      .then((cache) => Promise.all(PRECACHE_URLS.map((url) => cache.add(url).catch(() => {}))))
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
  if (
    url.pathname.startsWith('/_next/static/') ||
    url.pathname.startsWith('/icons/') ||
    url.pathname.startsWith('/fonts/')
  ) {
    event.respondWith(cacheFirst(request));
  } else if (url.pathname === '/') {
    event.respondWith(networkFirst(request, { cacheName: STATIC_CACHE }));
  } else if (/\/api\/.*/i.test(url.pathname)) {
    event.respondWith(networkFirst(request, { cacheName: API_CACHE, trim: true, checkAge: true }));
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

async function networkFirst(request, { cacheName, trim = false, checkAge = false }) {
  const cache = await caches.open(cacheName);
  try {
    const response = await fetch(request);
    if (response.ok) {
      if (trim) await trimCache(cache, MAX_ENTRIES - 1);
      await cache.put(request, response.clone());
    }
    return response;
  } catch {
    const cached = await cache.match(request);
    if (cached) {
      if (checkAge) {
        const date = cached.headers.get('date');
        if (!date || (Date.now() - new Date(date).getTime()) / 1000 <= MAX_AGE_SECONDS) {
          return cached;
        }
      } else {
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
