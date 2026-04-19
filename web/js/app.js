import { createPlayer, startSession } from "./api.js";
import { initI18n, onLanguageChange, setLanguage, t } from "./i18n.js";
import { state } from "./state.js";
import {
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
  initSessionActions();
  initLanguageSelect();
  onLanguageChange(renderCurrentLanguage);

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
  if (startForm && chipInput && bigBlindInput) {
    startForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      const chipRate = Number(chipInput.value);
      const bigBlind = Number(bigBlindInput.value);
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
        description: t("modal.startDescription", { chipRate, bigBlind }),
        confirmText: t("lobby.startSession"),
      });
      if (!confirmed) return;

      const res = await startSession({ chipRate, bigBlind });
      if (!res.ok || !res.body?.session_id) {
        showNotice(describeError(res, t("error.failedStartSession")), "error");
        return;
      }

      await Promise.all([loadSessions(), loadPlayersOverview()]);
      await openSession(res.body.session_id);
      showNotice(t("notice.sessionStarted"), "success");
    });
  }

  const createPlayerForm = document.getElementById("create-player-form");
  const createPlayerName = document.getElementById("create-player-name");
  if (createPlayerForm && createPlayerName) {
    createPlayerForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      const name = createPlayerName.value.trim();
      if (!name) {
        showNotice(t("notice.enterPlayerName"), "error");
        return;
      }

      const confirmed = await openModal({
        title: t("modal.createPlayerTitle"),
        description: t("modal.createPlayerDescription", { name }),
        confirmText: t("lobby.createPlayer"),
      });
      if (!confirmed) return;

      const res = await createPlayer(name);
      if (!res.ok) {
        showNotice(describeError(res, t("error.failedCreatePlayer")), "error");
        return;
      }

      createPlayerName.value = "";
      await loadPlayersOverview();
      showNotice(t("notice.playerCreated", { name }), "success");
    });
  }
});

function initLanguageSelect() {
  const select = document.getElementById("language-select");
  if (!select) return;

  select.addEventListener("change", () => {
    setLanguage(select.value);
  });
}

function renderCurrentLanguage() {
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
