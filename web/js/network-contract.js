export const READ_TIMEOUT_MS = 5_000;
export const WRITE_TIMEOUT_MS = 8_000;

export const ERROR_KINDS = Object.freeze({
  NONE: "none",
  OFFLINE: "offline",
  TIMEOUT: "timeout",
  NETWORK: "network",
  AUTHORIZATION: "authorization",
  RETRYABLE_HTTP: "retryable_http",
  DOMAIN: "domain",
  INVALID_RESPONSE: "invalid_response",
});

export const RETRY_ACTIONS = Object.freeze({
  ACCEPTED: "accepted",
  RETRY: "retry",
  BLOCK_AUTHORIZATION: "block_authorization",
  BLOCK_DOMAIN: "block_domain",
});

const RETRYABLE_STATUSES = new Set([408, 425, 429]);

function defaultFetch(...args) {
  if (typeof globalThis.fetch !== "function") {
    return Promise.reject(new Error("Fetch is unavailable"));
  }
  return globalThis.fetch(...args);
}

function defaultOnlineCheck() {
  return globalThis.navigator?.onLine !== false;
}

function isWriteMethod(method) {
  const normalized = String(method || "GET").toUpperCase();
  return normalized !== "GET" && normalized !== "HEAD";
}

function defaultTimeoutForMethod(method) {
  return isWriteMethod(method) ? WRITE_TIMEOUT_MS : READ_TIMEOUT_MS;
}

function validateTimeout(timeoutMs) {
  if (!Number.isFinite(timeoutMs) || timeoutMs < 0) {
    throw new TypeError("timeoutMs must be a non-negative finite number");
  }
  return timeoutMs;
}

export function classifyHTTPResponse({ status, ok, invalidJSON = false }) {
  if (invalidJSON) return ERROR_KINDS.INVALID_RESPONSE;
  if (ok) return ERROR_KINDS.NONE;
  if (status === 401 || status === 403) return ERROR_KINDS.AUTHORIZATION;
  if (RETRYABLE_STATUSES.has(status) || status >= 500) {
    return ERROR_KINDS.RETRYABLE_HTTP;
  }
  if (status >= 400 && status < 500) return ERROR_KINDS.DOMAIN;
  return ERROR_KINDS.INVALID_RESPONSE;
}

export function retryPolicy(errorKind, status = 0) {
  if (errorKind === ERROR_KINDS.NONE && status >= 200 && status < 300) {
    return RETRY_ACTIONS.ACCEPTED;
  }
  if (errorKind === ERROR_KINDS.AUTHORIZATION || status === 401 || status === 403) {
    return RETRY_ACTIONS.BLOCK_AUTHORIZATION;
  }
  if (
    errorKind === ERROR_KINDS.OFFLINE ||
    errorKind === ERROR_KINDS.TIMEOUT ||
    errorKind === ERROR_KINDS.NETWORK ||
    errorKind === ERROR_KINDS.RETRYABLE_HTTP ||
    RETRYABLE_STATUSES.has(status) ||
    status >= 500
  ) {
    return RETRY_ACTIONS.RETRY;
  }
  if (errorKind === ERROR_KINDS.DOMAIN || (status >= 400 && status < 500)) {
    return RETRY_ACTIONS.BLOCK_DOMAIN;
  }
  return RETRY_ACTIONS.RETRY;
}

export function createRequestClient({
  baseURL = "",
  fetchImpl = defaultFetch,
  isOnline = defaultOnlineCheck,
  AbortControllerImpl = globalThis.AbortController,
  setTimeoutImpl = globalThis.setTimeout.bind(globalThis),
  clearTimeoutImpl = globalThis.clearTimeout.bind(globalThis),
} = {}) {
  return async function request(path, options = {}) {
    const method = String(options.method || "GET").toUpperCase();
    const timeoutMs = validateTimeout(
      options.timeoutMs ?? defaultTimeoutForMethod(method),
    );
    const { timeoutMs: _timeoutMs, ...fetchOptions } = options;
    const externalSignal = fetchOptions.signal;
    const controller =
      typeof AbortControllerImpl === "function" ? new AbortControllerImpl() : null;
    let timedOut = false;
    let timeoutID = null;
    let removeExternalAbort = null;

    if (controller) {
      fetchOptions.signal = controller.signal;
      if (externalSignal) {
        const abortFromCaller = () => controller.abort(externalSignal.reason);
        if (externalSignal.aborted) {
          abortFromCaller();
        } else if (typeof externalSignal.addEventListener === "function") {
          externalSignal.addEventListener("abort", abortFromCaller, { once: true });
          removeExternalAbort = () =>
            externalSignal.removeEventListener("abort", abortFromCaller);
        }
      }
    }

    const requestOptions = {
      credentials: "same-origin",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      ...fetchOptions,
      method,
    };

    const timeoutPromise =
      timeoutMs > 0
        ? new Promise((_, reject) => {
            timeoutID = setTimeoutImpl(() => {
              timedOut = true;
              controller?.abort();
              const error = new Error(`Request timed out after ${timeoutMs}ms`);
              error.name = "TimeoutError";
              reject(error);
            }, timeoutMs);
          })
        : null;

    try {
      const fetchPromise = Promise.resolve().then(() =>
        fetchImpl(`${baseURL}${path}`, requestOptions),
      );
      const response = await (timeoutPromise
        ? Promise.race([fetchPromise, timeoutPromise])
        : fetchPromise);

      let text = "";
      try {
        text = await response.text();
      } catch (error) {
        return {
          ok: response.ok,
          status: response.status,
          body: null,
          text: String(error),
          errorKind: ERROR_KINDS.INVALID_RESPONSE,
        };
      }

      let body = null;
      let invalidJSON = false;
      if (text) {
        try {
          body = JSON.parse(text);
        } catch {
          invalidJSON = true;
        }
      }

      return {
        ok: response.ok,
        status: response.status,
        body,
        text,
        errorKind: classifyHTTPResponse({
          status: response.status,
          ok: response.ok,
          invalidJSON,
        }),
      };
    } catch (error) {
      let errorKind = ERROR_KINDS.NETWORK;
      if (timedOut || error?.name === "TimeoutError") {
        errorKind = ERROR_KINDS.TIMEOUT;
      } else if (!isOnline()) {
        errorKind = ERROR_KINDS.OFFLINE;
      }
      return {
        ok: false,
        status: 0,
        body: null,
        text: String(error),
        errorKind,
      };
    } finally {
      if (timeoutID !== null) clearTimeoutImpl(timeoutID);
      removeExternalAbort?.();
    }
  };
}

export function createRequestId() {
  if (
    typeof globalThis.crypto !== "undefined" &&
    typeof globalThis.crypto.randomUUID === "function"
  ) {
    return globalThis.crypto.randomUUID();
  }

  if (
    typeof globalThis.crypto !== "undefined" &&
    typeof globalThis.crypto.getRandomValues === "function"
  ) {
    const bytes = new Uint8Array(16);
    globalThis.crypto.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = [...bytes].map((byte) => byte.toString(16).padStart(2, "0"));
    return [
      hex.slice(0, 4).join(""),
      hex.slice(4, 6).join(""),
      hex.slice(6, 8).join(""),
      hex.slice(8, 10).join(""),
      hex.slice(10, 16).join(""),
    ].join("-");
  }

  return `req-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
}

function requireIdentifier(value, field) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new TypeError(`${field} must be a non-empty string`);
  }
  return value.trim();
}

function requireRequestId(requestId) {
  if (typeof requestId !== "string" || requestId.trim() === "") {
    throw new TypeError("requestId must be a non-empty string");
  }
  return requestId;
}

function requireChips(chips) {
  const normalized = Number(chips);
  if (!Number.isFinite(normalized) || normalized <= 0) {
    throw new TypeError("chips must be a positive finite number");
  }
  return normalized;
}

function commandEnvelope(requestId, payload) {
  return Object.freeze({
    requestId,
    payload: Object.freeze(payload),
  });
}

export function serializeBuyInCommand({ sessionId, playerId, chips, requestId }) {
  const ownedRequestId = requireRequestId(requestId ?? createRequestId());
  return commandEnvelope(ownedRequestId, {
    session_id: requireIdentifier(sessionId, "sessionId"),
    player_id: requireIdentifier(playerId, "playerId"),
    chips: requireChips(chips),
    request_id: ownedRequestId,
  });
}

export function serializeCashOutCommand({ sessionId, playerId, chips, requestId }) {
  const ownedRequestId = requireRequestId(requestId ?? createRequestId());
  return commandEnvelope(ownedRequestId, {
    session_id: requireIdentifier(sessionId, "sessionId"),
    player_id: requireIdentifier(playerId, "playerId"),
    chips: requireChips(chips),
    request_id: ownedRequestId,
  });
}

export function serializeReverseOperationCommand({ operationId, requestId }) {
  const ownedRequestId = requireRequestId(requestId ?? createRequestId());
  return commandEnvelope(ownedRequestId, {
    target_operation_id: requireIdentifier(operationId, "operationId"),
    request_id: ownedRequestId,
  });
}
