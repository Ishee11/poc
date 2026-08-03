const LOCAL_COMMAND_KINDS = new Set(["buy_in", "cash_out"]);
export const SESSION_REPLAY_REQUEST_EVENT = "poker-session-replay-request";

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
