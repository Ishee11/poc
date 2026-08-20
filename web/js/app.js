import {
	buyIn,
	cashOut,
	clearAdminAccountPlayer,
	getAccount,
	getAccountAvailablePlayers,
	getAdminAccounts,
	getAuthConfig,
  getCurrentUser,
  getUnlinkedPlayers,
	login,
	logout,
	register,
	replaceAdminAccountPlayer,
  reverseOperation,
	startSession,
	selectAccountPlayer,
} from "./api.js";
import {
	accountRequiresOnboarding,
	buildPlayerSelection,
	ownershipConflictRequiresRefresh,
	playerContext,
} from "./account-ownership-ui.js";
import { initI18n, onLanguageChange, setLanguage, t } from "./i18n.js";
import {
  blockOutboxCommand,
  claimNextReplayCommand,
  countPendingAndBlockedCommands,
  initializeLocalDatabase,
  readReplayDiagnostics,
  releaseAuthorizationBlockedCommands,
  retryOutboxCommand,
} from "./offline-db.js";
import { createOutboxReplay } from "./offline-sync.js";
import { SESSION_REPLAY_REQUEST_EVENT } from "./session-projection.js";
import { resolveLocalFirstSessionWrites } from "./rollout.js";
import { state } from "./state.js";
import {
  applyLatestSessionDefaults,
  firstActiveSessionId,
  initSessionsFilter,
  loadSessions,
  renderSessions,
  syncSelect,
} from "./ui/lobby.js";
import {
  loadPlayerDetail,
  applyPlayersOverviewFilter,
  loadPlayersOverview,
  openPlayersStats,
  renderPlayerDetail,
  renderPlayers,
  renderPlayersOverview,
  renderPlayersStatsPage,
  sortOverviewPlayers,
} from "./ui/player.js";
import {
  initSessionActions,
  openSession,
  openSessionResults,
  reconcileReplayedSessionCommand,
  renderActionPlayerOptions,
  renderExpenseForm,
  renderExpenses,
  renderOperations,
  renderSession,
  renderSessionSyncStatus,
  renderSettlement,
} from "./ui/session.js";
import { initBlindsClock, openBlindsClock, renderBlindsClock } from "./ui/blinds.js";
import {
  describeError,
  escapeHtml,
  openModal,
  pushRoute,
  replaceRoute,
  routeToAccount,
  routeToHome,
  setScreen,
  showNotice,
} from "./utils.js";

const sessionOutboxReplay = createOutboxReplay({
  store: {
    claimNextReplayCommand,
    retryOutboxCommand,
    blockOutboxCommand,
    countPendingAndBlockedCommands,
    readReplayDiagnostics,
  },
  send: async (command) => {
    const input = {
      sessionId: command.session_id,
      playerId: command.payload.player_id,
      chips: command.payload.chips,
      requestId: command.request_id,
    };
    const result = command.kind === "buy_in"
      ? await buyIn(input)
      : command.kind === "cash_out"
        ? await cashOut(input)
        : await reverseOperation({
            operationId: command.payload.target_operation_id,
            requestId: command.request_id,
          });
    console.info("session_replay_attempt", {
      request_id: command.request_id,
      command_kind: command.kind,
      session_id: command.session_id,
      attempt: command.attempts + 1,
      acknowledgement_result: result.ok ? "accepted" : result.errorKind,
      idempotent_replay: result.body?.idempotent_replay === true,
    });
    return result;
  },
  reconcile: reconcileReplayedSessionCommand,
  isActive: () => document.visibilityState !== "hidden",
  onStatus: ({
    status,
    pendingCount,
    blockedCount,
    lastSuccessfulReplayAt,
    errorDetails,
  }) => {
    state.sessionReplayStatus = status;
    state.sessionPendingCount = pendingCount;
    state.sessionBlockedCount = blockedCount;
    state.sessionLastSuccessfulReplayAt = lastSuccessfulReplayAt;
    state.sessionReplayError = errorDetails;
    renderSessionSyncStatus();
  },
});

let replayLifecycleInitialized = false;

document.addEventListener("DOMContentLoaded", async () => {
  initI18n();
  registerAppShellServiceWorker();
  initializeLocalRuntime();
  await loadAuthConfig();
  applyUiFeatureFlags();
  initPlayersSort();
  initPlayersOverviewFilters();
  initSessionsFilter();
  initAdminLoginFooter();
  if (state.authUiEnabled) {
    initAuth();
    initAccountPanel();
    initGuestPlayerSelect();
  }
  initSessionActions();
  initBlindsClock();
  initLanguageSelect();
  onLanguageChange(renderCurrentLanguage);

  if (state.authUiEnabled) {
    await loadCurrentUser();
    await loadGuestPlayers();
  } else {
    await loadCurrentAdminUser();
  }
  if (isSessionRoute()) {
    const routePromise = openInitialRoute();
    void Promise.all([loadSessions(), loadPlayersOverview()]).catch((error) => {
      console.error("Background lobby refresh failed:", error);
    });
    await routePromise;
  } else {
    await Promise.all([loadSessions(), loadPlayersOverview()]);
    await openInitialRoute();
  }

  window.addEventListener("popstate", () => {
    openInitialRoute({ fromHistory: true });
  });

  const openButton = document.getElementById("open-workspace-btn");
  const sessionSelect = document.getElementById("active-session-select");
  if (openButton && sessionSelect) {
    openButton.addEventListener("click", async () => {
      let sessionId = sessionSelect.value;
      if (!sessionId) {
        sessionId = firstActiveSessionId();
      }

      if (!sessionId) {
        showNotice(t("notice.noSession"), "info");
        return;
      }

      await openSession(sessionId);
    });
  }

  const startForm = document.getElementById("start-session-form");
  const startToggle = document.getElementById("start-session-toggle");
  renderStartChipRateLabel();
  applyLatestSessionDefaults();
  if (startForm && startToggle) {
    const handleStartSession = async (event) => {
      event.preventDefault();

      if (navigator.onLine === false) {
        showNotice(t("error.onlineRequired"), "error");
        return;
      }

      const currency = defaultCurrency();
      const values = await openModal({
        title: t("modal.startTitle"),
        confirmText: t("lobby.startSession"),
        confirmClass: "rebuy-action",
        fields: [
          {
            name: "chip_rate",
            label: t("lobby.chipRate", { currencySymbol: currencySymbol() }),
            type: "number",
            min: 1,
            value: defaultStartNumber("chip_rate", 1),
          },
          {
            name: "big_blind",
            label: t("lobby.bigBlind"),
            type: "number",
            min: 1,
            value: defaultStartNumber("big_blind", 1),
          },
        ],
      });
      if (!values) return;

      const chipRate = Number(values.chip_rate);
      const bigBlind = Number(values.big_blind);
      if (!Number.isFinite(chipRate) || chipRate <= 0) {
        showNotice(t("notice.validChipRate"), "error");
        return;
      }
      if (!Number.isFinite(bigBlind) || bigBlind <= 0) {
        showNotice(t("notice.validBigBlind"), "error");
        return;
      }

      const res = await startSession({ chipRate, bigBlind, currency });
      if (!res.ok || !res.body?.session_id) {
        showNotice(describeError(res, t("error.failedStartSession")), "error");
        return;
      }

      await Promise.all([loadSessions(), loadPlayersOverview()]);
      applyLatestSessionDefaults({ force: true });
      await openSession(res.body.session_id);
      showNotice(t("notice.sessionStarted"), "success");
    };

    startToggle.addEventListener("click", handleStartSession);
    startForm.addEventListener("submit", handleStartSession);
  }

});

async function loadAuthConfig() {
  const res = await getAuthConfig();
  if (res.ok && typeof res.body?.enabled === "boolean") {
    state.authUiEnabled = res.body.enabled;
  }
}

function initializeLocalRuntime() {
  state.localRuntimeStatus = "initializing";
  state.localRuntimeError = "";
  void initializeLocalDatabase()
    .then(() => {
      state.localRuntimeStatus = "available";
      renderSessionSyncStatus();
      initializeReplayLifecycle();
      if (state.authUser) {
        void resumeReplayAfterAuthentication();
      } else {
        void sessionOutboxReplay.requestReplay();
      }
    })
    .catch((error) => {
      state.localRuntimeStatus = "unavailable";
      state.localRuntimeError = error instanceof Error ? error.message : "IndexedDB unavailable";
      renderSessionSyncStatus();
      console.warn("Local session runtime is unavailable; continuing online-only", error);
    });
}

function initializeReplayLifecycle() {
  if (replayLifecycleInitialized) return;
  replayLifecycleInitialized = true;
  window.addEventListener(SESSION_REPLAY_REQUEST_EVENT, (event) => {
    void sessionOutboxReplay.requestReplay({
      allowEarlyRetry: event.detail?.allowEarlyRetry === true,
    });
  });
  window.addEventListener("online", () => {
    renderSessionSyncStatus();
    void sessionOutboxReplay.requestReplay({ allowEarlyRetry: true });
  });
  window.addEventListener("offline", renderSessionSyncStatus);
  document.addEventListener("visibilitychange", () => {
    if (document.visibilityState === "visible") {
      void sessionOutboxReplay.requestReplay({ allowEarlyRetry: true });
    }
  });
}

function registerAppShellServiceWorker() {
  if (!("serviceWorker" in navigator)) return;
  void navigator.serviceWorker.register("/sw.js").catch((error) => {
    console.warn("App shell service worker registration failed", error);
  });
}

async function resumeReplayAfterAuthentication() {
  if (state.localRuntimeStatus !== "available") return;
  await releaseAuthorizationBlockedCommands();
  await sessionOutboxReplay.requestReplay({ allowEarlyRetry: true });
}

function applyUiFeatureFlags() {
  const localFirst = resolveLocalFirstSessionWrites({
    storage: window.localStorage,
    documentRef: document,
  });
  state.localFirstSessionWritesEnabled = localFirst.enabled;
  state.localFirstSessionWritesFlagSource = localFirst.source;
  console.info("session_runtime_rollout", {
    local_first_writes_enabled: localFirst.enabled,
    flag_source: localFirst.source,
  });
  document.body.classList.toggle("auth-ui-disabled", !state.authUiEnabled);
}

function initAdminLoginFooter() {
  const form = document.getElementById("admin-login-form");
  const logoutButton = document.getElementById("admin-login-logout-btn");

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();

      const email = document.getElementById("admin-login-email")?.value?.trim() || "";
      const password = document.getElementById("admin-login-password")?.value || "";
      if (!email || !password) {
        showNotice(t("notice.authCredentialsRequired"), "error");
        return;
      }

      const res = await login({ email, password });
      if (!res.ok || !res.body?.user) {
        showNotice(describeError(res, t("error.loginFailed")), "error");
        return;
      }

      if (res.body.user.role !== "admin") {
        await logout();
        state.authUser = null;
        state.authChecked = true;
        syncAdminMode();
        renderAdminLoginFooter();
        showNotice(t("error.adminRequired"), "error");
        return;
      }

      state.authUser = res.body.user;
      state.authChecked = true;
      state.authLoginOpen = false;
      form.reset();
      renderAuthPanel();
      renderAdminLoginFooter();
      syncAdminMode();
      if (state.authUiEnabled) await loadAccount();
      await resumeReplayAfterAuthentication();
      showNotice(t("notice.loginSuccess"), "success");
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }

  if (logoutButton) {
    logoutButton.addEventListener("click", async () => {
      const res = await logout();
      if (!res.ok && res.status !== 401) {
        showNotice(describeError(res, t("error.logoutFailed")), "error");
        return;
      }

      state.authUser = null;
      state.authChecked = true;
      state.authLoginOpen = false;
      clearAccount();
      renderAuthPanel();
      renderAdminLoginFooter();
      syncAdminMode();
      showNotice(t("notice.logoutSuccess"), "success");
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }
}

async function loadCurrentAdminUser() {
  const res = await getCurrentUser();
  state.authChecked = true;
  state.authUser = res.ok && res.body?.user ? res.body.user : null;
  renderAdminLoginFooter();
  syncAdminMode();
  if (state.authUser) await resumeReplayAfterAuthentication();
}

function initAuth() {
  const showLoginButton = document.getElementById("auth-show-login-btn");
  const form = document.getElementById("auth-login-form");
  const accountButton = document.getElementById("auth-account-btn");
  const logoutButton = document.getElementById("auth-logout-btn");
  const registerButton = document.getElementById("auth-register-btn");
	const loginModeButton = document.getElementById("auth-login-mode-btn");
	const playerMode = document.getElementById("auth-player-mode");

  if (showLoginButton) {
    showLoginButton.addEventListener("click", () => {
      state.authLoginOpen = true;
      state.authMode = "login";
      renderAuthPanel();
      document.getElementById("auth-email")?.focus();
    });
  }

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();

      const email = document.getElementById("auth-email")?.value?.trim() || "";
      const password = document.getElementById("auth-password")?.value || "";
      const passwordConfirm =
        document.getElementById("auth-password-confirm")?.value || "";
		const registering = state.authMode === "register";
      if (!email || !password) {
        showNotice(t("notice.authCredentialsRequired"), "error");
        return;
      }
		if (registering && password !== passwordConfirm) {
        showNotice(t("notice.authPasswordsMismatch"), "error");
        return;
		}
		const player = registering
			? buildPlayerSelection(
				playerMode?.value,
				document.getElementById("auth-existing-player")?.value,
				document.getElementById("auth-new-player-name")?.value,
			)
			: null;
		if (registering && !player) {
			showNotice(t("notice.playerSelectionRequired"), "error");
			return;
		}
		if (registering && !window.confirm(t("ownership.confirm"))) return;

		const res = registering
			? await register({ email, password, player })
        : await login({ email, password });
      if (!res.ok || !res.body?.user) {
        showNotice(
          describeError(
            res,
            t(registering ? "error.registerFailed" : "error.loginFailed"),
          ),
          "error",
        );
        return;
      }

      state.authUser = res.body.user;
      state.authChecked = true;
      state.authLoginOpen = false;
      state.authMode = "login";
      form.reset();
      renderAuthPanel();
      renderAdminLoginFooter();
      syncAdminMode();
      await loadAccount();
		if (state.accountOnboardingRequired) await openAccount({ replace: true });
      await resumeReplayAfterAuthentication();
      showNotice(
        t(registering ? "notice.registrationSuccess" : "notice.loginSuccess"),
        "success",
      );
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }

  if (logoutButton) {
    logoutButton.addEventListener("click", async () => {
      const res = await logout();
      if (!res.ok && res.status !== 401) {
        showNotice(describeError(res, t("error.logoutFailed")), "error");
        return;
      }

      state.authUser = null;
      state.authChecked = true;
      state.authLoginOpen = false;
      state.authMode = "login";
      clearAccount();
      renderAuthPanel();
      renderAdminLoginFooter();
      syncAdminMode();
      await loadGuestPlayers();
      if (window.location.pathname === "/account") {
        setScreen("lobby");
        pushRoute(routeToHome());
      }
      showNotice(t("notice.logoutSuccess"), "success");
    });
  }

  if (accountButton) {
    accountButton.addEventListener("click", async () => {
      await openAccount();
    });
  }

  if (registerButton) {
    registerButton.addEventListener("click", () => {
		state.authMode = "register";
		void loadRegistrationPlayers();
		renderAuthPanel();
      document.getElementById("auth-password")?.focus();
    });
  }

  if (loginModeButton) {
    loginModeButton.addEventListener("click", () => {
      state.authMode = "login";
      renderAuthPanel();
      document.getElementById("auth-email")?.focus();
    });
	}

	if (playerMode) {
		playerMode.addEventListener("change", renderRegistrationOwnership);
	}
}

async function loadRegistrationPlayers() {
	state.registrationPlayersLoading = true;
	renderRegistrationOwnership();
	const res = await getUnlinkedPlayers({ limit: 200 });
	state.registrationPlayersLoading = false;
	state.registrationPlayers = res.ok && Array.isArray(res.body?.players) ? res.body.players : [];
	renderRegistrationOwnership();
	if (!res.ok) showNotice(describeError(res, t("error.failedLoadAvailablePlayers")), "error");
}

function renderRegistrationOwnership() {
	const container = document.getElementById("auth-player-selection");
	const mode = document.getElementById("auth-player-mode");
	const existingField = document.getElementById("auth-existing-player-field");
	const newField = document.getElementById("auth-new-player-field");
	const existing = document.getElementById("auth-existing-player");
	if (!container || !mode || !existingField || !newField || !existing) return;
	const registering = state.authMode === "register";
	container.hidden = !registering;
	existingField.hidden = mode.value !== "existing";
	newField.hidden = mode.value !== "new";
	existing.disabled = state.registrationPlayersLoading;
	existing.innerHTML = `
		<option value="">${escapeHtml(t(state.registrationPlayersLoading ? "ownership.loading" : "account.selectPlayer"))}</option>
		${state.registrationPlayers.map((player) => `<option value="${escapeHtml(playerId(player))}">${escapeHtml(playerContext(player))}</option>`).join("")}
	`;
}

async function loadCurrentUser() {
  const res = await getCurrentUser();
  state.authChecked = true;
  state.authUser = res.ok && res.body?.user ? res.body.user : null;
  renderAuthPanel();
  renderAdminLoginFooter();
  syncAdminMode();
  if (state.authUser) {
    await loadAccount();
	if (state.accountOnboardingRequired) {
		await openAccount({ replace: true });
	}
    await resumeReplayAfterAuthentication();
  } else {
    clearAccount();
  }
}

function renderAuthPanel() {
  if (!state.authUiEnabled) return;

  const form = document.getElementById("auth-login-form");
  const showLoginButton = document.getElementById("auth-show-login-btn");
  const registerRow = document.getElementById("auth-register-row");
  const registerButton = document.getElementById("auth-register-btn");
  const loginModeButton = document.getElementById("auth-login-mode-btn");
  const confirmPassword = document.getElementById("auth-password-confirm");
  const submitButton = document.getElementById("auth-submit-btn");
  const modeHint = document.getElementById("auth-mode-hint");
  const userPanel = document.getElementById("auth-user-panel");
  const userName = document.getElementById("auth-user-name");
  const guestPlayerLabel = document.getElementById("guest-player-label");
  const authPanel = document.querySelector(".auth-panel");

  if (!form || !showLoginButton || !registerRow || !userPanel || !userName) return;

  const user = state.authUser;
  authPanel?.classList.toggle("authenticated", Boolean(user));
  authPanel?.classList.toggle("login-open", !user && state.authLoginOpen);
  showLoginButton.hidden = Boolean(user) || state.authLoginOpen;
  form.hidden = Boolean(user) || !state.authLoginOpen;
  registerRow.hidden = Boolean(user) || !state.authLoginOpen;
  userPanel.hidden = !user;
  if (guestPlayerLabel) guestPlayerLabel.hidden = Boolean(user);

  const registering = state.authMode === "register";
  if (registerButton) registerButton.hidden = registering;
  if (loginModeButton) loginModeButton.hidden = !registering;
  if (modeHint) modeHint.hidden = registering;
  if (confirmPassword) confirmPassword.hidden = !registering;
	if (submitButton) {
		submitButton.textContent = t(registering ? "auth.register" : "auth.login");
	}
	renderRegistrationOwnership();

  if (user) {
    userName.textContent = `${user.email} · ${user.role}`;
  } else {
    userName.textContent = "-";
  }
}

function renderAdminLoginFooter() {
  const form = document.getElementById("admin-login-form");
  const userPanel = document.getElementById("admin-login-user-panel");
  const userName = document.getElementById("admin-login-user-name");
  const disclosure = document.getElementById("admin-login-disclosure");

  if (!form || !userPanel || !userName) return;

  const user = state.authUser?.role === "admin" ? state.authUser : null;
  form.hidden = Boolean(user);
  userPanel.hidden = !user;
  userName.textContent = user ? `${user.email} · ${user.role}` : "-";

  if (disclosure && user) {
    disclosure.open = false;
  }
}

function initGuestPlayerSelect() {
  state.guestPlayerId = loadGuestPlayerId();

  const select = document.getElementById("guest-player-select");
  if (!select) return;

  select.addEventListener("change", async () => {
    state.guestPlayerId = select.value;
    saveGuestPlayerId(state.guestPlayerId);
    await loadSessions();
  });
}

async function loadGuestPlayers() {
  if (!state.authUiEnabled) {
    state.guestPlayers = [];
    state.guestPlayerId = "";
    renderGuestPlayerSelect();
    return;
  }

  if (state.authUser) {
    state.guestPlayers = [];
    renderGuestPlayerSelect();
    return;
  }

  const res = await getUnlinkedPlayers({ limit: 200 });
  state.guestPlayers = res.ok && Array.isArray(res.body?.players) ? res.body.players : [];
  const selectedExists = state.guestPlayers.some((player) => {
    const id = playerId(player);
    return id === state.guestPlayerId;
  });
  if (state.guestPlayerId && !selectedExists) {
    state.guestPlayerId = "";
    saveGuestPlayerId("");
  }
  renderGuestPlayerSelect();
}

function renderGuestPlayerSelect() {
  const select = document.getElementById("guest-player-select");
  if (!select) return;

  select.innerHTML = `
    <option value="">${escapeHtml(t("guest.noPlayer"))}</option>
    ${state.guestPlayers
      .map((player) => {
        const id = playerId(player);
        return `<option value="${escapeHtml(id)}">${escapeHtml(player.name)}</option>`;
      })
      .join("")}
  `;
  select.value = state.guestPlayerId;
}

async function openAccount({ replace = false } = {}) {
  if (!state.authUiEnabled) {
    setScreen("lobby");
    if (replace) {
      replaceRoute(routeToHome());
    } else {
      pushRoute(routeToHome());
    }
    return;
  }

  setScreen("account");
  if (replace) {
    replaceRoute(routeToAccount());
  } else {
    pushRoute(routeToAccount());
  }

	if (state.authUser) {
		await loadAccount();
  } else {
    clearAccount();
    renderAccountPanel();
  }
}

function initAccountPanel() {
  if (!state.authUiEnabled) return;

  const form = document.getElementById("account-link-form");
	const mode = document.getElementById("account-player-mode");
	const adminSearch = document.getElementById("admin-account-search-form");
	const adminAccount = document.getElementById("admin-account-select");
	const adminReplace = document.getElementById("admin-account-replace");
	const adminClear = document.getElementById("admin-account-clear");

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();
		const selection = buildPlayerSelection(
			mode?.value,
			document.getElementById("account-player-select")?.value,
			document.getElementById("account-new-player-name")?.value,
		);
		if (!selection) {
			showNotice(t("notice.playerSelectionRequired"), "error");
        return;
      }
		if (!window.confirm(t("ownership.confirm"))) return;

		const res = await selectAccountPlayer(selection);
      if (!res.ok) {
			showNotice(describeError(res, t("error.failedClaimPlayer")), "error");
			if (ownershipConflictRequiresRefresh(res)) await loadAccount();
        return;
      }

		showNotice(t("notice.accountPlayerClaimed"), "success");
      await loadAccount();
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }

	mode?.addEventListener("change", renderAccountPanel);
	adminSearch?.addEventListener("submit", async () => {
		state.adminAccountsQuery = document.getElementById("admin-account-query")?.value?.trim() || "";
		await loadAdminAccounts();
	});
	adminAccount?.addEventListener("change", renderAdminOwnershipPanel);
	adminReplace?.addEventListener("click", replaceAdminOwnership);
	adminClear?.addEventListener("click", clearAdminOwnership);
}

async function loadAccount() {
  if (!state.authUiEnabled) {
    clearAccount();
    return;
  }

  if (!state.authUser) {
    clearAccount();
    return;
  }

  state.accountLoading = true;
  renderAccountPanel();

	const [accountRes, availableRes] = await Promise.all([
    getAccount(),
    getAccountAvailablePlayers({ limit: 200 }),
  ]);

  state.accountLoading = false;
  if (!accountRes.ok) {
    state.accountPlayers = [];
    state.accountAvailablePlayers = [];
    renderAccountPanel();
    showNotice(describeError(accountRes, t("error.failedLoadAccount")), "error");
    return;
  }

	state.account = accountRes.body;
	state.accountOnboardingRequired = accountRequiresOnboarding(accountRes.body);
	state.accountPlayers = accountRes.body?.player ? [accountRes.body.player] : [];
  state.accountAvailablePlayers = availableRes.ok && Array.isArray(availableRes.body?.players)
    ? availableRes.body.players
    : [];
	renderAccountPanel();
	if (state.adminMode) await loadAdminAccounts();

  if (!availableRes.ok) {
    showNotice(describeError(availableRes, t("error.failedLoadAvailablePlayers")), "error");
  }
}

function clearAccount() {
	state.account = null;
	state.accountOnboardingRequired = false;
  state.accountPlayers = [];
  state.accountAvailablePlayers = [];
  state.accountLoading = false;
  renderAccountPanel();
}

function renderAccountPanel() {
  const panel = document.getElementById("account-panel");
  const linked = document.getElementById("account-linked-players");
  const select = document.getElementById("account-player-select");
	const form = document.getElementById("account-link-form");
	const mode = document.getElementById("account-player-mode");
	const existingField = document.getElementById("account-existing-player-field");
	const newField = document.getElementById("account-new-player-field");
	const guidance = document.getElementById("account-admin-guidance");
	const adminPanel = document.getElementById("admin-account-ownership");
	if (!panel || !linked || !select || !form || !mode || !existingField || !newField) return;

  if (!state.authUiEnabled) {
    panel.hidden = true;
    return;
  }

  const user = state.authUser;
  panel.hidden = false;
	if (!user) {
    linked.innerHTML = `<div class="empty-inline">${escapeHtml(t("account.loginRequired"))}</div>`;
		form.hidden = true;
		if (adminPanel) adminPanel.hidden = true;
		return;
	}

  if (state.accountLoading) {
    linked.innerHTML = `<div class="empty-inline">${escapeHtml(t("account.loading"))}</div>`;
	} else if (!state.account?.player) {
		linked.innerHTML = `<div class="empty-inline">${escapeHtml(t("account.onboardingRequired"))}</div>`;
  } else {
		const player = state.account.player;
		linked.innerHTML = `
          <div class="account-player-row">
			<strong>${escapeHtml(player.name)}</strong>
			<span class="hint">${escapeHtml(playerId(player).slice(0, 8))}</span>
          </div>
        `;
  }

  select.innerHTML = `
    <option value="">${escapeHtml(t("account.selectPlayer"))}</option>
    ${state.accountAvailablePlayers
      .map((player) => {
        const id = playerId(player);
		return `<option value="${escapeHtml(id)}">${escapeHtml(playerContext(player))}</option>`;
      })
      .join("")}
  `;
	existingField.hidden = mode.value !== "existing";
	newField.hidden = mode.value !== "new";
	form.hidden = state.accountLoading || Boolean(state.account?.player);
	if (guidance) guidance.hidden = !state.account?.player;
	if (!state.accountLoading && !state.account?.player && mode.value === "existing" && state.accountAvailablePlayers.length === 0) {
    linked.insertAdjacentHTML(
      "beforeend",
      `<div class="hint account-empty-hint">${escapeHtml(t("account.noAvailablePlayers"))}</div>`,
    );
	}
	if (adminPanel) adminPanel.hidden = !state.adminMode;
	renderAdminOwnershipPanel();
}

async function loadAdminAccounts() {
	if (!state.adminMode) return;
	state.adminAccountsLoading = true;
	renderAdminOwnershipPanel();
	const res = await getAdminAccounts({ query: state.adminAccountsQuery, limit: 100 });
	state.adminAccountsLoading = false;
	state.adminAccounts = res.ok && Array.isArray(res.body?.accounts) ? res.body.accounts : [];
	renderAdminOwnershipPanel();
	if (!res.ok) showNotice(describeError(res, t("error.failedLoadAdminAccounts")), "error");
}

function renderAdminOwnershipPanel() {
	const panel = document.getElementById("admin-account-ownership");
	const accountSelect = document.getElementById("admin-account-select");
	const playerSelect = document.getElementById("admin-account-player-select");
	const current = document.getElementById("admin-account-current");
	const empty = document.getElementById("admin-account-empty");
	if (!panel || !accountSelect || !playerSelect || !current || !empty) return;
	panel.hidden = !state.adminMode;
	if (!state.adminMode) return;
	const previous = accountSelect.value;
	accountSelect.disabled = state.adminAccountsLoading;
	accountSelect.innerHTML = state.adminAccounts.map((account) => `<option value="${escapeHtml(account.id)}">${escapeHtml(account.email)} · ${escapeHtml(account.player?.name || t("adminOwnership.unassigned"))}</option>`).join("");
	if (state.adminAccounts.some((account) => account.id === previous)) accountSelect.value = previous;
	const selected = state.adminAccounts.find((account) => account.id === accountSelect.value) || state.adminAccounts[0];
	current.textContent = selected?.player ? t("adminOwnership.current", { player: selected.player.name }) : t("adminOwnership.unassigned");
	playerSelect.innerHTML = `<option value="">${escapeHtml(t("account.selectPlayer"))}</option>${state.accountAvailablePlayers.map((player) => `<option value="${escapeHtml(playerId(player))}">${escapeHtml(playerContext(player))}</option>`).join("")}`;
	empty.hidden = state.adminAccountsLoading || state.adminAccounts.length > 0;
}

async function replaceAdminOwnership() {
	const userId = document.getElementById("admin-account-select")?.value || "";
	const playerIdValue = document.getElementById("admin-account-player-select")?.value || "";
	if (!userId || !playerIdValue) {
		showNotice(t("notice.adminOwnershipSelectionRequired"), "error");
		return;
	}
	if (!window.confirm(t("adminOwnership.confirmReplace"))) return;
	const res = await replaceAdminAccountPlayer(userId, playerIdValue);
	if (!res.ok) {
		showNotice(describeError(res, t("error.failedReplaceOwnership")), "error");
		if (ownershipConflictRequiresRefresh(res)) await Promise.all([loadAccount(), loadAdminAccounts()]);
		return;
	}
	showNotice(t("notice.adminOwnershipReplaced"), "success");
	await loadAccount();
}

async function clearAdminOwnership() {
	const userId = document.getElementById("admin-account-select")?.value || "";
	if (!userId || !window.confirm(t("adminOwnership.confirmClear"))) return;
	const res = await clearAdminAccountPlayer(userId);
	if (!res.ok) {
		showNotice(describeError(res, t("error.failedClearOwnership")), "error");
		return;
	}
	showNotice(t("notice.adminOwnershipCleared"), "success");
	await loadAccount();
}

function initLanguageSelect() {
  const select = document.getElementById("language-select");
  if (!select) return;

  select.addEventListener("change", () => {
    setLanguage(select.value);
  });
}

function initPlayersSort() {
  const select = document.getElementById("overview-players-sort");
  if (!select) return;

  select.value = state.overviewPlayersSort;
  select.closest("label")?.addEventListener("click", (event) => {
    event.stopPropagation();
  });
  select.addEventListener("click", (event) => {
    event.stopPropagation();
  });
  select.addEventListener("change", () => {
    state.overviewPlayersSort = select.value || "last_activity";
    sortOverviewPlayers();
    renderPlayersOverview();
  });
}

function initPlayersOverviewFilters() {
  const filter = document.getElementById("overview-players-filter-toggle");
  if (!filter) return;

  filter.addEventListener("click", async (event) => {
    event.preventDefault();
    event.stopPropagation();
    state.overviewPlayersShowAll = !state.overviewPlayersShowAll;
    applyPlayersOverviewFilter();
  });
}

function renderCurrentLanguage() {
  if (state.authUiEnabled) {
    renderAuthPanel();
    renderAccountPanel();
    renderGuestPlayerSelect();
  }
  renderStartChipRateLabel();
  renderSessions();
  syncSelect();
  renderPlayersOverview();
  if (state.session) {
    renderSession();
    renderOperations();
    renderActionPlayerOptions();
    renderExpenseForm();
    renderExpenses();
    renderSettlement();
  }
  if (state.players.length) {
    renderPlayers();
  }
  if (state.selectedPlayerDetail) {
    renderPlayerDetail();
  }
  if (document.body.dataset.screen === "players-stats") {
    renderPlayersStatsPage();
  }
  if (document.body.dataset.screen === "blinds") {
    renderBlindsClock();
  }
}

function isSessionRoute() {
  const [, section, rawId] = window.location.pathname.split("/");
  return section === "session" && Boolean(rawId);
}

function defaultCurrency() {
  return "RUB";
}

function currencySymbol() {
  return "₽";
}

function renderStartChipRateLabel() {
  const label = document.getElementById("start-chip-rate-label");
  if (!label) return;

  label.textContent = t("lobby.chipRate", {
    currencySymbol: currencySymbol(),
  });
}

function defaultStartNumber(field, fallback) {
  const latest = state.overviewSessions[0];
  const value = Number(latest?.[field]);
  return Number.isFinite(value) && value > 0 ? value : fallback;
}

async function openInitialRoute({ fromHistory = false } = {}) {
	if (state.authUser && state.accountOnboardingRequired) {
		await openAccount({ replace: true });
		return;
	}
  const [, section, rawId, subSection] = window.location.pathname.split("/");
  const id = rawId ? decodeURIComponent(rawId) : "";

  if (section === "session" && id && subSection === "results") {
    await openSessionResults(id, { replace: !fromHistory });
    return;
  }

  if (section === "session" && id) {
    await openSession(id, { replace: !fromHistory });
    return;
  }

  if (section === "player" && id) {
    await loadPlayerDetail(id, { replace: !fromHistory });
    return;
  }

  if (section === "players" && id === "stats") {
    await openPlayersStats({ replace: !fromHistory });
    return;
  }

  if (section === "account") {
    if (!state.authUiEnabled) {
      setScreen("lobby");
      if (!fromHistory) replaceRoute(routeToHome());
      return;
    }
    await openAccount({ replace: !fromHistory });
    return;
  }

  if (section === "blinds") {
    await openBlindsClock({
      replace: !fromHistory,
      mode: id === "presentation" ? "presentation" : "default",
    });
    return;
  }

  setScreen("lobby");
  if (!fromHistory) replaceRoute(routeToHome());
}

function syncAdminMode() {
  state.adminMode = state.authUser?.role === "admin";
  document.body.classList.toggle("admin-mode", state.adminMode);
  if (state.session) {
    renderSession();
    renderOperations();
    renderActionPlayerOptions();
    renderExpenseForm();
    renderExpenses();
    renderSettlement();
  }
}

function loadGuestPlayerId() {
  try {
    return localStorage.getItem("poker-guest-player-id") || "";
  } catch {
    return "";
  }
}

function saveGuestPlayerId(playerId) {
  try {
    if (playerId) {
      localStorage.setItem("poker-guest-player-id", playerId);
    } else {
      localStorage.removeItem("poker-guest-player-id");
    }
  } catch {}
}
