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
