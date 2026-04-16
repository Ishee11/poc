import { loadSessions } from "./lobby.js";

document.addEventListener("DOMContentLoaded", async () => {
  await loadSessions();
});
