self.addEventListener("install", (event) => {
  event.waitUntil(
    caches
      .open(SHELL_CACHE_NAME)
      .then(async (cache) => {
        await cache.addAll(REQUIRED_SHELL_ASSETS);
        await Promise.all(
          OPTIONAL_SHELL_ASSETS.map(async (asset) => {
            try {
              await cache.add(asset);
            } catch {
              // Decorative assets must not make the shell installation fail.
            }
          }),
        );
      })
      .then(() => self.skipWaiting()),
  );
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((names) =>
        Promise.all(
          names
            .filter(
              (name) => name.startsWith(SHELL_CACHE_PREFIX) && name !== SHELL_CACHE_NAME,
            )
            .map((name) => caches.delete(name)),
        ),
      )
      .then(() => clients.claim())
      .then(() => refreshSafeShellClients()),
  );
});

self.addEventListener("fetch", (event) => {
  const request = event.request;
  if (request.method !== "GET") return;

  const url = new URL(request.url);
  if (url.origin !== self.location.origin) return;

  if (request.mode === "navigate") {
    event.respondWith(networkFirstNavigation(request));
    return;
  }

  if (isKnownStaticAsset(url.pathname)) {
    event.respondWith(cacheFirstStatic(request));
  }
  // API and other same-origin requests deliberately pass through untouched.
});

self.addEventListener("push", (event) => {
  if (!event.data) {
    return;
  }

  let payload = {};
  try {
    payload = event.data.json();
  } catch {
    payload = { title: "Blind Timer", body: event.data.text() };
  }

  event.waitUntil(
    self.registration.showNotification(payload.title || "Blind Timer", {
      body: payload.body || "",
      tag: payload.tag || "blind-clock",
      renotify: true,
      data: {
        url: payload.url || "/blinds/presentation",
      },
    }),
  );
});

const SHELL_CACHE_PREFIX = "poker-session-control-shell-";
const SHELL_CACHE_VERSION = "v8-2026-08-21-session-access-guard";
const SHELL_CACHE_NAME = `${SHELL_CACHE_PREFIX}${SHELL_CACHE_VERSION}`;

const REQUIRED_SHELL_ASSETS = Object.freeze([
  "/",
  "/manifest.webmanifest",
  "/static/css/main.css",
  "/static/js/app.js",
  "/static/js/account-ownership-ui.js",
  "/static/js/api.js",
  "/static/js/i18n.js",
  "/static/js/network-contract.js",
  "/static/js/offline-db.js",
  "/static/js/offline-sync.js",
  "/static/js/rollout.js",
  "/static/js/session-cache.js",
  "/static/js/session-projection.js",
  "/static/js/state.js",
  "/static/js/sync-status.js",
  "/static/js/utils.js",
  "/static/js/ui/blinds.js",
  "/static/js/ui/lobby.js",
  "/static/js/ui/player.js",
  "/static/js/ui/session.js",
  "/static/svg/01-home-filled-white.svg",
  "/static/svg/02-spade-filled-gold.svg",
  "/static/svg/03-timer-outline-white.svg",
  "/static/svg/04-play-filled-white.svg",
  "/static/svg/05-plus-circle-gold.svg",
  "/static/svg/07-chevron-right-white.svg",
  "/static/svg/08-calendar-outline-gold.svg",
  "/static/svg/09-clock-outline-gold.svg",
  "/static/svg/11-calculator-outline-green.svg",
  "/static/svg/13-user-outline-white.svg",
]);

const OPTIONAL_SHELL_ASSETS = Object.freeze([
  "/static/assets/poker-mark.png",
  "/static/svg/06-chevron-down-white.svg",
  "/static/svg/10-poker-chips-gold.svg",
  "/static/svg/12-users-outline-gold.svg",
]);

function isKnownStaticAsset(pathname) {
  return pathname === "/manifest.webmanifest" || pathname.startsWith("/static/");
}

async function refreshSafeShellClients() {
  const windowClients = await clients.matchAll({ type: "window", includeUncontrolled: true });
  await Promise.all(
    windowClients.map((client) => {
      const url = new URL(client.url);
      if (url.origin !== self.location.origin) return undefined;
      if (url.pathname !== "/" && url.pathname !== "/account") return undefined;
      return client.navigate(client.url);
    }),
  );
}

async function networkFirstNavigation(request) {
  try {
    return await fetch(request);
  } catch {
    const cached = await caches.match("/", { cacheName: SHELL_CACHE_NAME });
    if (cached) return cached;
    throw new Error("Offline shell is unavailable");
  }
}

async function cacheFirstStatic(request) {
  const cached = await caches.match(request, { cacheName: SHELL_CACHE_NAME });
  if (cached) return cached;

  const response = await fetch(request);
  if (response?.ok) {
    const cache = await caches.open(SHELL_CACHE_NAME);
    await cache.put(request, response.clone());
  }
  return response;
}

self.addEventListener("notificationclick", (event) => {
  event.notification.close();

  const targetURL = event.notification.data?.url || "/blinds/presentation";

  event.waitUntil(
    clients.matchAll({ type: "window", includeUncontrolled: true }).then((windowClients) => {
      for (const client of windowClients) {
        if ("focus" in client) {
          client.navigate(targetURL);
          return client.focus();
        }
      }

      if (clients.openWindow) {
        return clients.openWindow(targetURL);
      }

      return undefined;
    }),
  );
});
