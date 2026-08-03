const LOCAL_COMMAND_KINDS = new Set(["buy_in", "cash_out"]);
export const SESSION_REPLAY_REQUEST_EVENT = "poker-session-replay-request";

export const REVERSE_TARGET_KINDS = Object.freeze({
  PENDING_UNSENT: "pending_unsent",
  POSSIBLY_SENT: "possibly_sent",
  SERVER_CONFIRMED: "server_confirmed",
  ALREADY_REVERSED: "already_reversed",
  UNAVAILABLE: "unavailable",
});

export class LocalProjectionError extends Error {
  constructor(code, message) {
    super(message);
    this.name = "LocalProjectionError";
    this.code = code;
  }
}

function requireRecord(value, code, message) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) {
    throw new LocalProjectionError(code, message);
  }
  return value;
}

function requireIdentifier(value, field) {
  if (typeof value !== "string" || value.trim() === "") {
    throw new LocalProjectionError("invalid_command", `${field} is required`);
  }
  return value.trim();
}

function requireChips(value) {
  const chips = Number(value);
  if (!Number.isSafeInteger(chips) || chips <= 0) {
    throw new LocalProjectionError(
      "invalid_chips",
      "chips must be a positive safe integer",
    );
  }
  return chips;
}

function requireSequence(value) {
  if (!Number.isInteger(value) || value <= 0) {
    throw new LocalProjectionError("invalid_command", "sequence must be positive");
  }
  return value;
}

function playerIdentifier(player) {
  return String(player?.player_id || player?.id || "");
}

function projectedProfit(profitChips, chipRate) {
  const rate = Number(chipRate);
  return Number.isFinite(rate) && rate > 0 ? Math.trunc(profitChips / rate) : 0;
}

function effectivePlayerInGame(operations, playerId, excludedOperationId) {
  const reversedTargets = new Set(
    operations
      .filter((operation) => operation.type === "reversal" && operation.reference_id)
      .map((operation) => operation.reference_id),
  );
  const latest = operations.find(
    (operation) =>
      operation.id !== excludedOperationId &&
      operation.player_id === playerId &&
      (operation.type === "buy_in" || operation.type === "cash_out") &&
      !reversedTargets.has(operation.id),
  );
  return latest?.type === "buy_in";
}

function inverseOperation(snapshot, target, { removeTarget = false } = {}) {
  const chips = requireChips(target?.chips);
  const playerId = requireIdentifier(target?.player_id, "player_id");
  if (target.type !== "buy_in" && target.type !== "cash_out") {
    throw new LocalProjectionError("invalid_reverse_target", "operation cannot be reversed");
  }
  const playerIndex = snapshot.players.findIndex(
    (player) => playerIdentifier(player) === playerId,
  );
  if (playerIndex < 0) {
    throw new LocalProjectionError("player_not_found", "player is not cached");
  }

  const totalBuyIn =
    (Number(snapshot.session.totalBuyIn) || 0) - (target.type === "buy_in" ? chips : 0);
  const totalCashOut =
    (Number(snapshot.session.totalCashOut) || 0) - (target.type === "cash_out" ? chips : 0);
  if (totalBuyIn < 0 || totalCashOut < 0 || totalBuyIn - totalCashOut < 0) {
    throw new LocalProjectionError("invalid_reverse_target", "reverse would break totals");
  }

  const players = snapshot.players.map((player, index) => {
    if (index !== playerIndex) return { ...player };
    const buyIn = (Number(player.buy_in) || 0) - (target.type === "buy_in" ? chips : 0);
    const cashOut =
      (Number(player.cash_out) || 0) - (target.type === "cash_out" ? chips : 0);
    if (buyIn < 0 || cashOut < 0) {
      throw new LocalProjectionError("invalid_reverse_target", "reverse would break player totals");
    }
    const profitChips = cashOut - buyIn;
    return {
      ...player,
      buy_in: buyIn,
      cash_out: cashOut,
      profit_chips: profitChips,
      profit_money: projectedProfit(profitChips, snapshot.session.chipRate),
      in_game: target.type === "cash_out"
        ? true
        : effectivePlayerInGame(snapshot.operations, playerId, target.id),
    };
  });

  return {
    ...snapshot,
    session: {
      ...snapshot.session,
      totalBuyIn,
      totalCashOut,
      totalChips: totalBuyIn - totalCashOut,
    },
    players,
    operations: removeTarget
      ? snapshot.operations.filter((operation) => operation.id !== target.id)
      : snapshot.operations.map((operation) => ({ ...operation })),
    local_revision: (Number(snapshot.local_revision) || 0) + 1,
    last_server_refresh_status: "local",
  };
}

function validateProjectionInput(snapshot, command) {
  requireRecord(snapshot, "invalid_snapshot", "session snapshot is required");
  const session = requireRecord(
    snapshot.session,
    "invalid_snapshot",
    "normalized session is required",
  );
  if (!Array.isArray(snapshot.players) || !Array.isArray(snapshot.operations)) {
    throw new LocalProjectionError("invalid_snapshot", "players and operations are required");
  }
  if (session.status !== "active") {
    throw new LocalProjectionError("session_not_active", "session is not active");
  }

  const kind = command?.kind;
  if (!LOCAL_COMMAND_KINDS.has(kind)) {
    throw new LocalProjectionError("invalid_command", "unsupported command kind");
  }
  const sessionId = requireIdentifier(command.session_id, "session_id");
  if (snapshot.session_id !== sessionId || session.id !== sessionId) {
    throw new LocalProjectionError("session_mismatch", "command belongs to another session");
  }
  const playerId = requireIdentifier(command.player_id, "player_id");
  const playerIndex = snapshot.players.findIndex(
    (player) => playerIdentifier(player) === playerId,
  );
  if (playerIndex < 0) {
    throw new LocalProjectionError("player_not_found", "player is not cached");
  }

  const chips = requireChips(command.chips);
  if (kind === "cash_out") {
    if (!snapshot.players[playerIndex].in_game) {
      throw new LocalProjectionError("player_not_in_game", "player is not in game");
    }
    if (chips > (Number(session.totalChips) || 0)) {
      throw new LocalProjectionError(
        "invalid_cash_out",
        "cash out exceeds chips on table",
      );
    }
  }

  return { session, kind, sessionId, playerId, playerIndex, chips };
}

function provisionalOperation(command, chips) {
  return {
    id: requireIdentifier(command.provisional_operation_id, "provisional_operation_id"),
    request_id: requireIdentifier(command.request_id, "request_id"),
    session_id: requireIdentifier(command.session_id, "session_id"),
    player_id: requireIdentifier(command.player_id, "player_id"),
    type: command.kind,
    chips,
    created_at: requireIdentifier(command.created_at, "created_at"),
    sync_status: "pending",
    sequence: requireSequence(command.sequence),
  };
}

export function projectSessionCommand(snapshot, command) {
  const { session, kind, playerIndex, chips } = validateProjectionInput(snapshot, command);
  const operation = provisionalOperation(command, chips);
  const nextPlayers = snapshot.players.map((player, index) => {
    if (index !== playerIndex) return { ...player };
    const buyIn =
      (Number(player.buy_in) || 0) + (kind === "buy_in" ? chips : 0);
    const cashOut =
      (Number(player.cash_out) || 0) + (kind === "cash_out" ? chips : 0);
    const profitChips = cashOut - buyIn;
    return {
      ...player,
      buy_in: buyIn,
      cash_out: cashOut,
      profit_chips: profitChips,
      profit_money: projectedProfit(profitChips, session.chipRate),
      in_game: kind === "buy_in",
    };
  });

  const totalBuyIn =
    (Number(session.totalBuyIn) || 0) + (kind === "buy_in" ? chips : 0);
  const totalCashOut =
    (Number(session.totalCashOut) || 0) + (kind === "cash_out" ? chips : 0);

  return {
    ...snapshot,
    session: {
      ...session,
      totalBuyIn,
      totalCashOut,
      totalChips: totalBuyIn - totalCashOut,
    },
    players: nextPlayers,
    operations: [operation, ...snapshot.operations.map((item) => ({ ...item }))],
    cached_at: command.created_at,
    local_revision: (Number(snapshot.local_revision) || 0) + 1,
    last_server_refresh_status: "local",
  };
}

export function cancelPendingSessionCommandProjection(snapshot, command) {
  requireRecord(snapshot, "invalid_snapshot", "session snapshot is required");
  const operation = snapshot.operations.find(
    (item) =>
      item.id === command?.provisional_operation_id &&
      item.request_id === command?.request_id,
  );
  if (!operation || operation.sync_status !== "pending") {
    throw new LocalProjectionError("reverse_target_unavailable", "pending operation is missing");
  }
  if (operation.type !== command.kind) {
    throw new LocalProjectionError("reverse_target_unavailable", "command lineage is invalid");
  }
  return {
    ...inverseOperation(snapshot, operation, { removeTarget: true }),
    cached_at: requireIdentifier(command.cancelled_at, "cancelled_at"),
  };
}

export function projectReverseSessionCommand(snapshot, command) {
  requireRecord(snapshot, "invalid_snapshot", "session snapshot is required");
  if (snapshot.session?.status !== "active") {
    throw new LocalProjectionError("session_not_active", "session is not active");
  }
  const sessionId = requireIdentifier(command?.session_id, "session_id");
  if (snapshot.session_id !== sessionId || snapshot.session?.id !== sessionId) {
    throw new LocalProjectionError("session_mismatch", "command belongs to another session");
  }
  const targetOperationId = requireIdentifier(
    command?.target_operation_id,
    "target_operation_id",
  );
  const target = snapshot.operations.find((operation) => operation.id === targetOperationId);
  if (!target || target.sync_status === "pending") {
    throw new LocalProjectionError("reverse_target_unavailable", "confirmed target is missing");
  }
  const alreadyReversed = snapshot.operations.some(
    (operation) =>
      operation.type === "reversal" && operation.reference_id === targetOperationId,
  );
  if (alreadyReversed) {
    throw new LocalProjectionError("already_reversed", "operation is already reversed");
  }

  const reversed = inverseOperation(snapshot, target);
  const reversal = {
    id: requireIdentifier(command.provisional_operation_id, "provisional_operation_id"),
    request_id: requireIdentifier(command.request_id, "request_id"),
    session_id: sessionId,
    player_id: target.player_id,
    type: "reversal",
    chips: Number(target.chips),
    created_at: requireIdentifier(command.created_at, "created_at"),
    reference_id: targetOperationId,
    sync_status: "pending",
    sequence: requireSequence(command.sequence),
  };
  return {
    ...reversed,
    operations: [reversal, ...reversed.operations],
    cached_at: command.created_at,
  };
}

export function reapplyPendingSessionCommands(snapshot, commands) {
  if (!Array.isArray(commands) || commands.length === 0) return snapshot;
  const originalRevision = snapshot.local_revision;
  const serverCachedAt = snapshot.cached_at;
  const ordered = [...commands].sort((left, right) => left.sequence - right.sequence);
  const projected = ordered.reduce((current, command) => {
    const payload = requireRecord(
      command.payload,
      "invalid_command",
      "pending payload is required",
    );
    if (command.kind === "reverse_operation") {
      return projectReverseSessionCommand(current, {
        kind: command.kind,
        request_id: command.request_id,
        session_id: command.session_id,
        target_operation_id: payload.target_operation_id,
        sequence: command.sequence,
        created_at: command.created_at,
        provisional_operation_id: command.provisional_operation_id,
      });
    }
    return projectSessionCommand(current, {
      kind: command.kind,
      request_id: command.request_id,
      session_id: command.session_id,
      player_id: payload.player_id,
      chips: payload.chips,
      sequence: command.sequence,
      created_at: command.created_at,
      provisional_operation_id: command.provisional_operation_id,
    });
  }, snapshot);
  return {
    ...projected,
    cached_at: serverCachedAt,
    local_revision: originalRevision,
    last_server_refresh_status: "fresh_with_pending",
  };
}

export function createPendingCommand({
  kind,
  sessionId,
  playerId,
  chips,
  requestId,
  sequence,
  createdAt,
  provisionalOperationId,
  payload,
}) {
  if (!LOCAL_COMMAND_KINDS.has(kind)) {
    throw new LocalProjectionError("invalid_command", "unsupported command kind");
  }
  const normalizedChips = requireChips(chips);
  const normalizedRequestId = requireIdentifier(requestId, "request_id");
  const normalizedSessionId = requireIdentifier(sessionId, "session_id");
  const normalizedPlayerId = requireIdentifier(playerId, "player_id");
  const normalizedCreatedAt = requireIdentifier(createdAt, "created_at");
  const normalizedSequence = requireSequence(sequence);

  const projectionCommand = {
    kind,
    request_id: normalizedRequestId,
    session_id: normalizedSessionId,
    player_id: normalizedPlayerId,
    chips: normalizedChips,
    sequence: normalizedSequence,
    created_at: normalizedCreatedAt,
    provisional_operation_id: requireIdentifier(
      provisionalOperationId,
      "provisional_operation_id",
    ),
  };
  const outboxCommand = {
    request_id: normalizedRequestId,
    session_id: normalizedSessionId,
    sequence: normalizedSequence,
    kind,
    payload: { ...requireRecord(payload, "invalid_command", "payload is required") },
    created_at: normalizedCreatedAt,
    status: "pending",
    attempts: 0,
    last_attempt_at: null,
    next_attempt_at: null,
    last_error_kind: null,
    provisional_operation_id: projectionCommand.provisional_operation_id,
  };
  return { projectionCommand, outboxCommand };
}

export function createPendingReverseCommand({
  sessionId,
  targetOperationId,
  requestId,
  sequence,
  createdAt,
  provisionalOperationId,
  payload,
}) {
  const normalizedRequestId = requireIdentifier(requestId, "request_id");
  const normalizedSessionId = requireIdentifier(sessionId, "session_id");
  const normalizedTargetId = requireIdentifier(targetOperationId, "target_operation_id");
  const normalizedCreatedAt = requireIdentifier(createdAt, "created_at");
  const normalizedSequence = requireSequence(sequence);
  const projectionCommand = {
    kind: "reverse_operation",
    request_id: normalizedRequestId,
    session_id: normalizedSessionId,
    target_operation_id: normalizedTargetId,
    sequence: normalizedSequence,
    created_at: normalizedCreatedAt,
    provisional_operation_id: requireIdentifier(
      provisionalOperationId,
      "provisional_operation_id",
    ),
  };
  const outboxCommand = {
    request_id: normalizedRequestId,
    session_id: normalizedSessionId,
    sequence: normalizedSequence,
    kind: "reverse_operation",
    payload: { ...requireRecord(payload, "invalid_command", "payload is required") },
    created_at: normalizedCreatedAt,
    status: "pending",
    attempts: 0,
    last_attempt_at: null,
    next_attempt_at: null,
    last_error_kind: null,
    provisional_operation_id: projectionCommand.provisional_operation_id,
    target_lineage_id: normalizedTargetId,
  };
  return { projectionCommand, outboxCommand };
}

export function classifyReverseTarget(snapshot, commands, operationId) {
  if (!snapshot || !Array.isArray(snapshot.operations) || !Array.isArray(commands)) {
    return { kind: REVERSE_TARGET_KINDS.UNAVAILABLE };
  }
  const operation = snapshot.operations.find((item) => item.id === operationId);
  if (!operation || snapshot.session?.status !== "active") {
    return { kind: REVERSE_TARGET_KINDS.UNAVAILABLE };
  }
  if (
    operation.type === "reversal" ||
    snapshot.operations.some(
      (item) => item.type === "reversal" && item.reference_id === operationId,
    ) ||
    commands.some(
      (command) =>
        command.kind === "reverse_operation" &&
        command.payload?.target_operation_id === operationId,
    )
  ) {
    return { kind: REVERSE_TARGET_KINDS.ALREADY_REVERSED, operation };
  }

  const originalCommand = commands.find(
    (command) =>
      command.request_id === operation.request_id ||
      command.provisional_operation_id === operation.id,
  );
  if (originalCommand) {
    const neverSent =
      originalCommand.status === "pending" &&
      originalCommand.attempts === 0 &&
      originalCommand.last_attempt_at === null;
    const unknownOutcomeKinds = new Set([
      "offline",
      "timeout",
      "network",
      "retryable_http",
      "invalid_response",
    ]);
    const possiblySent =
      originalCommand.status === "sending" ||
      unknownOutcomeKinds.has(originalCommand.last_error_kind);
    return {
      kind: neverSent
        ? REVERSE_TARGET_KINDS.PENDING_UNSENT
        : possiblySent
          ? REVERSE_TARGET_KINDS.POSSIBLY_SENT
          : REVERSE_TARGET_KINDS.UNAVAILABLE,
      operation,
      command: originalCommand,
    };
  }
  if (operation.sync_status === "pending" || String(operation.id).startsWith("local-")) {
    return { kind: REVERSE_TARGET_KINDS.UNAVAILABLE, operation };
  }
  if (operation.type === "buy_in" || operation.type === "cash_out") {
    return { kind: REVERSE_TARGET_KINDS.SERVER_CONFIRMED, operation };
  }
  return { kind: REVERSE_TARGET_KINDS.UNAVAILABLE, operation };
}

export function createSingleFlightCommitter(commit) {
  const inFlight = new Map();
  return function commitOnce(key, input) {
    if (inFlight.has(key)) return inFlight.get(key);
    const pending = Promise.resolve()
      .then(() => commit(input))
      .finally(() => inFlight.delete(key));
    inFlight.set(key, pending);
    return pending;
  };
}

export async function commitProjectedSessionCommand({
  snapshot,
  projectionCommand,
  outboxCommand,
  persist,
  onCommitted,
  requestReplay,
}) {
  const nextSnapshot = projectSessionCommand(snapshot, projectionCommand);
  await persist(nextSnapshot, outboxCommand);
  onCommitted?.(nextSnapshot, outboxCommand);
  requestReplay?.(outboxCommand);
  return { snapshot: nextSnapshot, command: outboxCommand };
}
