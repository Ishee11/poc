import {
  ERROR_KINDS,
  RETRY_ACTIONS,
  retryPolicy,
  serializeBuyInCommand,
  serializeCashOutCommand,
  serializeReverseOperationCommand,
} from "./network-contract.js";

export const SENDING_LEASE_MS = 30_000;
export const RETRY_BASE_DELAY_MS = 1_000;
export const RETRY_MAX_DELAY_MS = 30_000;

export const REPLAY_STATUSES = Object.freeze({
  IDLE: "idle",
  SYNCING: "syncing",
  WAITING: "waiting_for_retry",
  AUTHORIZATION_BLOCKED: "authorization_blocked",
  DOMAIN_BLOCKED: "domain_blocked",
});

function timestampMilliseconds(value) {
  if (typeof value === "number") return Number.isFinite(value) ? value : NaN;
  if (value instanceof Date) return value.getTime();
  return Date.parse(value);
}

function timestampISO(value) {
  const milliseconds = timestampMilliseconds(value);
  if (!Number.isFinite(milliseconds)) throw new TypeError("Replay clock is invalid");
  return new Date(milliseconds).toISOString();
}

function canonicalRecord(record) {
  if (record === null || typeof record !== "object" || Array.isArray(record)) {
    throw new TypeError("Command payload must be an object");
  }
  return JSON.stringify(
    Object.fromEntries(Object.keys(record).sort().map((key) => [key, record[key]])),
  );
}

export function validatePersistedSessionCommand(command) {
  if (command?.status !== "sending") {
    throw new TypeError("Command is not leased for sending");
  }
  const input = command.kind === "reverse_operation"
    ? {
        operationId: command.payload?.target_operation_id,
        requestId: command.request_id,
      }
    : {
        sessionId: command.session_id,
        playerId: command.payload?.player_id,
        chips: command.payload?.chips,
        requestId: command.request_id,
      };
  const serialized = command.kind === "buy_in"
    ? serializeBuyInCommand(input)
    : command.kind === "cash_out"
      ? serializeCashOutCommand(input)
      : command.kind === "reverse_operation"
        ? serializeReverseOperationCommand(input)
        : null;
  if (!serialized) throw new TypeError("Unsupported replay command kind");
  if (
    serialized.requestId !== command.request_id ||
    (command.kind !== "reverse_operation" &&
      serialized.payload.session_id !== command.session_id) ||
    canonicalRecord(serialized.payload) !== canonicalRecord(command.payload)
  ) {
    throw new TypeError("Persisted command payload does not match its identity");
  }
  return serialized;
}

export function computeReplayBackoffMs(
  attempts,
  {
    baseDelayMs = RETRY_BASE_DELAY_MS,
    maxDelayMs = RETRY_MAX_DELAY_MS,
    random = Math.random,
  } = {},
) {
  if (!Number.isInteger(attempts) || attempts <= 0) {
    throw new TypeError("attempts must be a positive integer");
  }
  const exponential = Math.min(maxDelayMs, baseDelayMs * (2 ** (attempts - 1)));
  const jitter = 0.75 + Math.max(0, Math.min(1, Number(random()) || 0)) * 0.5;
  return Math.min(maxDelayMs, Math.max(1, Math.round(exponential * jitter)));
}

function serverErrorDetails(result) {
  const body = result?.body;
  return {
    status: Number(result?.status) || 0,
    code: typeof body?.code === "string"
      ? body.code
      : typeof body?.error === "string"
        ? body.error
        : null,
    message: typeof body?.message === "string" ? body.message : null,
    details: body?.details ?? null,
  };
}

export function createOutboxReplay({
  store,
  send,
  reconcile,
  onStatus = () => {},
  now = () => Date.now(),
  random = Math.random,
  setTimeoutImpl = globalThis.setTimeout.bind(globalThis),
  clearTimeoutImpl = globalThis.clearTimeout.bind(globalThis),
  isActive = () => true,
  leaseTimeoutMs = SENDING_LEASE_MS,
  baseDelayMs = RETRY_BASE_DELAY_MS,
  maxDelayMs = RETRY_MAX_DELAY_MS,
}) {
  let activeReplay = null;
  let replayRequested = false;
  let earlyRetryRequested = false;
  let retryTimer = null;
  let disposed = false;
  let lastSuccessfulReplayAt = null;

  function clearRetryTimer() {
    if (retryTimer !== null) clearTimeoutImpl(retryTimer);
    retryTimer = null;
  }

  async function publishStatus(status) {
    const [counts, diagnostics] = await Promise.all([
      store.countPendingAndBlockedCommands(),
      store.readReplayDiagnostics?.() || null,
    ]);
    onStatus({
      status,
      pendingCount: counts.pending,
      blockedCount: counts.blocked,
      lastSuccessfulReplayAt,
      errorDetails: diagnostics?.errorDetails || null,
    });
  }

  function scheduleRetry(nextAttemptAt) {
    clearRetryTimer();
    if (disposed || !isActive() || !Number.isFinite(nextAttemptAt)) return;
    const delay = Math.max(0, nextAttemptAt - timestampMilliseconds(now()));
    retryTimer = setTimeoutImpl(() => {
      retryTimer = null;
      void requestReplay();
    }, delay);
  }

  async function retryCommand(command, result, errorKind = result?.errorKind) {
    const attempts = command.attempts + 1;
    const attemptedAt = timestampISO(now());
    const delay = computeReplayBackoffMs(attempts, {
      baseDelayMs,
      maxDelayMs,
      random,
    });
    const nextAttemptAt = timestampMilliseconds(attemptedAt) + delay;
    await store.retryOutboxCommand({
      requestId: command.request_id,
      attempts,
      lastAttemptAt: attemptedAt,
      nextAttemptAt: new Date(nextAttemptAt).toISOString(),
      errorKind,
      errorDetails: serverErrorDetails(result),
    });
    await publishStatus(REPLAY_STATUSES.WAITING);
    scheduleRetry(nextAttemptAt);
  }

  async function blockCommand(command, result, action, errorKind = result?.errorKind) {
    await store.blockOutboxCommand({
      requestId: command.request_id,
      attempts: command.attempts + 1,
      lastAttemptAt: timestampISO(now()),
      errorKind,
      errorDetails: serverErrorDetails(result),
      conflict: action === RETRY_ACTIONS.BLOCK_DOMAIN,
    });
    await publishStatus(
      action === RETRY_ACTIONS.BLOCK_AUTHORIZATION
        ? REPLAY_STATUSES.AUTHORIZATION_BLOCKED
        : REPLAY_STATUSES.DOMAIN_BLOCKED,
    );
  }

  async function runReplayLoop(allowEarlyRetry) {
    clearRetryTimer();
    while (!disposed) {
      const claim = await store.claimNextReplayCommand({
        now: timestampISO(now()),
        leaseTimeoutMs,
        allowEarlyRetry,
      });
      if (!claim.command) {
        if (claim.blockedErrorKind === ERROR_KINDS.AUTHORIZATION) {
          await publishStatus(REPLAY_STATUSES.AUTHORIZATION_BLOCKED);
        } else if (claim.blockedErrorKind) {
          await publishStatus(REPLAY_STATUSES.DOMAIN_BLOCKED);
        } else if (Number.isFinite(claim.nextAttemptAt)) {
          await publishStatus(REPLAY_STATUSES.WAITING);
        } else {
          await publishStatus(REPLAY_STATUSES.IDLE);
        }
        if (Number.isFinite(claim.nextAttemptAt)) scheduleRetry(claim.nextAttemptAt);
        return;
      }

      const command = claim.command;
      await publishStatus(REPLAY_STATUSES.SYNCING);
      let serialized;
      try {
        serialized = validatePersistedSessionCommand(command);
      } catch (error) {
        await blockCommand(
          command,
          { status: 0, body: { code: "invalid_local_command", message: error.message } },
          RETRY_ACTIONS.BLOCK_DOMAIN,
          ERROR_KINDS.DOMAIN,
        );
        return;
      }

      let result;
      try {
        result = await send(command, serialized);
      } catch (error) {
        result = {
          ok: false,
          status: 0,
          body: null,
          errorKind: ERROR_KINDS.NETWORK,
          text: String(error),
        };
      }
      const action = retryPolicy(result.errorKind, result.status);
      if (action === RETRY_ACTIONS.RETRY) {
        await retryCommand(command, result);
        return;
      }
      if (action !== RETRY_ACTIONS.ACCEPTED) {
        await blockCommand(command, result, action);
        return;
      }

      try {
        const reconciled = await reconcile(command, result);
        if (reconciled === false) throw new Error("Reconciliation revision changed");
      } catch (error) {
        await retryCommand(
          command,
          { status: 0, body: { code: "reconciliation_failed", message: error.message } },
          ERROR_KINDS.INVALID_RESPONSE,
        );
        return;
      }
      lastSuccessfulReplayAt = timestampISO(now());
    }
  }

  function requestReplay({ allowEarlyRetry = false } = {}) {
    if (disposed) return Promise.resolve();
    replayRequested = true;
    earlyRetryRequested ||= allowEarlyRetry;
    if (allowEarlyRetry) clearRetryTimer();
    if (activeReplay) return activeReplay;

    activeReplay = Promise.resolve()
      .then(async () => {
        while (replayRequested && !disposed) {
          replayRequested = false;
          const allowEarly = earlyRetryRequested;
          earlyRetryRequested = false;
          await runReplayLoop(allowEarly);
        }
      })
      .finally(() => {
        activeReplay = null;
        if (replayRequested && !disposed) void requestReplay();
      });
    return activeReplay;
  }

  function dispose() {
    disposed = true;
    clearRetryTimer();
  }

  return Object.freeze({ requestReplay, dispose });
}
