function countOrNull(value) {
  if (value === null || value === undefined || value === "") return null;
  const count = Number(value);
  return Number.isSafeInteger(count) && count >= 0 ? count : null;
}

export function playerSessionVisibility(detail) {
  const total = countOrNull(detail?.total_sessions_count);
  const visible = countOrNull(detail?.visible_sessions_count);

  if (total === null || visible === null || visible > total) {
    return { kind: "unavailable", total, visible };
  }
  if (total === 0) {
    return { kind: "empty", total, visible };
  }
  if (visible === 0) {
    return { kind: "hidden", total, visible };
  }
  if (visible < total) {
    return { kind: "partial", total, visible };
  }
  return { kind: "complete", total, visible };
}

export function sessionHistoryMessageKey(visibility) {
  if (visibility?.kind === "hidden") return "player.sessionsUnavailableEmpty";
  if (visibility?.kind === "unavailable") return "player.sessionHistoryUnavailable";
  return "common.noSessions";
}

export function financialVisibilityNoticeRequired(visibility) {
  return ["partial", "hidden", "unavailable"].includes(visibility?.kind);
}

export function playerSessionListCount(visibility, returnedCount) {
  if (visibility?.kind === "partial") {
    return { visible: visibility.visible, total: visibility.total };
  }
  return {
    visible: visibility?.visible ?? countOrNull(returnedCount) ?? 0,
    total: null,
  };
}
