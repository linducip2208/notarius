/* SaktiNotaris service worker — jujur & hemat.
 * Strategi: aset statis (/assets/*) cache-first agar cepat + hemat kuota;
 * navigasi (HTML) network-first agar data selalu segar, fallback ke /offline
 * bila tidak ada koneksi DAN tidak ada cache. TIDAK meng-cache halaman
 * privat (/app, /clients, ...) demi keamanan data kantor.
 */
const CACHE = 'saktinotaris-v1';
const PRECACHE = [
  '/offline',
  '/assets/vendor/tabler.min.css',
  '/assets/vendor/tabler.min.js',
  '/assets/brand/favicon.svg',
  '/assets/brand/icon-192.png',
  '/assets/brand/icon-512.png',
  '/assets/brand/apple-touch-icon.png',
];

self.addEventListener('install', (e) => {
  e.waitUntil(caches.open(CACHE).then((c) => c.addAll(PRECACHE)).then(() => self.skipWaiting()));
});

self.addEventListener('activate', (e) => {
  e.waitUntil(
    caches.keys().then((keys) => Promise.all(keys.filter((k) => k !== CACHE).map((k) => caches.delete(k)))).then(() => self.clients.claim()),
  );
});

self.addEventListener('fetch', (e) => {
  const req = e.request;
  if (req.method !== 'GET') return;
  const url = new URL(req.url);
  if (url.origin !== self.location.origin) return;

  // Navigasi: jaringan dulu, cache bila offline, /offline bila tak ada keduanya.
  if (req.mode === 'navigate') {
    e.respondWith(
      fetch(req)
        .then((res) => {
          const copy = res.clone();
          caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => {});
          return res;
        })
        .catch(() => caches.match(req).then((hit) => hit || caches.match('/offline'))),
    );
    return;
  }

  // Aset statis milik sendiri: cache dulu, isi cache di latar.
  if (url.pathname.startsWith('/assets/')) {
    e.respondWith(
      caches.match(req).then((hit) => {
        const net = fetch(req).then((res) => {
          if (res && res.ok) {
            const copy = res.clone();
            caches.open(CACHE).then((c) => c.put(req, copy)).catch(() => {});
          }
          return res;
        }).catch(() => hit);
        return hit || net;
      }),
    );
  }
});

// Web Push: tampilkan notifikasi dari payload JSON {title, body, url}.
// Payload dikirim oleh gateway/CLI manual (lihat docs/ops/WEBPUSH.md) —
// aplikasi sendiri tidak mengirim Web Push terenkripsi (tanpa sender).
self.addEventListener('push', (e) => {
  let data = { title: 'SaktiNotaris', body: 'Ada pembaruan.', url: '/app' };
  try {
    if (e.data) data = Object.assign(data, e.data.json());
  } catch (_) {
    /* abaikan payload non-JSON, pakai default */
  }
  e.waitUntil(
    self.registration.showNotification(data.title, {
      body: data.body,
      icon: '/assets/brand/icon-192.png',
      badge: '/assets/brand/icon-192.png',
      data: { url: data.url || '/app' },
    }),
  );
});

self.addEventListener('notificationclick', (e) => {
  e.notification.close();
  const target = (e.notification.data && e.notification.data.url) || '/app';
  e.waitUntil(clients.openWindow(target));
});
