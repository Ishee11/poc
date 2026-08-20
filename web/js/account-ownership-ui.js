export function buildPlayerSelection(mode, playerId, name) {
	if (mode === "existing") {
		return playerId ? { mode: "existing", player_id: playerId } : null;
	}
	if (mode === "new") {
		const normalized = String(name || "").trim();
		return normalized ? { mode: "new", name: normalized } : null;
	}
	return null;
}

export function accountRequiresOnboarding(account) {
	return account?.onboarding_required === true && account?.player == null;
}

export function ownershipConflictRequiresRefresh(result) {
	return result?.status === 409 && [
		"player_already_linked",
		"account_already_linked",
	].includes(result?.body?.error);
}

export function playerContext(player) {
	const id = String(player?.player_id || player?.id || "");
	const shortId = id ? id.slice(0, 8) : "-";
	const sessions = Number(player?.sessions_count || 0);
	const lastPlayed = player?.last_played_at
		? new Date(player.last_played_at).toLocaleDateString()
		: "-";
	return `${player?.name || "-"} · ${sessions} · ${lastPlayed} · ${shortId}`;
}
