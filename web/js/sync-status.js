export const SYNC_UI_STATUSES = Object.freeze({
  ONLINE_FRESH: "online_fresh",
  ONLINE_SYNCING: "online_syncing",
  OFFLINE_CLEAN: "offline_clean",
  OFFLINE_PENDING: "offline_pending",
  RETRY_WAIT: "retry_wait",
  AUTHORIZATION_BLOCKED: "authorization_blocked",
  DOMAIN_BLOCKED: "domain_blocked",
  LOCAL_STORAGE_UNAVAILABLE: "local_storage_unavailable",
});

export function deriveSyncUIStatus({
  isOnline,
  localRuntimeStatus,
  replayStatus,
  pendingCount = 0,
  blockedCount = 0,
}) {
  const pending = Math.max(0, Number(pendingCount) || 0);
  const blocked = Math.max(0, Number(blockedCount) || 0);
  let kind;

  if (localRuntimeStatus === "unavailable") {
    kind = SYNC_UI_STATUSES.LOCAL_STORAGE_UNAVAILABLE;
  } else if (replayStatus === "authorization_blocked") {
    kind = SYNC_UI_STATUSES.AUTHORIZATION_BLOCKED;
  } else if (replayStatus === "domain_blocked" || blocked > 0) {
    kind = SYNC_UI_STATUSES.DOMAIN_BLOCKED;
  } else if (!isOnline) {
    kind = pending > 0
      ? SYNC_UI_STATUSES.OFFLINE_PENDING
      : SYNC_UI_STATUSES.OFFLINE_CLEAN;
  } else if (replayStatus === "waiting_for_retry") {
    kind = SYNC_UI_STATUSES.RETRY_WAIT;
  } else if (replayStatus === "syncing" || pending > 0) {
    kind = SYNC_UI_STATUSES.ONLINE_SYNCING;
  } else {
    kind = SYNC_UI_STATUSES.ONLINE_FRESH;
  }

  return Object.freeze({
    kind,
    pendingCount: pending,
    blockedCount: blocked,
    action: kind === SYNC_UI_STATUSES.RETRY_WAIT
      ? "retry"
      : kind === SYNC_UI_STATUSES.AUTHORIZATION_BLOCKED
        ? "authenticate"
        : null,
  });
}
