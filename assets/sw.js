const CACHE_VERSION = 'cartero-v1';

const PRECACHE_URLS = [
    '/?date=today',
    '/?date=yesterday',
    '/assets/favicon.jpg',
];

// Install: pre-cache key pages and assets
self.addEventListener('install', function(event) {
    event.waitUntil(
        caches.open(CACHE_VERSION).then(function(cache) {
            return cache.addAll(PRECACHE_URLS);
        }).then(function() {
            return self.skipWaiting();
        })
    );
});

// Activate: delete old versioned caches
self.addEventListener('activate', function(event) {
    event.waitUntil(
        caches.keys().then(function(cacheNames) {
            return Promise.all(
                cacheNames
                    .filter(function(name) { return name !== CACHE_VERSION; })
                    .map(function(name) { return caches.delete(name); })
            );
        }).then(function() {
            return self.clients.claim();
        })
    );
});

// Fetch: network-first with cache fallback
self.addEventListener('fetch', function(event) {
    // Only handle GET requests
    if (event.request.method !== 'GET') return;

    const url = new URL(event.request.url);

    // Skip cross-origin requests (fonts, external links)
    if (url.origin !== self.location.origin) return;

    if (isNavigationRequest(event.request)) {
        event.respondWith(networkFirstNavigation(event.request));
    } else {
        event.respondWith(cacheFirstAsset(event.request));
    }
});

function isNavigationRequest(request) {
    return request.mode === 'navigate' ||
        (request.method === 'GET' && request.headers.get('accept') &&
         request.headers.get('accept').includes('text/html'));
}

// Network-first: try network, update cache, fall back to cache on failure
function networkFirstNavigation(request) {
    return fetch(request).then(function(response) {
        if (response.ok) {
            // Clone and store in cache before returning
            var responseClone = response.clone();
            caches.open(CACHE_VERSION).then(function(cache) {
                cache.put(request, responseClone);
            });
        }
        return response;
    }).catch(function() {
        // Network failed — try exact URL match first
        return caches.match(request).then(function(cached) {
            if (cached) return cached;

            // Fall back to the today page if we have it
            return caches.match('/?date=today').then(function(fallback) {
                if (fallback) return fallback;
                // Last resort: any cached page
                return caches.match('/');
            });
        });
    });
}

// Cache-first: serve from cache, fetch and update in background if missing
function cacheFirstAsset(request) {
    return caches.match(request).then(function(cached) {
        if (cached) {
            // Revalidate in background
            fetch(request).then(function(response) {
                if (response.ok) {
                    caches.open(CACHE_VERSION).then(function(cache) {
                        cache.put(request, response);
                    });
                }
            }).catch(function() {});
            return cached;
        }
        // Not in cache, fetch and store
        return fetch(request).then(function(response) {
            if (response.ok) {
                var responseClone = response.clone();
                caches.open(CACHE_VERSION).then(function(cache) {
                    cache.put(request, responseClone);
                });
            }
            return response;
        });
    });
}
