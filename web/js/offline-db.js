export const LOCAL_DB_NAME = "poker-session-local";
export const LOCAL_DB_VERSION = 2;
export const LOCAL_RUNTIME_CHANGE_EVENT = "poker-local-runtime-change";

const STORE_SESSION_SNAPSHOTS = "session_snapshots";
const STORE_OUTBOX = "outbox";
const STORE_SYNC_STATE = "sync_state";
const INDEX_OUTBOX_SESSION_SEQUENCE = "by_session_sequence";
const INDEX_OUTBOX_SESSION_STATUS_SEQUENCE = "by_session_status_sequence";

const COMMAND_KINDS = new Set(["buy_in", "cash_out", "reverse_operation"]);
const COMMAND_STATUSES = new Set(["pending", "sending", "blocked", "conflict"]);

function defaultIndexedDB() {
  return globalThis.indexedDB;
}

function isRecord(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

function isIdentifier(value) {
  return typeof value === "string" && value.trim() !== "";
}

function hasOwn(record, key) {
  return Object.prototype.hasOwnProperty.call(record, key);
}

function isTimestamp(value) {
  return (
    (typeof value === "string" && value.trim() !== "") ||
    (typeof value === "number" && Number.isFinite(value))
  );
}

function hasOptionalArray(record, key) {
  return record[key] === undefined || Array.isArray(record[key]);
}

function isOptionalTimestamp(value) {
  return value === null || isTimestamp(value);
}

function timestampMilliseconds(value) {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : null;
}

export function isValidSessionSnapshot(snapshot) {
  return (
    isRecord(snapshot) &&
    isIdentifier(snapshot.session_id) &&
    isRecord(snapshot.session) &&
    Array.isArray(snapshot.players) &&
    Array.isArray(snapshot.operations) &&
    hasOptionalArray(snapshot, "expenses") &&
    hasOptionalArray(snapshot, "settlements") &&
    isTimestamp(snapshot.cached_at) &&
    Number.isInteger(snapshot.local_revision) &&
    snapshot.local_revision >= 0 &&
    isIdentifier(snapshot.last_server_refresh_status)
  );
}

export function isValidOutboxCommand(command) {
  return (
    isRecord(command) &&
    isIdentifier(command.request_id) &&
    isIdentifier(command.session_id) &&
    Number.isInteger(command.sequence) &&
    command.sequence > 0 &&
    COMMAND_KINDS.has(command.kind) &&
    isRecord(command.payload) &&
    isTimestamp(command.created_at) &&
    COMMAND_STATUSES.has(command.status) &&
    Number.isInteger(command.attempts) &&
    command.attempts >= 0 &&
    hasOwn(command, "last_attempt_at") &&
    isOptionalTimestamp(command.last_attempt_at) &&
    hasOwn(command, "next_attempt_at") &&
    isOptionalTimestamp(command.next_attempt_at) &&
    hasOwn(command, "last_error_kind") &&
    (command.last_error_kind === null || typeof command.last_error_kind === "string")
  );
}

function isValidSyncStateRecord(record) {
  return isRecord(record) && isIdentifier(record.key);
}

function requireValidSnapshot(snapshot) {
  if (!isValidSessionSnapshot(snapshot)) {
    throw new TypeError("Invalid session snapshot record");
  }
  return snapshot;
}

function requireValidCommand(command) {
  if (!isValidOutboxCommand(command)) {
    throw new TypeError("Invalid outbox command record");
  }
  return command;
}

function normalizeSyncStateUpdates(updates) {
  if (updates === undefined || updates === null) return [];
  const records = Array.isArray(updates) ? updates : [updates];
  if (!records.every(isValidSyncStateRecord)) {
    throw new TypeError("Invalid sync state record");
  }
  return records;
}

function requestResult(request) {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("IndexedDB request failed"));
  });
}

function transactionCompletion(transaction) {
  return new Promise((resolve, reject) => {
    transaction.oncomplete = () => resolve();
    transaction.onabort = () =>
      reject(transaction.error || new Error("IndexedDB transaction aborted"));
    transaction.onerror = () => {
      // The abort event carries the final transaction outcome.
    };
  });
}

function createStoreIfMissing(database, name, options) {
  if (!database.objectStoreNames.contains(name)) {
    return database.createObjectStore(name, options);
  }
  return null;
}

function applyMigrations(database, transaction, oldVersion, newVersion) {
  if (oldVersion < 1 && newVersion >= 1) {
    createStoreIfMissing(database, STORE_SESSION_SNAPSHOTS, { keyPath: "session_id" });
    createStoreIfMissing(database, STORE_OUTBOX, { keyPath: "request_id" });
    createStoreIfMissing(database, STORE_SYNC_STATE, { keyPath: "key" });
  }

  if (oldVersion < 2 && newVersion >= 2) {
    const outbox = transaction.objectStore(STORE_OUTBOX);
    if (!outbox.indexNames.contains(INDEX_OUTBOX_SESSION_SEQUENCE)) {
      outbox.createIndex(INDEX_OUTBOX_SESSION_SEQUENCE, ["session_id", "sequence"], {
        unique: true,
      });
    }
    if (!outbox.indexNames.contains(INDEX_OUTBOX_SESSION_STATUS_SEQUENCE)) {
      outbox.createIndex(
        INDEX_OUTBOX_SESSION_STATUS_SEQUENCE,
        ["session_id", "status", "sequence"],
        { unique: false },
      );
    }
  }

  transaction.objectStore(STORE_SYNC_STATE).put({
    key: "schema_version",
    version: newVersion,
    migrated_at: new Date().toISOString(),
  });
}

function openDatabase(indexedDBFactory, name, version) {
  if (!indexedDBFactory || typeof indexedDBFactory.open !== "function") {
    return Promise.reject(new Error("IndexedDB is unavailable"));
  }

  return new Promise((resolve, reject) => {
    let request;
    try {
      request = indexedDBFactory.open(name, version);
    } catch (error) {
      reject(error);
      return;
    }

    request.onupgradeneeded = (event) => {
      try {
        applyMigrations(
          request.result,
          request.transaction,
          event.oldVersion || 0,
          event.newVersion || version,
        );
      } catch (error) {
        try {
          request.transaction?.abort();
        } catch {
          // The upgrade transaction may already be aborting.
        }
        reject(error);
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error || new Error("Unable to open IndexedDB"));
    request.onblocked = () => reject(new Error("IndexedDB upgrade is blocked"));
  });
}

function createChangeEvent(detail) {
  if (typeof CustomEvent === "function") {
    return new CustomEvent(LOCAL_RUNTIME_CHANGE_EVENT, { detail });
  }
  const event = new Event(LOCAL_RUNTIME_CHANGE_EVENT);
  Object.defineProperty(event, "detail", { value: detail });
  return event;
}

export function createLocalDatabaseClient({
  indexedDBFactory = defaultIndexedDB(),
  name = LOCAL_DB_NAME,
  version = LOCAL_DB_VERSION,
  eventTarget = new EventTarget(),
  broadcastChannelFactory = globalThis.BroadcastChannel,
} = {}) {
  let databasePromise = null;
  let database = null;
  let broadcastChannel = null;

  function getBroadcastChannel() {
    if (broadcastChannel || typeof broadcastChannelFactory !== "function") {
      return broadcastChannel;
    }
    try {
      broadcastChannel = new broadcastChannelFactory(`${name}-changes`);
      broadcastChannel.addEventListener?.("message", (event) => {
        eventTarget.dispatchEvent(createChangeEvent(event.data));
      });
    } catch {
      broadcastChannel = null;
    }
    return broadcastChannel;
  }

  async function open() {
    if (!databasePromise) {
      databasePromise = openDatabase(indexedDBFactory, name, version)
        .then((openedDatabase) => {
          database = openedDatabase;
          openedDatabase.onversionchange = () => {
            openedDatabase.close();
            if (database === openedDatabase) {
              database = null;
              databasePromise = null;
            }
          };
          getBroadcastChannel();
          return openedDatabase;
        })
        .catch((error) => {
          databasePromise = null;
          throw error;
        });
    }
    return databasePromise;
  }

  async function runTransaction(storeNames, mode, operation) {
    const openedDatabase = await open();
    const transaction = openedDatabase.transaction(storeNames, mode);
    const completion = transactionCompletion(transaction);
    try {
      const result = await operation(transaction);
      await completion;
      return result;
    } catch (error) {
      try {
        transaction.abort();
      } catch {
        // The transaction may already be complete or aborted.
      }
      try {
        await completion;
      } catch {
        // Preserve the request or validation error that caused the abort.
      }
      throw error;
    }
  }

  function emitChange(detail) {
    eventTarget.dispatchEvent(createChangeEvent(detail));
    try {
      getBroadcastChannel()?.postMessage(detail);
    } catch {
      // Same-tab notification already succeeded; cross-tab delivery is best effort.
    }
  }

  async function readSessionSnapshot(sessionId) {
    if (!isIdentifier(sessionId)) return null;
    const record = await runTransaction([STORE_SESSION_SNAPSHOTS], "readonly", (transaction) =>
      requestResult(transaction.objectStore(STORE_SESSION_SNAPSHOTS).get(sessionId)),
    );
    return isValidSessionSnapshot(record) ? record : null;
  }

  async function writeServerSnapshot(snapshot, syncStateUpdates) {
    requireValidSnapshot(snapshot);
    const updates = normalizeSyncStateUpdates(syncStateUpdates);
    const stores = [STORE_SESSION_SNAPSHOTS];
    if (updates.length > 0) stores.push(STORE_SYNC_STATE);

    await runTransaction(stores, "readwrite", async (transaction) => {
      await requestResult(transaction.objectStore(STORE_SESSION_SNAPSHOTS).put(snapshot));
      if (updates.length > 0) {
        const syncStateStore = transaction.objectStore(STORE_SYNC_STATE);
        for (const update of updates) {
          await requestResult(syncStateStore.put(update));
        }
      }
    });
    emitChange({ type: "server_snapshot_written", sessionId: snapshot.session_id });
    return snapshot;
  }

  async function writeServerSnapshotIfRevision(
    snapshot,
    expectedLocalRevision,
    syncStateUpdates,
  ) {
    requireValidSnapshot(snapshot);
    if (!Number.isInteger(expectedLocalRevision) || expectedLocalRevision < 0) {
      throw new TypeError("Expected local revision must be a non-negative integer");
    }
    const updates = normalizeSyncStateUpdates(syncStateUpdates);
    const stores = [STORE_SESSION_SNAPSHOTS];
    if (updates.length > 0) stores.push(STORE_SYNC_STATE);

    const written = await runTransaction(stores, "readwrite", async (transaction) => {
      const snapshotStore = transaction.objectStore(STORE_SESSION_SNAPSHOTS);
      const current = await requestResult(snapshotStore.get(snapshot.session_id));
      const currentRevision = isValidSessionSnapshot(current)
        ? current.local_revision
        : 0;
      if (currentRevision !== expectedLocalRevision) return false;

      await requestResult(snapshotStore.put(snapshot));
      if (updates.length > 0) {
        const syncStateStore = transaction.objectStore(STORE_SYNC_STATE);
        for (const update of updates) {
          await requestResult(syncStateStore.put(update));
        }
      }
      return true;
    });
    if (written) {
      emitChange({ type: "server_snapshot_written", sessionId: snapshot.session_id });
    }
    return written;
  }

  async function writeProjectedSnapshotWithCommand(snapshot, command, syncStateUpdates) {
    requireValidSnapshot(snapshot);
    requireValidCommand(command);
    if (snapshot.session_id !== command.session_id) {
      throw new TypeError("Snapshot and command must belong to the same session");
    }
    const updates = normalizeSyncStateUpdates(syncStateUpdates);

    await runTransaction(
      [STORE_SESSION_SNAPSHOTS, STORE_OUTBOX, STORE_SYNC_STATE],
      "readwrite",
      async (transaction) => {
        await requestResult(transaction.objectStore(STORE_SESSION_SNAPSHOTS).put(snapshot));
        await requestResult(transaction.objectStore(STORE_OUTBOX).put(command));
        const syncStateStore = transaction.objectStore(STORE_SYNC_STATE);
        for (const update of updates) {
          await requestResult(syncStateStore.put(update));
        }
      },
    );
    emitChange({
      type: "local_projection_committed",
      sessionId: snapshot.session_id,
      requestId: command.request_id,
    });
    return { snapshot, command };
  }

  async function deleteSessionSnapshot(sessionId) {
    if (!isIdentifier(sessionId)) {
      throw new TypeError("A session id is required");
    }
    await runTransaction([STORE_SESSION_SNAPSHOTS], "readwrite", (transaction) =>
      requestResult(transaction.objectStore(STORE_SESSION_SNAPSHOTS).delete(sessionId)),
    );
    emitChange({ type: "session_snapshot_deleted", sessionId });
  }

  async function listSessionCommands(sessionId) {
    if (!isIdentifier(sessionId)) return [];
    const records = await runTransaction([STORE_OUTBOX], "readonly", (transaction) =>
      requestResult(
        transaction
          .objectStore(STORE_OUTBOX)
          .index(INDEX_OUTBOX_SESSION_SEQUENCE)
          .getAll(),
      ),
    );
    return records
      .filter((record) => isValidOutboxCommand(record) && record.session_id === sessionId)
      .sort((left, right) => left.sequence - right.sequence);
  }

  async function listPendingCommands(sessionId) {
    const commands = await listSessionCommands(sessionId);
    return commands.filter((command) => command.status === "pending");
  }

  async function listSessionProjectionCommands(sessionId, { excludeRequestId = "" } = {}) {
    const commands = await listSessionCommands(sessionId);
    return commands.filter(
      (command) =>
        command.request_id !== excludeRequestId &&
        (command.kind === "buy_in" || command.kind === "cash_out"),
    );
  }

  async function claimNextReplayCommand({
    now = new Date().toISOString(),
    leaseTimeoutMs,
    allowEarlyRetry = false,
  }) {
    const nowMs = timestampMilliseconds(now);
    if (nowMs === null) throw new TypeError("A valid replay timestamp is required");
    if (!Number.isFinite(leaseTimeoutMs) || leaseTimeoutMs <= 0) {
      throw new TypeError("leaseTimeoutMs must be a positive finite number");
    }

    const result = await runTransaction([STORE_OUTBOX], "readwrite", async (transaction) => {
      const store = transaction.objectStore(STORE_OUTBOX);
      const records = (await requestResult(store.getAll())).filter(isValidOutboxCommand);
      const sessions = new Map();

      for (const record of records) {
        let command = record;
        const lastAttemptMs = timestampMilliseconds(command.last_attempt_at);
        if (
          command.status === "sending" &&
          lastAttemptMs !== null &&
          lastAttemptMs <= nowMs - leaseTimeoutMs
        ) {
          command = { ...command, status: "pending" };
          await requestResult(store.put(command));
        }
        const sessionCommands = sessions.get(command.session_id) || [];
        sessionCommands.push(command);
        sessions.set(command.session_id, sessionCommands);
      }

      const candidates = [];
      let nextAttemptAt = null;
      let blockedErrorKind = null;
      for (const sessionCommands of sessions.values()) {
        sessionCommands.sort((left, right) => left.sequence - right.sequence);
        const command = sessionCommands[0];
        if (command.status === "blocked" || command.status === "conflict") {
          blockedErrorKind ||= command.last_error_kind;
          continue;
        }
        if (command.status !== "pending") continue;

        const commandNextAttemptMs = timestampMilliseconds(command.next_attempt_at);
        if (
          !allowEarlyRetry &&
          commandNextAttemptMs !== null &&
          commandNextAttemptMs > nowMs
        ) {
          if (nextAttemptAt === null || commandNextAttemptMs < nextAttemptAt) {
            nextAttemptAt = commandNextAttemptMs;
          }
          continue;
        }
        candidates.push(command);
      }

      candidates.sort((left, right) => {
        const createdDifference =
          (timestampMilliseconds(left.created_at) || 0) -
          (timestampMilliseconds(right.created_at) || 0);
        if (createdDifference !== 0) return createdDifference;
        const sessionDifference = left.session_id.localeCompare(right.session_id);
        return sessionDifference || left.sequence - right.sequence;
      });

      const selected = candidates[0];
      if (!selected) {
        return { command: null, nextAttemptAt, blockedErrorKind };
      }
      const claimed = {
        ...selected,
        status: "sending",
        last_attempt_at: now,
      };
      await requestResult(store.put(claimed));
      return { command: claimed, nextAttemptAt: null, blockedErrorKind: null };
    });

    if (result.command) {
      emitChange({
        type: "outbox_command_sending",
        sessionId: result.command.session_id,
        requestId: result.command.request_id,
      });
    }
    return result;
  }

  async function retryOutboxCommand({
    requestId,
    attempts,
    lastAttemptAt,
    nextAttemptAt,
    errorKind,
    errorDetails = null,
  }) {
    if (!isIdentifier(requestId) || !Number.isInteger(attempts) || attempts <= 0) {
      throw new TypeError("A request id and positive attempts count are required");
    }
    const updated = await runTransaction([STORE_OUTBOX], "readwrite", async (transaction) => {
      const store = transaction.objectStore(STORE_OUTBOX);
      const command = await requestResult(store.get(requestId));
      if (!isValidOutboxCommand(command)) return null;
      const next = {
        ...command,
        status: "pending",
        attempts,
        last_attempt_at: lastAttemptAt,
        next_attempt_at: nextAttemptAt,
        last_error_kind: errorKind,
        last_error_details: errorDetails,
      };
      requireValidCommand(next);
      await requestResult(store.put(next));
      return next;
    });
    if (updated) {
      emitChange({
        type: "outbox_command_retry_scheduled",
        sessionId: updated.session_id,
        requestId: updated.request_id,
      });
    }
    return updated;
  }

  async function blockOutboxCommand({
    requestId,
    attempts,
    lastAttemptAt,
    errorKind,
    errorDetails = null,
    conflict = false,
  }) {
    if (!isIdentifier(requestId) || !Number.isInteger(attempts) || attempts <= 0) {
      throw new TypeError("A request id and positive attempts count are required");
    }
    const updated = await runTransaction([STORE_OUTBOX], "readwrite", async (transaction) => {
      const store = transaction.objectStore(STORE_OUTBOX);
      const command = await requestResult(store.get(requestId));
      if (!isValidOutboxCommand(command)) return null;
      const next = {
        ...command,
        status: conflict ? "conflict" : "blocked",
        attempts,
        last_attempt_at: lastAttemptAt,
        next_attempt_at: null,
        last_error_kind: errorKind,
        last_error_details: errorDetails,
      };
      requireValidCommand(next);
      await requestResult(store.put(next));
      return next;
    });
    if (updated) {
      emitChange({
        type: "outbox_command_blocked",
        sessionId: updated.session_id,
        requestId: updated.request_id,
      });
    }
    return updated;
  }

  async function reconcileOutboxCommand({
    requestId,
    sessionId,
    snapshot,
    expectedLocalRevision,
    acknowledgement = null,
    completedAt = new Date().toISOString(),
  }) {
    requireValidSnapshot(snapshot);
    if (!isIdentifier(requestId) || snapshot.session_id !== sessionId) {
      throw new TypeError("Reconciliation identity does not match the snapshot");
    }
    if (!Number.isInteger(expectedLocalRevision) || expectedLocalRevision < 0) {
      throw new TypeError("Expected local revision must be a non-negative integer");
    }

    const reconciled = await runTransaction(
      [STORE_SESSION_SNAPSHOTS, STORE_OUTBOX, STORE_SYNC_STATE],
      "readwrite",
      async (transaction) => {
        const snapshots = transaction.objectStore(STORE_SESSION_SNAPSHOTS);
        const outbox = transaction.objectStore(STORE_OUTBOX);
        const syncState = transaction.objectStore(STORE_SYNC_STATE);
        const [currentSnapshot, command, commands] = await Promise.all([
          requestResult(snapshots.get(sessionId)),
          requestResult(outbox.get(requestId)),
          requestResult(outbox.getAll()),
        ]);
        if (
          !isValidSessionSnapshot(currentSnapshot) ||
          currentSnapshot.local_revision !== expectedLocalRevision ||
          !isValidOutboxCommand(command) ||
          command.session_id !== sessionId
        ) {
          return false;
        }

        await requestResult(snapshots.put(snapshot));
        await requestResult(outbox.delete(requestId));
        const pendingCount = commands.filter(
          (item) =>
            isValidOutboxCommand(item) &&
            item.request_id !== requestId &&
            (item.status === "pending" || item.status === "sending"),
        ).length;
        await requestResult(syncState.put({
          key: "last_successful_replay",
          request_id: requestId,
          session_id: sessionId,
          acknowledgement,
          completed_at: completedAt,
        }));
        await requestResult(syncState.put({ key: "pending_count", value: pendingCount }));
        return true;
      },
    );
    if (reconciled) {
      emitChange({ type: "outbox_command_reconciled", sessionId, requestId });
    }
    return reconciled;
  }

  async function nextSessionCommandSequence(sessionId) {
    if (!isIdentifier(sessionId)) {
      throw new TypeError("A session id is required");
    }
    const commands = await listSessionCommands(sessionId);
    return commands.reduce(
      (maximum, command) => Math.max(maximum, command.sequence),
      0,
    ) + 1;
  }

  async function countPendingAndBlockedCommands(sessionId) {
    const records = isIdentifier(sessionId)
      ? await listSessionCommands(sessionId)
      : await runTransaction([STORE_OUTBOX], "readonly", (transaction) =>
          requestResult(transaction.objectStore(STORE_OUTBOX).getAll()),
        );
    return records.reduce(
      (counts, command) => {
        if (!isValidOutboxCommand(command)) return counts;
        if (command.status === "pending" || command.status === "sending") {
          counts.pending += 1;
        }
        if (command.status === "blocked" || command.status === "conflict") {
          counts.blocked += 1;
        }
        return counts;
      },
      { pending: 0, blocked: 0 },
    );
  }

  function subscribeLocalRuntimeChanges(listener) {
    eventTarget.addEventListener(LOCAL_RUNTIME_CHANGE_EVENT, listener);
    return () => eventTarget.removeEventListener(LOCAL_RUNTIME_CHANGE_EVENT, listener);
  }

  function close() {
    database?.close();
    database = null;
    databasePromise = null;
    broadcastChannel?.close?.();
    broadcastChannel = null;
  }

  return Object.freeze({
    open,
    close,
    readSessionSnapshot,
    writeServerSnapshot,
    writeServerSnapshotIfRevision,
    writeProjectedSnapshotWithCommand,
    deleteSessionSnapshot,
    listPendingCommands,
    listSessionProjectionCommands,
    claimNextReplayCommand,
    retryOutboxCommand,
    blockOutboxCommand,
    reconcileOutboxCommand,
    nextSessionCommandSequence,
    countPendingAndBlockedCommands,
    subscribeLocalRuntimeChanges,
  });
}

const defaultClient = createLocalDatabaseClient();

export const initializeLocalDatabase = defaultClient.open;
export const closeLocalDatabase = defaultClient.close;
export const readSessionSnapshot = defaultClient.readSessionSnapshot;
export const writeServerSnapshot = defaultClient.writeServerSnapshot;
export const writeServerSnapshotIfRevision = defaultClient.writeServerSnapshotIfRevision;
export const writeProjectedSnapshotWithCommand =
  defaultClient.writeProjectedSnapshotWithCommand;
export const deleteSessionSnapshot = defaultClient.deleteSessionSnapshot;
export const listPendingCommands = defaultClient.listPendingCommands;
export const listSessionProjectionCommands = defaultClient.listSessionProjectionCommands;
export const claimNextReplayCommand = defaultClient.claimNextReplayCommand;
export const retryOutboxCommand = defaultClient.retryOutboxCommand;
export const blockOutboxCommand = defaultClient.blockOutboxCommand;
export const reconcileOutboxCommand = defaultClient.reconcileOutboxCommand;
export const nextSessionCommandSequence = defaultClient.nextSessionCommandSequence;
export const countPendingAndBlockedCommands = defaultClient.countPendingAndBlockedCommands;
export const subscribeLocalRuntimeChanges = defaultClient.subscribeLocalRuntimeChanges;
