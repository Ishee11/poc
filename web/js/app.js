import { createPlayer, startSession } from "./api.js";
import { loadSessions } from "./ui/lobby.js";
import { loadPlayersOverview } from "./ui/player.js";
import { initSessionActions, openSession } from "./ui/session.js";
import { describeError, showNotice } from "./utils.js";

document.addEventListener("DOMContentLoaded", async () => {
  initSessionActions();

  await Promise.all([loadSessions(), loadPlayersOverview()]);

  const openButton = document.getElementById("open-workspace-btn");
  const sessionSelect = document.getElementById("active-session-select");
  if (openButton && sessionSelect) {
    openButton.addEventListener("click", async () => {
      let sessionId = sessionSelect.value;
      if (!sessionId) {
        const first = document.querySelector(
          "#overview-sessions-wrap [data-open-session]",
        );
        sessionId = first?.getAttribute("data-open-session") || "";
      }

      if (!sessionId) {
        showNotice("No session available to open.", "info");
        return;
      }

      await openSession(sessionId);
    });
  }

  const startForm = document.getElementById("start-session-form");
  const chipInput = document.getElementById("start-chip-rate");
  if (startForm && chipInput) {
    startForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      const chipRate = Number(chipInput.value);
      if (!Number.isFinite(chipRate) || chipRate <= 0) {
        showNotice("Enter a valid chip rate.", "error");
        return;
      }

      const res = await startSession({ chipRate });
      if (!res.ok || !res.body?.session_id) {
        showNotice(describeError(res, "Failed to start session"), "error");
        return;
      }

      await Promise.all([loadSessions(), loadPlayersOverview()]);
      await openSession(res.body.session_id);
      showNotice("Session started.", "success");
    });
  }

  const createPlayerForm = document.getElementById("create-player-form");
  const createPlayerName = document.getElementById("create-player-name");
  if (createPlayerForm && createPlayerName) {
    createPlayerForm.addEventListener("submit", async (event) => {
      event.preventDefault();

      const name = createPlayerName.value.trim();
      if (!name) {
        showNotice("Enter player name.", "error");
        return;
      }

      const res = await createPlayer(name);
      if (!res.ok) {
        showNotice(describeError(res, "Failed to create player"), "error");
        return;
      }

      createPlayerName.value = "";
      await loadPlayersOverview();
      showNotice(`Player ${name} created.`, "success");
    });
  }
});
