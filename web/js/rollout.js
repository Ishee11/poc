export const LOCAL_FIRST_SESSION_WRITES_KEY = "poker-local-first-session-writes";

export function resolveLocalFirstSessionWrites({ storage, documentRef } = {}) {
  let stored = null;
  try {
    stored = storage?.getItem?.(LOCAL_FIRST_SESSION_WRITES_KEY) ?? null;
  } catch {
    return Object.freeze({ enabled: false, source: "storage_unavailable", raw: null });
  }
  if (stored === "true" || stored === "false") {
    return Object.freeze({ enabled: stored === "true", source: "local_storage", raw: stored });
  }

  const configured = documentRef
    ?.querySelector?.('meta[name="poker-local-first-session-writes"]')
    ?.getAttribute?.("content");
  if (configured === "true" || configured === "false") {
    return Object.freeze({ enabled: configured === "true", source: "deployment", raw: configured });
  }
  return Object.freeze({ enabled: false, source: "unknown", raw: configured ?? stored });
}

export function shouldUseLocalFirstSessionWrites({ enabled, runtimeStatus }) {
  return enabled === true && runtimeStatus === "available";
}
