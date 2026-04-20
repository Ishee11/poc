import { getCurrentUser, login, logout, startSession } from "./api.js";
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
      renderAuthPanel();
      showNotice(t("notice.logoutSuccess"), "success");
    });
  }

  if (registerButton) {
    registerButton.addEventListener("click", () => {
      showNotice(t("notice.registrationPending"), "info");
    });
  }
}

async function loadCurrentUser() {
  const res = await getCurrentUser();
  state.authChecked = true;
  state.authUser = res.ok && res.body?.user ? res.body.user : null;
  renderAuthPanel();
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

function initLanguageSelect() {
  const select = document.getElementById("language-select");
  if (!select) return;

  select.addEventListener("change", () => {
    setLanguage(select.value);
  });
}

function renderCurrentLanguage() {
  renderAuthPanel();
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
