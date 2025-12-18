const CACHE_NAME = 'autobutler-v3';
const RUNTIME_CACHE = 'autobutler-runtime-v3';

// Critical assets to cache on install for instant loading
const STATIC_ASSETS = [
    '/',
    '/public/manifest.json',

    // Core CSS - Navigation and Layout (non-hydrating components)
    '/public/styles/site.css',
    '/public/styles/variables.css',
    '/public/styles/reset.css',
    '/public/styles/layout.css',
    '/public/styles/navigation.css',
    '/public/styles/landing.css',
    '/public/styles/buttons.css',
    '/public/styles/hero.css',
    '/public/styles/icons.css',
    '/public/styles/utility.css',
    '/public/styles/touch-feedback.css',

    // JavaScript libraries
    '/public/vendor/tailwind/tailwind.3.4.16.js',
    '/public/vendor/htmx/htmx.min.js',
    '/public/scripts/pwa-init.js',

    // Images and icons - ensures all visual assets load instantly
    '/public/img/butler.png',
    '/public/favicons/48x48.ico',

    // Note: SVG icons (book, settings, devices) are rendered inline via templ components
    // and are automatically included in the cached HTML pages
];

// Revalidation interval: only revalidate cached content after this time (in ms)
// This prevents constant background fetches while still keeping content fresh
const REVALIDATE_INTERVAL = 5 * 60 * 1000; // 5 minutes

// Store last revalidation times (in-memory, resets on SW restart)
const lastRevalidated = new Map();

// Install event - cache critical static assets for instant page loads
self.addEventListener('install', (event) => {
    event.waitUntil(
        caches
            .open(CACHE_NAME)
            .then((cache) => cache.addAll(STATIC_ASSETS))
            .then(() => self.skipWaiting())
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
    const cacheWhitelist = [CACHE_NAME, RUNTIME_CACHE];

    event.waitUntil(
        caches
            .keys()
            .then((cacheNames) => {
                return Promise.all(
                    cacheNames.map((cacheName) => {
                        if (!cacheWhitelist.includes(cacheName)) {
                            return caches.delete(cacheName);
                        }
                    })
                );
            })
            .then(() => self.clients.claim())
    );
});

// Fetch event - intelligent caching strategies optimized for speed
self.addEventListener('fetch', (event) => {
    const { request } = event;
    const url = new URL(request.url);

    // Skip cross-origin requests
    if (url.origin !== location.origin) {
        return;
    }

    // Skip non-GET requests
    if (request.method !== 'GET') {
        return;
    }

    // Network-only for API calls (always need fresh data)
    if (isAPIRequest(url)) {
        event.respondWith(networkOnly(request));
        return;
    }

    // Cache-first for all static assets (CSS, JS, images, fonts)
    // These are versioned via cache name, so cache-first is fast and safe
    if (isStaticAsset(url)) {
        event.respondWith(cacheFirst(request, url));
        return;
    }

    // Navigation requests: cache-first with lazy background revalidation
    // Serves instantly from cache, only revalidates periodically
    if (isNavigationRequest(request)) {
        event.respondWith(cacheFirstWithLazyRevalidate(request, url));
        return;
    }

    // Default: cache-first for everything else
    event.respondWith(cacheFirst(request, url));
});

// Cache-first strategy: serve from cache immediately, fallback to network
async function cacheFirst(request, url) {
    // Check both caches in parallel for speed
    const [staticCache, runtimeCache] = await Promise.all([
        caches.open(CACHE_NAME),
        caches.open(RUNTIME_CACHE),
    ]);

    // Try static cache first (pre-cached assets)
    const staticCached = await staticCache.match(request);
    if (staticCached) {
        return staticCached;
    }

    // Try runtime cache
    const runtimeCached = await runtimeCache.match(request);
    if (runtimeCached) {
        return runtimeCached;
    }

    // Cache miss - fetch from network
    try {
        const response = await fetch(request);

        if (response && response.status === 200 && response.type === 'basic') {
            // Cache the response for future use (don't await - fire and forget)
            runtimeCache.put(request, response.clone());
        }

        return response;
    } catch {
        return offlineResponse();
    }
}

// Cache-first with lazy background revalidation for HTML pages
// Only revalidates if content is older than REVALIDATE_INTERVAL
async function cacheFirstWithLazyRevalidate(request, url) {
    const cacheKey = url.pathname;

    // Check both caches
    const [staticCache, runtimeCache] = await Promise.all([
        caches.open(CACHE_NAME),
        caches.open(RUNTIME_CACHE),
    ]);

    const staticCached = await staticCache.match(request);
    const runtimeCached = await runtimeCache.match(request);
    const cached = runtimeCached || staticCached;

    if (cached) {

        // Check if we should revalidate in the background
        const lastTime = lastRevalidated.get(cacheKey) || 0;
        const now = Date.now();

        if (now - lastTime > REVALIDATE_INTERVAL) {
            // Mark as revalidating to prevent duplicate fetches
            lastRevalidated.set(cacheKey, now);

            // Lazy revalidate in background (don't block response)
            lazyRevalidate(request, runtimeCache);
        }

        return cached;
    }

    // No cache - must fetch from network
    try {
        const response = await fetch(request);

        if (response && response.status === 200 && response.type === 'basic') {
            runtimeCache.put(request, response.clone());
            lastRevalidated.set(cacheKey, Date.now());
        }

        return response;
    } catch {
        // Try to return homepage as fallback for navigation
        const fallback = await staticCache.match('/');
        return fallback || offlineResponse();
    }
}

// Lazy background revalidation - doesn't block the response
function lazyRevalidate(request, cache) {
    // Use setTimeout to ensure this runs after the response is sent
    setTimeout(() => {
        fetch(request)
            .then((response) => {
                if (response && response.status === 200 && response.type === 'basic') {
                    cache.put(request, response);
                }
            })
            .catch(() => {
                // Silent fail - cache still has valid content
            });
    }, 100);
}

// Network-only for API requests (no caching)
async function networkOnly(request) {
    try {
        return await fetch(request);
    } catch {
        return new Response(JSON.stringify({ error: 'Offline' }), {
            status: 503,
            statusText: 'Service Unavailable',
            headers: { 'Content-Type': 'application/json' },
        });
    }
}

// Offline fallback response
function offlineResponse() {
    return new Response('Offline - Content not available', {
        status: 503,
        statusText: 'Service Unavailable',
        headers: { 'Content-Type': 'text/plain' },
    });
}

// Helper: Check if request is for a static asset (CSS, JS, images, fonts)
function isStaticAsset(url) {
    const path = url.pathname;
    return (
        path.startsWith('/public/') ||
        path.match(/\.(css|js|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot|webp)$/i)
    );
}

// Helper: Check if request is a navigation request (HTML page)
function isNavigationRequest(request) {
    return (
        request.mode === 'navigate' ||
        request.headers.get('accept')?.includes('text/html')
    );
}

// Helper: Check if request is an API call
function isAPIRequest(url) {
    return url.pathname.startsWith('/api/');
}

// Background sync for failed requests (future enhancement)
self.addEventListener('sync', () => {
    // Implement background sync logic here
});

// Handle push notifications (future enhancement)
self.addEventListener('push', () => {
    // Implement push notification logic here
});
