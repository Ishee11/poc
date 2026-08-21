(function installPokerStartup(global) {
  "use strict";

  const CACHE_PREFIX = "poker-session-control-shell-";
  const shellVersion =
    global.document?.querySelector?.('meta[name="poker-shell-version"]')?.content || "unknown";
  const recoveryKey = `poker-startup-recovery:${shellVersion}`;
  let state = "loading";
  let phase = "document";
  let shellReady = false;
  let lastFailure = null;

  function element(id) {
    return global.document?.getElementById?.(id) || null;
  }

  function errorName(error) {
    return typeof error?.name === "string" && error.name ? error.name.slice(0, 80) : "Error";
  }

  function errorMessage(error) {
    if (typeof error?.message === "string") return error.message;
    if (typeof error === "string") return error;
    return "";
  }

  function classify(category, error) {
    const message = errorMessage(error);
    if (
      category === "module" ||
      /ChunkLoadError|dynamically imported module|Importing a module script failed|module script/i.test(message)
    ) {
      return "chunk_loading";
    }
    if (error?.name === "TimeoutError" || /timed out|timeout/i.test(message)) return "timeout";
    if (global.navigator?.onLine === false) return "offline";
    return category || "bootstrap";
  }

  function diagnostic(category, error, failurePhase = phase) {
    const detail = {
      category: classify(category, error),
      phase: String(failurePhase || "unknown").slice(0, 80),
      error_name: errorName(error),
      online: global.navigator?.onLine !== false,
      shell_version: shellVersion,
    };
    global.console?.error?.("startup_failure", detail);
    return detail;
  }

  function render() {
    const shell = element("startup-shell");
    if (!shell) return;

    const message = element("startup-message");
    const detail = element("startup-detail");
    const retry = element("startup-retry");
    const update = element("startup-update");
    shell.dataset.state = state;

    if (state === "ready") {
      shell.hidden = true;
      return;
    }

    shell.hidden = false;
    if (state === "loading") {
      if (message) message.textContent = "Загрузка приложения…";
      if (detail) detail.hidden = true;
      if (retry) retry.hidden = true;
      if (update) update.hidden = true;
      return;
    }

    const category = lastFailure?.category || "bootstrap";
    if (message) message.textContent = "Не удалось загрузить приложение";
    if (detail) {
      detail.hidden = false;
      detail.textContent = category === "offline" || category === "network" || category === "timeout"
        ? "Проверьте сеть или VPN и повторите попытку."
        : "Можно повторить загрузку или безопасно обновить файлы приложения.";
    }
    if (retry) retry.hidden = false;
    if (update) update.hidden = false;
  }

  function setPhase(nextPhase) {
    phase = String(nextPhase || "unknown").slice(0, 80);
  }

  function ready() {
    shellReady = true;
    state = "ready";
    render();
  }

  function fail(category, error, options = {}) {
    const failure = diagnostic(category, error, options.phase || phase);
    lastFailure = failure;
    const blocksApplication =
      options.fatal === true || !shellReady || failure.category === "chunk_loading";
    if (blocksApplication) {
      state = "error";
      render();
    }
    return failure;
  }

  function degraded(category, error, failurePhase) {
    return diagnostic(category, error, failurePhase);
  }

  async function updateApplication() {
    const detail = element("startup-detail");
    let attempted = false;
    try {
      attempted = global.sessionStorage?.getItem?.(recoveryKey) === "attempted";
    } catch {
      // Recovery stays user-driven when session storage is unavailable.
    }
    if (attempted) {
      if (detail) {
        detail.hidden = false;
        detail.textContent = "Обновление уже выполнялось. Проверьте сеть или VPN и нажмите «Повторить».";
      }
      return false;
    }

    try {
      global.sessionStorage?.setItem?.(recoveryKey, "attempted");
    } catch {
      // Continue with scoped cleanup; no automatic caller can create a loop.
    }

    const update = element("startup-update");
    if (update) update.disabled = true;
    try {
      const registrations = await global.navigator?.serviceWorker?.getRegistrations?.() || [];
      await Promise.allSettled(
        registrations
          .filter((registration) => registration.scope?.startsWith(`${global.location.origin}/`))
          .map(async (registration) => {
            try {
              await registration.update?.();
            } finally {
              await registration.unregister?.();
            }
          }),
      );
      const names = await global.caches?.keys?.() || [];
      await Promise.all(
        names.filter((name) => name.startsWith(CACHE_PREFIX)).map((name) => global.caches.delete(name)),
      );
      global.location.reload();
      return true;
    } catch (error) {
      fail("service_worker", error, { fatal: true, phase: "recovery" });
      if (update) update.disabled = false;
      return false;
    }
  }

  function bindControls() {
    element("startup-retry")?.addEventListener?.("click", () => global.location.reload());
    element("startup-update")?.addEventListener?.("click", () => void updateApplication());
    render();
  }

  global.addEventListener?.("error", (event) => {
    if (event?.target && event.target !== global) {
      const tagName = String(event.target.tagName || "").toLowerCase();
      fail(tagName === "script" ? "module" : "asset", new Error(`${tagName || "resource"} load failed`));
      return;
    }
    fail("bootstrap", event?.error || new Error("Unhandled startup error"));
  }, true);
  global.addEventListener?.("unhandledrejection", (event) => {
    fail("bootstrap", event?.reason || new Error("Unhandled promise rejection"));
  });

  global.pokerStartup = Object.freeze({
    ready,
    fail,
    degraded,
    setPhase,
    updateApplication,
    shellVersion,
  });

  if (global.document?.readyState === "loading") {
    global.document.addEventListener("DOMContentLoaded", bindControls, { once: true });
  } else {
    bindControls();
  }
})(globalThis);
