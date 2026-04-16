import { loadSessions } from "./ui/lobby.js";
import { loadPlayers } from "./ui/player.js";
import { loadOperations } from "./ui/session.js";
import { openSession } from "./ui/session.js";

document.addEventListener("DOMContentLoaded", async () => {
  await loadSessions();

  const btn = document.getElementById("open-workspace-btn");
  const select = document.getElementById("active-session-select");

  if (!btn || !select) return;

  btn.onclick = async () => {
    let sessionId = select.value;

    // если не выбрано — берём первую сессию из state
    if (!sessionId) {
      const first = document.querySelector("[data-open-session]");
      if (first) {
        sessionId = first.getAttribute("data-open-session");
      }
    }

    if (!sessionId) return;

    await openSession(sessionId);
    await loadPlayers(sessionId);
    await loadOperations(sessionId);
  };
});
