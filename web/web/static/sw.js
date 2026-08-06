const CACHE_NAME = 'investo-v1';
const STATIC_ASSETS = [
    '/',
    '/dashboard',
    '/saham',
    '/forex',
    '/screener',
    '/berita',
    '/blog',
    '/ideas',
    '/leaderboard',
    '/static/favicon.svg',
    '/static/manifest.json',
    '/offline'
];

self.addEventListener('install', (event) => {
    event.waitUntil(
        caches.open(CACHE_NAME).then((cache) => {
            return cache.addAll(STATIC_ASSETS);
        })
    );
    self.skipWaiting();
});

self.addEventListener('activate', (event) => {
    event.waitUntil(
        caches.keys().then((keys) => {
            return Promise.all(
                keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key))
            );
        })
    );
    self.clients.claim();
});

self.addEventListener('fetch', (event) => {
    if (event.request.method !== 'GET') return;

    const url = new URL(event.request.url);

    if (url.pathname.startsWith('/api/') || url.pathname === '/ws') {
        event.respondWith(networkFirst(event.request));
        return;
    }

    if (url.pathname.startsWith('/static/')) {
        event.respondWith(cacheFirst(event.request));
        return;
    }

    event.respondWith(networkFirst(event.request));
});

self.addEventListener('push', (event) => {
    let data = { title: 'Investo', body: 'Update baru tersedia', icon: '/static/favicon.svg' };
    if (event.data) {
        try {
            data = { ...data, ...event.data.json() };
        } catch(e) {}
    }

    event.waitUntil(
        self.registration.showNotification(data.title, {
            body: data.body,
            icon: data.icon,
            badge: '/static/favicon.svg',
            tag: data.tag || 'investo-notification',
            data: data.url ? { url: data.url } : {},
            vibrate: [200, 100, 200],
            requireInteraction: data.requireInteraction || false,
            actions: data.actions || []
        })
    );
});

self.addEventListener('notificationclick', (event) => {
    event.notification.close();
    const urlToOpen = (event.notification.data && event.notification.data.url) || '/dashboard';

    event.waitUntil(
        clients.matchAll({ type: 'window', includeUncontrolled: true }).then((windowClients) => {
            for (const client of windowClients) {
                if (client.url.includes(urlToOpen) && 'focus' in client) {
                    return client.focus();
                }
            }
            if (clients.openWindow) {
                return clients.openWindow(urlToOpen);
            }
        })
    );
});

async function cacheFirst(request) {
    const cached = await caches.match(request);
    if (cached) return cached;

    try {
        const response = await fetch(request);
        if (response.ok) {
            const cache = await caches.open(CACHE_NAME);
            cache.put(request, response.clone());
        }
        return response;
    } catch(e) {
        return new Response('Offline — resource not available', { status: 503 });
    }
}

async function networkFirst(request) {
    try {
        const response = await fetch(request);
        if (response.ok) {
            const cache = await caches.open(CACHE_NAME);
            cache.put(request, response.clone());
        }
        return response;
    } catch(e) {
        const cached = await caches.match(request);
        if (cached) return cached;

        if (request.headers.get('Accept') && request.headers.get('Accept').includes('text/html')) {
            const offlinePage = await caches.match('/');
            if (offlinePage) return offlinePage;
        }

        return new Response(JSON.stringify({ error: 'offline' }), {
            status: 503,
            headers: { 'Content-Type': 'application/json' }
        });
    }
}
