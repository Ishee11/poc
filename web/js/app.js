import { loadSessions } from "./ui/lobby.js";

document.addEventListener("DOMContentLoaded", async () => {
  await loadSessions();
});
