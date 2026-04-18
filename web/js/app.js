import { loadSessions } from "./ui/lobby.js";
import { openSession, initSessionActions } from "./ui/session.js";

document.addEventListener("DOMContentLoaded", async () => {
  initSessionActions();

  await loadSessions();

  const btn = document.getElementById("open-workspace-btn");
  const select = document.getElementById("active-session-select");

  if (!btn || !select) return;

  btn.addEventListener("click", async () => {
    let sessionId = select.value;

    if (!sessionId) {
      const first = document.querySelector("[data-open-session]");
      if (first) {
        sessionId = first.getAttribute("data-open-session");
      }
    }

    if (!sessionId) {
      alert("No session available");
      return;
    }

    await openSession(sessionId);
  });
});
