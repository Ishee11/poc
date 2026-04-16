import { loadSessions } from "./ui/lobby.js";
import { openSession } from "./ui/session.js";

document.addEventListener("DOMContentLoaded", async () => {
  await loadSessions();

  const btn = document.getElementById("open-workspace-btn");
  const select = document.getElementById("active-session-select");

  if (!btn || !select) return;

  btn.onclick = async () => {
    let sessionId = select.value;

    if (!sessionId) {
      const first = document.querySelector("[data-open-session]");
      if (first) {
        sessionId = first.getAttribute("data-open-session");
      }
    }

    // ❗ ВАЖНО: защита
    if (!sessionId) {
      alert("No session available");
      return;
    }

    await openSession(sessionId);
  };
});
