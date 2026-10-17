/*
 * Service worker for web push (customer and merchant web, 2026-10-17).
 *
 * The engine sends an encrypted payload (RFC 8291) that the browser decrypts
 * and hands here as JSON: { title, body, url, tag, urgent }.
 * One file, two registrations: scope "/" for the customer and "/store/" for
 * the merchant panel, so each keeps its own subscription.
 * No fetch handler: this worker never caches or intercepts pages.
 */
self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("push", (event) => {
  let data = {};
  try {
    data = event.data ? event.data.json() : {};
  } catch (e) {
    data = { body: event.data ? event.data.text() : "" };
  }
  const title = data.title || "RahalGo";
  const urgent = data.urgent === true;
  const options = {
    body: data.body || "",
    icon: "/pwa/icon-192.png",
    badge: "/pwa/icon-192.png",
    dir: "rtl",
    lang: "ar",
    tag: data.tag || undefined,
    renotify: !!data.tag,
    requireInteraction: urgent,
    vibrate: urgent ? [300, 120, 300, 120, 300] : [200],
    data: { url: data.url || self.registration.scope },
  };
  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener("notificationclick", (event) => {
  event.notification.close();
  const target = new URL(
    (event.notification.data && event.notification.data.url) || "/",
    self.location.origin,
  ).href;
  event.waitUntil(
    (async () => {
      const all = await self.clients.matchAll({ type: "window", includeUncontrolled: true });
      // Same page already open: focus it.
      for (const c of all) {
        if (c.url === target && "focus" in c) return c.focus();
      }
      // Any page of the site open: bring it to the target.
      for (const c of all) {
        if (new URL(c.url).origin === self.location.origin && "focus" in c) {
          try {
            const moved = "navigate" in c ? await c.navigate(target) : null;
            return (moved || c).focus();
          } catch (e) {
            return c.focus();
          }
        }
      }
      return self.clients.openWindow(target);
    })(),
  );
});
