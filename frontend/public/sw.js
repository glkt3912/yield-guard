const API_CACHE = 'api-cache';
const MAX_ENTRIES = 64;
const MAX_AGE_SECONDS = 300;

self.addEventListener('install', () => self.skipWaiting());

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.filter((k) => k !== API_CACHE).map((k) => caches.delete(k))))
      .then(() => self.clients.claim())
  );
});

self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  if (/\/api\/.*/i.test(url.pathname) && request.method === 'GET') {
    event.respondWith(networkFirst(request));
  }
});

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
