/* Investo service worker v2 — local-first PWA. */
const CACHE_NAME = 'investo-v2';
const PRECACHE = [
    '/',
    '/dashboard',
    '/saham',
    '/forex',
    '/screener',
    '/berita',
    '/static/offline.html',
    '/static/manifest.json',
    '/static/favicon.svg',
    '/static/icons/icon-192.png',
    '/static/icons/icon-512.png',
    '/static/css/vendor/tabler.css',
    '/static/css/output.css',
    '/static/css/fintech.css',
    '/static/css/app.css',
    '/static/js/app.js',
    '/static/js/vendor/alpine.min.js',
    '/static/js/vendor/lightweight-charts.js',
    '/static/js/vendor/chart.umd.min.js',
    '/static/js/vendor/marked.min.js'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => cache.addAll(PRECACHE)).catch(() => {})
    );
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keys) => Promise.all(
            keys.filter((k) => k !== CACHE_NAME).map((k) => caches.delete(k))
        )).then(() => self.clients.claim())
    );
});

self.addEventListener('fetch', (event) => {
    if (event.request.method !== 'GET') return;
    const url = new URL(event.request.url);
    if (url.origin !== self.location.origin) return; // never cache third parties

    // API + realtime: network only (real data, never stale)
    if (url.pathname.startsWith('/api/') || url.pathname === '/ws') {
        event.respondWith(fetch(event.request).catch(() => offlineJson()));
        return;
    }
    // versioned static: cache first
    if (url.pathname.startsWith('/static/')) {
        event.respondWith(cacheFirst(event.request));
        return;
    }
    // pages: network first, offline fallback
    event.respondWith(networkFirstPage(event.request));
});

async function cacheFirst(request) {
    const cached = await caches.match(request);
    if (cached) return cached;
    try {
        const res = await fetch(request);
        if (res.ok) {
            const cache = await caches.open(CACHE_NAME);
            cache.put(request, res.clone());
        }
        return res;
    } catch (e) {
        return new Response('Offline', { status: 503 });
    }
}

async function networkFirstPage(request) {
    try {
        const res = await fetch(request);
        if (res.ok && res.headers.get('Content-Type') && res.headers.get('Content-Type').includes('text/html')) {
            const cache = await caches.open(CACHE_NAME);
            cache.put(request, res.clone());
        }
        return res;
    } catch (e) {
        const cached = await caches.match(request);
        if (cached) return cached;
        return caches.match('/static/offline.html');
    }
}

function offlineJson() {
    return new Response(JSON.stringify({ error: 'offline', message: 'Anda sedang offline.' }), {
        status: 503,
        headers: { 'Content-Type': 'application/json' }
    });
}

self.addEventListener('push', (event) => {
    let data = { title: 'Investo', body: 'Update baru tersedia', icon: '/static/icons/icon-192.png' };
    if (event.data) {
        try { data = { ...data, ...event.data.json() }; } catch (e) {}
    }
    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: data.icon || '/static/icons/icon-192.png',
            badge: '/static/icons/icon-192.png',
            tag: data.tag || 'investo-notification',
            data: data.url ? { url: data.url } : {}
        })
    );
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    const target = (event.notification.data && event.notification.data.url) || '/dashboard';
    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then((wins) => {
            for (const w of wins) {
                if (w.url.includes(target) && 'focus' in w) return w.focus();
            }
            if (clients.openWindow) return clients.openWindow(target);
        })
    );
});
