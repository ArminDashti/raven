/// <reference lib="webworker" />
import { clientsClaim } from 'workbox-core'
import { ExpirationPlugin } from 'workbox-expiration'
import { cleanupOutdatedCaches, createHandlerBoundToURL, precacheAndRoute } from 'workbox-precaching'
import { NavigationRoute, registerRoute } from 'workbox-routing'
import { CacheFirst, NetworkOnly } from 'workbox-strategies'
import { CacheableResponsePlugin } from 'workbox-cacheable-response'

declare let self: ServiceWorkerGlobalScope

// New deploy ⇒ new SW revision (vite-plugin-pwa / Workbox). autoUpdate + skipWaiting
// so clients pick up the shell without staying stuck on an old SW forever.
precacheAndRoute(self.__WB_MANIFEST)
cleanupOutdatedCaches()
void self.skipWaiting()
clientsClaim()

const base = import.meta.env.BASE_URL

try {
  registerRoute(
    new NavigationRoute(createHandlerBoundToURL(`${base}index.html`), {
      allowlist: [new RegExp(`^${base.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}`)],
    }),
  )
} catch {
  /* index.html missing from precache in unusual builds */
}

// Hashed build assets are precached; this covers same-origin runtime static media.
registerRoute(
  ({ request, url }) =>
    url.origin === self.location.origin &&
    (request.destination === 'image' ||
      request.destination === 'font' ||
      request.destination === 'style' ||
      request.destination === 'script'),
  new CacheFirst({
    cacheName: 'static-runtime',
    plugins: [
      new CacheableResponsePlugin({ statuses: [0, 200] }),
      new ExpirationPlugin({ maxEntries: 128, maxAgeSeconds: 60 * 60 * 24 * 30 }),
    ],
  }),
)

// Entire API is auth-bearing / private — never put responses in Cache Storage.
registerRoute(({ url }) => url.pathname.startsWith('/bugs/api/'), new NetworkOnly())

self.addEventListener('push', (event) => {
  let data = { title: 'Raven', body: 'New notification', url: '/bugs/' }
  try {
    if (event.data) {
      data = { ...data, ...event.data.json() }
    }
  } catch {
    /* ignore */
  }
  event.waitUntil(
    self.registration.showNotification(data.title || 'Raven', {
      body: data.body || '',
      data: { url: data.url || '/bugs/' },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const target = (event.notification.data && event.notification.data.url) || '/bugs/'
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clients) => {
      for (const client of clients) {
        if ('focus' in client) {
          client.navigate(target)
          return client.focus()
        }
      }
      if (self.clients.openWindow) {
        return self.clients.openWindow(target)
      }
    }),
  )
})
