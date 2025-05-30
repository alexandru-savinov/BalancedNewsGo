/**
 * Service Worker for NewsBalancer
 * Provides caching for static assets and offline functionality
 */

const CACHE_NAME = 'newsbalancer-v1';
const STATIC_CACHE_URLS = [
  '/',
  '/articles',
  '/static/css/main.css',
  '/static/css/components/bias-slider.css',
  '/static/css/components/articles.css',
  '/static/css/components/performance-dashboard.css',
  '/static/js/main.js',
  '/static/js/utils/ApiClient.js',
  '/static/js/utils/PerformanceMonitor.js',
  '/static/js/utils/ComponentPerformanceMonitor.js',
  '/static/js/utils/LazyLoader.js',
  '/static/js/components/BiasSlider.js',
  '/static/js/components/ArticleCard.js',
  '/static/js/pages/articles.js',
  '/static/js/pages/article-detail.js',
  '/static/js/pages/admin.js'
];

// Install event - cache static assets
self.addEventListener('install', (event) => {
  console.log('[ServiceWorker] Install');
  
  event.waitUntil(
    caches.open(CACHE_NAME)
      .then((cache) => {
        console.log('[ServiceWorker] Caching static assets');
        return cache.addAll(STATIC_CACHE_URLS);
      })
      .then(() => {
        // Force the waiting service worker to become the active service worker
        return self.skipWaiting();
      })
      .catch((error) => {
        console.error('[ServiceWorker] Failed to cache static assets:', error);
      })
  );
});

// Activate event - clean up old caches
self.addEventListener('activate', (event) => {
  console.log('[ServiceWorker] Activate');
  
  event.waitUntil(
    caches.keys()
      .then((cacheNames) => {
        return Promise.all(
          cacheNames.map((cacheName) => {
            if (cacheName !== CACHE_NAME) {
              console.log('[ServiceWorker] Removing old cache:', cacheName);
              return caches.delete(cacheName);
            }
          })
        );
      })
      .then(() => {
        // Ensure the service worker takes control of all clients
        return self.clients.claim();
      })
  );
});

// Fetch event - serve from cache with network fallback
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);
  
  // Only handle GET requests
  if (request.method !== 'GET') {
    return;
  }
  
  // Handle static assets with cache-first strategy
  if (isStaticAsset(url)) {
    event.respondWith(
      caches.match(request)
        .then((cachedResponse) => {
          if (cachedResponse) {
            return cachedResponse;
          }
          
          return fetch(request)
            .then((response) => {
              // Don't cache if not a valid response
              if (!response || response.status !== 200 || response.type !== 'basic') {
                return response;
              }
              
              // Clone the response to cache it
              const responseToCache = response.clone();
              caches.open(CACHE_NAME)
                .then((cache) => {
                  cache.put(request, responseToCache);
                });
              
              return response;
            });
        })
        .catch(() => {
          // Return offline fallback for static assets if available
          if (url.pathname.endsWith('.html')) {
            return caches.match('/');
          }
        })
    );
  }
  
  // Handle API requests with network-first strategy
  else if (isApiRequest(url)) {
    event.respondWith(
      fetch(request)
        .then((response) => {
          // Cache successful API responses for a short time
          if (response.status === 200 && request.method === 'GET') {
            const responseToCache = response.clone();
            caches.open(CACHE_NAME)
              .then((cache) => {
                // Cache API responses for 5 minutes
                const headers = new Headers(responseToCache.headers);
                headers.set('sw-cache-timestamp', Date.now().toString());
                const cachedResponse = new Response(responseToCache.body, {
                  status: responseToCache.status,
                  statusText: responseToCache.statusText,
                  headers: headers
                });
                cache.put(request, cachedResponse);
              });
          }
          return response;
        })
        .catch(() => {
          // Try to serve from cache for GET requests
          if (request.method === 'GET') {
            return caches.match(request)
              .then((cachedResponse) => {
                if (cachedResponse) {
                  // Check if cached response is still fresh (5 minutes)
                  const cacheTimestamp = cachedResponse.headers.get('sw-cache-timestamp');
                  if (cacheTimestamp) {
                    const age = Date.now() - parseInt(cacheTimestamp, 10);
                    if (age < 5 * 60 * 1000) { // 5 minutes
                      return cachedResponse;
                    }
                  }
                }
                
                // Return a basic offline response for API requests
                return new Response(
                  JSON.stringify({
                    success: false,
                    error: 'Offline - please check your connection'
                  }),
                  {
                    status: 503,
                    statusText: 'Service Unavailable',
                    headers: { 'Content-Type': 'application/json' }
                  }
                );
              });
          }
        })
    );
  }
  
  // For all other requests, use default browser behavior
});

// Helper functions
function isStaticAsset(url) {
  return url.pathname.startsWith('/static/') || 
         url.pathname.endsWith('.css') ||
         url.pathname.endsWith('.js') ||
         url.pathname.endsWith('.html') ||
         url.pathname === '/' ||
         url.pathname === '/articles' ||
         url.pathname === '/admin' ||
         url.pathname.startsWith('/article/');
}

function isApiRequest(url) {
  return url.pathname.startsWith('/api/');
}

// Background sync for offline actions (future enhancement)
self.addEventListener('sync', (event) => {
  console.log('[ServiceWorker] Background sync:', event.tag);
  
  if (event.tag === 'background-sync') {
    event.waitUntil(
      // Handle background sync operations
      Promise.resolve()
    );
  }
});

// Push notifications (future enhancement)
self.addEventListener('push', (event) => {
  console.log('[ServiceWorker] Push received');
  
  const options = {
    body: 'New articles available',
    icon: '/static/icons/icon-192x192.png',
    badge: '/static/icons/badge-72x72.png',
    tag: 'news-update',
    requireInteraction: false,
    actions: [
      {
        action: 'view',
        title: 'View Articles'
      },
      {
        action: 'dismiss',
        title: 'Dismiss'
      }
    ]
  };
  
  event.waitUntil(
    self.registration.showNotification('NewsBalancer', options)
  );
});
