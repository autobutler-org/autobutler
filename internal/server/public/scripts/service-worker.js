const CACHE_NAME = 'autobutler-v2';
const RUNTIME_CACHE = 'autobutler-runtime-v2';

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
  
  // Images and icons
  '/public/img/butler.png',
  '/public/favicons/48x48.ico'
];

// Install event - cache critical static assets for instant page loads
self.addEventListener('install', (event) => {
  console.log('[Service Worker] Installing...');
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('[Service Worker] Caching critical assets');
        return cache.addAll(STATIC_ASSETS);
      })
      .then(() => {
        console.log('[Service Worker] Installed successfully');
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[Service Worker] Installation failed:', error);
      })
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[Service Worker] Activating...');
  const cacheWhitelist = [CACHE_NAME, RUNTIME_CACHE];
  
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames.map((cacheName) => {
          if (!cacheWhitelist.includes(cacheName)) {
            console.log('[Service Worker] Deleting old cache:', cacheName);
            return caches.delete(cacheName);
          }
        })
      );
    }).then(() => {
      console.log('[Service Worker] Activated');
      return self.clients.claim();
    })
  );
});

// Fetch event - intelligent caching strategies
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip cross-origin requests
  if (url.origin !== location.origin) {
    return;
  }

  // Cache-first strategy for static assets (CSS, JS, images)
  if (isStaticAsset(request)) {
    event.respondWith(cacheFirst(request));
    return;
  }

  // Network-first strategy for HTML pages and API calls
  if (isNavigationRequest(request) || isAPIRequest(request)) {
    event.respondWith(networkFirst(request));
    return;
  }

  // Default: cache-first with network fallback
  event.respondWith(cacheFirst(request));
});

// Cache-first strategy: serve from cache, fallback to network
async function cacheFirst(request) {
  const cache = await caches.open(CACHE_NAME);
  const cached = await cache.match(request);
  
  if (cached) {
    console.log('[Service Worker] Serving from cache:', request.url);
    return cached;
  }

  try {
    const response = await fetch(request);
    
    if (response && response.status === 200) {
      // Clone and cache the response for future use
      const responseToCache = response.clone();
      const runtimeCache = await caches.open(RUNTIME_CACHE);
      await runtimeCache.put(request, responseToCache);
    }
    
    return response;
  } catch (error) {
    console.error('[Service Worker] Fetch failed:', error);
    
    // Try to get from runtime cache as last resort
    const runtimeCache = await caches.open(RUNTIME_CACHE);
    const runtimeCached = await runtimeCache.match(request);
    
    if (runtimeCached) {
      return runtimeCached;
    }
    
    // Return offline page or error response
    return new Response('Offline - Content not available', {
      status: 503,
      statusText: 'Service Unavailable',
      headers: new Headers({
        'Content-Type': 'text/plain'
      })
    });
  }
}

// Network-first strategy: try network first, fallback to cache
async function networkFirst(request) {
  try {
    const response = await fetch(request);
    
    if (response && response.status === 200) {
      // Cache successful responses
      const responseToCache = response.clone();
      const runtimeCache = await caches.open(RUNTIME_CACHE);
      await runtimeCache.put(request, responseToCache);
    }
    
    return response;
  } catch (error) {
    console.log('[Service Worker] Network failed, trying cache:', request.url);
    
    // Try static cache first
    const staticCache = await caches.open(CACHE_NAME);
    const staticCached = await staticCache.match(request);
    
    if (staticCached) {
      return staticCached;
    }
    
    // Try runtime cache
    const runtimeCache = await caches.open(RUNTIME_CACHE);
    const runtimeCached = await runtimeCache.match(request);
    
    if (runtimeCached) {
      return runtimeCached;
    }
    
    // Return offline fallback
    if (isNavigationRequest(request)) {
      const fallback = await staticCache.match('/');
      if (fallback) {
        return fallback;
      }
    }
    
    return new Response('Offline', {
      status: 503,
      statusText: 'Service Unavailable'
    });
  }
}

// Helper: Check if request is for a static asset
function isStaticAsset(request) {
  const url = new URL(request.url);
  const path = url.pathname;
  
  return path.match(/\.(css|js|png|jpg|jpeg|gif|svg|ico|woff|woff2|ttf|eot)$/i) ||
         path.startsWith('/public/');
}

// Helper: Check if request is a navigation request (HTML page)
function isNavigationRequest(request) {
  return request.mode === 'navigate' || 
         (request.method === 'GET' && request.headers.get('accept')?.includes('text/html'));
}

// Helper: Check if request is an API call
function isAPIRequest(request) {
  const url = new URL(request.url);
  return url.pathname.startsWith('/api/');
}

// Background sync for failed requests (future enhancement)
self.addEventListener('sync', (event) => {
  if (event.tag === 'sync-data') {
    console.log('[Service Worker] Background sync triggered');
    // Implement background sync logic here
  }
});

// Handle push notifications (future enhancement)
self.addEventListener('push', (event) => {
  console.log('[Service Worker] Push notification received');
  // Implement push notification logic here
});
