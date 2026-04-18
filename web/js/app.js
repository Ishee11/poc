import { loadSessions } from "./ui/lobby.js";
import { openSession, initSessionActions } from "./ui/session.js";
import { startSession } from "./api.js";

document.addEventListener("DOMContentLoaded", async () => {
  initSessionActions();

  await loadSessions();

  // ===== open session =====
  const btn = document.getElementById("open-workspace-btn");
  const select = document.getElementById("active-session-select");

  if (btn && select) {
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
  }

  // ===== start session =====
  const startForm = document.getElementById("start-session-form");
  const chipInput = document.getElementById("start-chip-rate");

  if (startForm && chipInput) {
    startForm.addEventListener("submit", async () => {
      const chipRate = Number(chipInput.value);

      if (!Number.isFinite(chipRate) || chipRate <= 0) {
        alert("Invalid chip rate");
        return;
      }

      const res = await startSession({ chipRate });

      if (!res.ok) {
        console.error("startSession failed:", res.text);
        alert("Failed to start session");
        return;
      }

      await openSession(res.body.session_id);

      await loadSessions();
    });
  }
});
