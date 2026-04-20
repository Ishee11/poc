import {
  getAccount,
  getAccountAvailablePlayers,
  getCurrentUser,
  linkAccountPlayer,
  login,
  logout,
  register,
  startSession,
  unlinkAccountPlayer,
} from "./api.js";
import { initI18n, onLanguageChange, setLanguage, t } from "./i18n.js";
import { state } from "./state.js";
import {
  applyLatestSessionDefaults,
  firstActiveSessionId,
  loadSessions,
  renderSessions,
  syncSelect,
} from "./ui/lobby.js";
import {
  loadPlayerDetail,
  loadPlayersOverview,
  renderPlayerDetail,
  renderPlayers,
  renderPlayersOverview,
} from "./ui/player.js";
import {
  initSessionActions,
  openSession,
  renderActionPlayerOptions,
  renderOperations,
  renderSession,
} from "./ui/session.js";
import {
  describeError,
  escapeHtml,
  openModal,
  replaceRoute,
  routeToHome,
  setScreen,
  showNotice,
} from "./utils.js";

document.addEventListener("DOMContentLoaded", async () => {
  syncDebugMode();
  initI18n();
  initAuth();
  initAccountPanel();
  initSessionActions();
  initLanguageSelect();
  onLanguageChange(renderCurrentLanguage);

  await loadCurrentUser();
  await Promise.all([loadSessions(), loadPlayersOverview()]);
  await openInitialRoute();

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
  const chipInput = document.getElementById("start-chip-rate");
  const bigBlindInput = document.getElementById("start-big-blind");
  renderStartChipRateLabel();
  applyLatestSessionDefaults();
  if (startForm && chipInput && bigBlindInput) {
    startForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      const chipRate = Number(chipInput.value);
      const bigBlind = Number(bigBlindInput.value);
      const currency = defaultCurrency();
      if (!Number.isFinite(chipRate) || chipRate <= 0) {
        showNotice(t("notice.validChipRate"), "error");
        return;
      }
      if (!Number.isFinite(bigBlind) || bigBlind <= 0) {
        showNotice(t("notice.validBigBlind"), "error");
        return;
      }

      const confirmed = await openModal({
        title: t("modal.startTitle"),
        description: t("modal.startDescription", {
          chipRate,
          bigBlind,
          currencySymbol: currencySymbol(),
        }),
        confirmText: t("lobby.startSession"),
      });
      if (!confirmed) return;

      const res = await startSession({ chipRate, bigBlind, currency });
      if (!res.ok || !res.body?.session_id) {
        showNotice(describeError(res, t("error.failedStartSession")), "error");
        return;
      }

      await Promise.all([loadSessions(), loadPlayersOverview()]);
      applyLatestSessionDefaults({ force: true });
      await openSession(res.body.session_id);
      showNotice(t("notice.sessionStarted"), "success");
    });
  }

});

function initAuth() {
  const showLoginButton = document.getElementById("auth-show-login-btn");
  const form = document.getElementById("auth-login-form");
  const logoutButton = document.getElementById("auth-logout-btn");
  const registerButton = document.getElementById("auth-register-btn");

  if (showLoginButton) {
    showLoginButton.addEventListener("click", () => {
      state.authLoginOpen = true;
      renderAuthPanel();
      document.getElementById("auth-email")?.focus();
    });
  }

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();

      const email = document.getElementById("auth-email")?.value?.trim() || "";
      const password = document.getElementById("auth-password")?.value || "";
      if (!email || !password) {
        showNotice(t("notice.authCredentialsRequired"), "error");
        return;
      }

      const res = await login({ email, password });
      if (!res.ok || !res.body?.user) {
        showNotice(describeError(res, t("error.loginFailed")), "error");
        return;
      }

      state.authUser = res.body.user;
      state.authChecked = true;
      state.authLoginOpen = false;
      form.reset();
      renderAuthPanel();
      await loadAccount();
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
      showNotice(t("notice.logoutSuccess"), "success");
    });
  }

  if (registerButton) {
    registerButton.addEventListener("click", async () => {
      const form = document.getElementById("auth-login-form");
      const email = document.getElementById("auth-email")?.value?.trim() || "";
      const password = document.getElementById("auth-password")?.value || "";
      if (!email || !password) {
        showNotice(t("notice.authCredentialsRequired"), "error");
        return;
      }

      const res = await register({ email, password });
      if (!res.ok || !res.body?.user) {
        showNotice(describeError(res, t("error.registerFailed")), "error");
        return;
      }

      state.authUser = res.body.user;
      state.authChecked = true;
      state.authLoginOpen = false;
      form?.reset();
      renderAuthPanel();
      await loadAccount();
      showNotice(t("notice.registrationSuccess"), "success");
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }
}

async function loadCurrentUser() {
  const res = await getCurrentUser();
  state.authChecked = true;
  state.authUser = res.ok && res.body?.user ? res.body.user : null;
  renderAuthPanel();
  if (state.authUser) {
    await loadAccount();
  } else {
    clearAccount();
  }
}

function renderAuthPanel() {
  const form = document.getElementById("auth-login-form");
  const showLoginButton = document.getElementById("auth-show-login-btn");
  const registerRow = document.getElementById("auth-register-row");
  const userPanel = document.getElementById("auth-user-panel");
  const userName = document.getElementById("auth-user-name");

  if (!form || !showLoginButton || !registerRow || !userPanel || !userName) return;

  const user = state.authUser;
  showLoginButton.hidden = Boolean(user) || state.authLoginOpen;
  form.hidden = Boolean(user) || !state.authLoginOpen;
  registerRow.hidden = Boolean(user) || !state.authLoginOpen;
  userPanel.hidden = !user;

  if (user) {
    userName.textContent = `${user.email} · ${user.role}`;
  } else {
    userName.textContent = "-";
  }
}

function initAccountPanel() {
  const form = document.getElementById("account-link-form");
  const linked = document.getElementById("account-linked-players");

  if (form) {
    form.addEventListener("submit", async (event) => {
      event.preventDefault();

      const select = document.getElementById("account-player-select");
      const playerId = select?.value || "";
      if (!playerId) {
        showNotice(t("notice.selectAccountPlayer"), "error");
        return;
      }

      const res = await linkAccountPlayer(playerId);
      if (!res.ok) {
        showNotice(describeError(res, t("error.failedLinkPlayer")), "error");
        return;
      }

      showNotice(t("notice.accountPlayerLinked"), "success");
      await loadAccount();
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }

  if (linked) {
    linked.addEventListener("click", async (event) => {
      if (!(event.target instanceof Element)) return;

      const button = event.target.closest("[data-account-unlink-player]");
      if (!button) return;

      const playerId = button.dataset.accountUnlinkPlayer;
      const res = await unlinkAccountPlayer(playerId);
      if (!res.ok) {
        showNotice(describeError(res, t("error.failedUnlinkPlayer")), "error");
        return;
      }

      showNotice(t("notice.accountPlayerUnlinked"), "success");
      await loadAccount();
      await Promise.all([loadSessions(), loadPlayersOverview()]);
    });
  }
}

async function loadAccount() {
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

  state.accountPlayers = Array.isArray(accountRes.body?.players)
    ? accountRes.body.players
    : [];
  state.accountAvailablePlayers = availableRes.ok && Array.isArray(availableRes.body?.players)
    ? availableRes.body.players
    : [];
  renderAccountPanel();

  if (!availableRes.ok) {
    showNotice(describeError(availableRes, t("error.failedLoadAvailablePlayers")), "error");
  }
}

function clearAccount() {
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
  if (!panel || !linked || !select || !form) return;

  const user = state.authUser;
  panel.hidden = !user;
  if (!user) return;

  if (state.accountLoading) {
    linked.innerHTML = `<div class="empty-inline">${escapeHtml(t("account.loading"))}</div>`;
  } else if (state.accountPlayers.length === 0) {
    linked.innerHTML = `<div class="empty-inline">${escapeHtml(t("account.noLinkedPlayers"))}</div>`;
  } else {
    linked.innerHTML = state.accountPlayers
      .map(
        (player) => {
          const id = player.player_id || player.id || "";
          return `
          <div class="account-player-row">
            <span>${escapeHtml(player.name)}</span>
            <button type="button" class="secondary" data-account-unlink-player="${escapeHtml(id)}">${escapeHtml(t("account.unlinkPlayer"))}</button>
          </div>
        `;
        },
      )
      .join("");
  }

  select.innerHTML = `
    <option value="">${escapeHtml(t("account.selectPlayer"))}</option>
    ${state.accountAvailablePlayers
      .map((player) => {
        const id = player.player_id || player.id || "";
        return `<option value="${escapeHtml(id)}">${escapeHtml(player.name)}</option>`;
      })
      .join("")}
  `;
  form.hidden = state.accountLoading || state.accountAvailablePlayers.length === 0;
  if (!state.accountLoading && state.accountAvailablePlayers.length === 0) {
    linked.insertAdjacentHTML(
      "beforeend",
      `<div class="hint account-empty-hint">${escapeHtml(t("account.noAvailablePlayers"))}</div>`,
    );
  }
}

function initLanguageSelect() {
  const select = document.getElementById("language-select");
  if (!select) return;

  select.addEventListener("change", () => {
    setLanguage(select.value);
  });
}

function renderCurrentLanguage() {
  renderAuthPanel();
  renderAccountPanel();
  renderStartChipRateLabel();
  renderSessions();
  syncSelect();
  renderPlayersOverview();
  if (state.session) {
    renderSession();
    renderOperations();
    renderActionPlayerOptions();
  }
  if (state.players.length) {
    renderPlayers();
  }
  if (state.selectedPlayerDetail) {
    renderPlayerDetail();
  }
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

async function openInitialRoute({ fromHistory = false } = {}) {
  syncDebugMode();
  const [, section, rawId] = window.location.pathname.split("/");
  const id = rawId ? decodeURIComponent(rawId) : "";

  if (section === "session" && id) {
    await openSession(id, { replace: !fromHistory });
    return;
  }

  if (section === "player" && id) {
    await loadPlayerDetail(id, { replace: !fromHistory });
    return;
  }

  setScreen("lobby");
  if (!fromHistory) replaceRoute(routeToHome());
}

function syncDebugMode() {
  state.debugMode = new URLSearchParams(window.location.search).has("debug");
  document.body.classList.toggle("debug-mode", state.debugMode);
}
