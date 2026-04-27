import { t } from "../i18n.js";
import {
  escapeHtml,
  formatNumber,
  openModal,
  pushRoute,
  replaceRoute,
  routeToBlinds,
  setScreen,
} from "../utils.js";

const STORAGE_KEY = "poker-blinds-clock-v1";

const defaultLevels = () => [
  { sb: 50, bb: 100, durationMinutes: 15 },
  { sb: 100, bb: 200, durationMinutes: 15 },
  { sb: 200, bb: 400, durationMinutes: 15 },
  { sb: 300, bb: 600, durationMinutes: 15 },
];

let blindsState = loadState();
let tickerId = null;
let selectedLevelIndex = 0;

export function initBlindsClock() {
  if (!tickerId) {
    tickerId = window.setInterval(() => {
      syncRuntime();
      if (document.body.dataset.screen === "blinds") {
        renderBlindsClock({ updateEditor: false });
      }
    }, 1000);
  }

  document.getElementById("open-blinds-clock-btn")?.addEventListener("click", () => {
    openBlindsClock();
  });

  document.getElementById("blinds-back-home-btn")?.addEventListener("click", () => {
    setScreen("lobby");
    pushRoute("/");
  });

  document.getElementById("blinds-start-btn")?.addEventListener("click", () => {
    startClock();
  });

  document.getElementById("blinds-pause-btn")?.addEventListener("click", () => {
    pauseClock();
  });

  document.getElementById("blinds-resume-btn")?.addEventListener("click", () => {
    resumeClock();
  });

  document.getElementById("blinds-reset-btn")?.addEventListener("click", async () => {
    await confirmResetClock();
  });

  document.getElementById("blinds-add-level-btn")?.addEventListener("click", () => {
    addLevel();
  });

  document.getElementById("blinds-delete-level-btn")?.addEventListener("click", async () => {
    await confirmDeleteSelectedLevel();
  });

  document.getElementById("blinds-delete-all-levels-btn")?.addEventListener("click", async () => {
    await confirmDeleteAllLevels();
  });

  document.getElementById("blinds-save-level-btn")?.addEventListener("click", () => {
    saveSelectedLevel();
  });

  document.getElementById("blinds-level-select")?.addEventListener("change", (event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLSelectElement)) return;
    selectedLevelIndex = clampIndex(Number(target.value), blindsState.levels.length);
    renderBlindsClock({ updateEditor: true });
  });
}

export function openBlindsClock({ replace = false } = {}) {
  syncRuntime();
  selectedLevelIndex = clampIndex(selectedLevelIndex, blindsState.levels.length);
  setScreen("blinds");
  renderBlindsClock({ updateEditor: true });

  if (replace) {
    replaceRoute(routeToBlinds());
  } else {
    pushRoute(routeToBlinds());
  }
}

export function renderBlindsClock({ updateEditor = true } = {}) {
  syncRuntime();

  const current = currentLevel();
  const next = nextLevel();
  const currentLevelEl = document.getElementById("blinds-current-level");
  const currentBlindEl = document.getElementById("blinds-current-blinds");
  const nextBlindEl = document.getElementById("blinds-next-blinds");
  const timerEl = document.getElementById("blinds-timer");
  const statusEl = document.getElementById("blinds-status");
  const totalLevelsEl = document.getElementById("blinds-total-levels");
  const upcomingLevelsEl = document.getElementById("blinds-upcoming-levels");
  const startButton = document.getElementById("blinds-start-btn");
  const pauseButton = document.getElementById("blinds-pause-btn");
  const resumeButton = document.getElementById("blinds-resume-btn");

  if (currentLevelEl) {
    currentLevelEl.textContent =
      current != null ? t("blinds.levelValue", { level: blindsState.currentLevelIndex + 1 }) : "-";
  }
  if (currentBlindEl) {
    currentBlindEl.textContent = current
      ? `${formatNumber(current.sb)} / ${formatNumber(current.bb)}`
      : "-";
  }
  if (nextBlindEl) {
    nextBlindEl.textContent = next
      ? `${formatNumber(next.sb)} / ${formatNumber(next.bb)}`
      : t("blinds.noNextLevel");
  }
  if (timerEl) {
    timerEl.textContent = formatDuration(blindsState.remainingMs);
  }
  if (statusEl) {
    const label = statusLabel();
    statusEl.textContent = label;
    statusEl.classList.toggle("status-finished", label === t("blinds.statusFinished"));
    statusEl.classList.toggle("status-active", label !== t("blinds.statusFinished"));
  }
  if (totalLevelsEl) {
    totalLevelsEl.textContent = formatNumber(blindsState.levels.length);
  }
  if (upcomingLevelsEl) {
    upcomingLevelsEl.textContent = formatNumber(
      Math.max(blindsState.levels.length - blindsState.currentLevelIndex - 1, 0),
    );
  }

  if (startButton) {
    startButton.disabled = blindsState.running || blindsState.levels.length === 0;
  }
  if (pauseButton) {
    pauseButton.disabled = !blindsState.running;
  }
  if (resumeButton) {
    resumeButton.disabled =
      blindsState.running ||
      blindsState.levels.length === 0 ||
      (blindsState.currentLevelIndex >= blindsState.levels.length - 1 && blindsState.remainingMs <= 0);
  }

  if (updateEditor) {
    renderLevelEditor();
  }
}

function renderLevelEditor() {
  const select = document.getElementById("blinds-level-select");
  const lockHint = document.getElementById("blinds-level-lock-hint");
  const form = document.getElementById("blinds-level-form");
  const sbInput = document.getElementById("blinds-level-sb");
  const bbInput = document.getElementById("blinds-level-bb");
  const durationInput = document.getElementById("blinds-level-duration");
  const addButton = document.getElementById("blinds-add-level-btn");
  const saveButton = document.getElementById("blinds-save-level-btn");
  const deleteButton = document.getElementById("blinds-delete-level-btn");
  const deleteAllButton = document.getElementById("blinds-delete-all-levels-btn");

  if (!select || !form || !sbInput || !bbInput || !durationInput) return;

  selectedLevelIndex = clampIndex(selectedLevelIndex, blindsState.levels.length);
  const selectedLevel = blindsState.levels[selectedLevelIndex] || null;
  const locked = !selectedLevel || isCompletedLevel(selectedLevelIndex) || blindsState.running;

  select.innerHTML = blindsState.levels.length
    ? blindsState.levels
        .map((level, index) => {
          const suffix = index < blindsState.currentLevelIndex
            ? t("blinds.levelLocked")
            : index === blindsState.currentLevelIndex
              ? t("blinds.levelCurrent")
              : "";
          const label = `${t("blinds.levelValue", { level: index + 1 })} - ${formatNumber(level.sb)}/${formatNumber(level.bb)}${suffix ? ` · ${suffix}` : ""}`;
          return `<option value="${index}">${escapeHtml(label)}</option>`;
        })
        .join("")
    : `<option value="">${escapeHtml(t("blinds.noLevels"))}</option>`;

  if (blindsState.levels.length) {
    select.value = String(selectedLevelIndex);
  }
  select.disabled = blindsState.levels.length === 0;

  sbInput.value = selectedLevel ? String(selectedLevel.sb) : "";
  bbInput.value = selectedLevel ? String(selectedLevel.bb) : "";
  durationInput.value = selectedLevel ? String(selectedLevel.durationMinutes) : "";

  sbInput.disabled = locked;
  bbInput.disabled = locked;
  durationInput.disabled = locked;
  saveButton.disabled = locked;
  deleteButton.disabled = locked || blindsState.levels.length <= 1;
  deleteAllButton.disabled = blindsState.levels.length === 0 || blindsState.running;
  if (addButton) {
    addButton.disabled = blindsState.running;
  }

  if (lockHint) {
    if (!selectedLevel) {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.noLevels");
    } else if (blindsState.running) {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.lockedWhileRunning");
    } else if (isCompletedLevel(selectedLevelIndex)) {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.lockedCompletedLevel");
    } else {
      lockHint.hidden = true;
      lockHint.textContent = "";
    }
  }
}

function startClock() {
  if (!blindsState.levels.length) return;

  if (blindsState.remainingMs <= 0) {
    blindsState.currentLevelIndex = 0;
    blindsState.remainingMs = levelDurationMs(blindsState.levels[0]);
  }

  blindsState.running = true;
  blindsState.lastTickAt = Date.now();
  saveState();
  renderBlindsClock({ updateEditor: true });
}

function pauseClock() {
  if (!blindsState.running) return;

  syncRuntime();
  blindsState.running = false;
  blindsState.lastTickAt = null;
  saveState();
  renderBlindsClock({ updateEditor: true });
}

function resumeClock() {
  if (!blindsState.levels.length || blindsState.running) return;

  if (blindsState.remainingMs <= 0 && blindsState.currentLevelIndex < blindsState.levels.length) {
    const level = blindsState.levels[blindsState.currentLevelIndex];
    blindsState.remainingMs = level ? levelDurationMs(level) : 0;
  }

  blindsState.running = true;
  blindsState.lastTickAt = Date.now();
  saveState();
  renderBlindsClock({ updateEditor: true });
}

async function confirmResetClock() {
  const confirmed = await openModal({
    title: t("blinds.resetTitle"),
    description: t("blinds.resetDescription"),
    confirmText: t("blinds.reset"),
  });
  if (!confirmed) return;

  resetClock();
}

function resetClock() {
  blindsState = {
    ...blindsState,
    currentLevelIndex: 0,
    remainingMs: blindsState.levels.length ? levelDurationMs(blindsState.levels[0]) : 0,
    running: false,
    lastTickAt: null,
  };
  selectedLevelIndex = 0;
  saveState();
  renderBlindsClock({ updateEditor: true });
}

function addLevel() {
  if (blindsState.running) return;

  const last = blindsState.levels.at(-1) || { sb: 50, bb: 100, durationMinutes: 15 };
  blindsState.levels.push({
    sb: Number(last.sb) || 50,
    bb: Number(last.bb) || 100,
    durationMinutes: Number(last.durationMinutes) || 15,
  });
  selectedLevelIndex = blindsState.levels.length - 1;
  saveState();
  renderBlindsClock({ updateEditor: true });
}

function saveSelectedLevel() {
  const level = blindsState.levels[selectedLevelIndex];
  const sbInput = document.getElementById("blinds-level-sb");
  const bbInput = document.getElementById("blinds-level-bb");
  const durationInput = document.getElementById("blinds-level-duration");
  if (!level || !sbInput || !bbInput || !durationInput) return;
  if (blindsState.running || isCompletedLevel(selectedLevelIndex)) return;

  const sb = Number(sbInput.value);
  const bb = Number(bbInput.value);
  const durationMinutes = Number(durationInput.value);
  if (!Number.isFinite(sb) || sb <= 0 || !Number.isFinite(bb) || bb <= 0 || !Number.isFinite(durationMinutes) || durationMinutes <= 0) {
    renderBlindsClock({ updateEditor: true });
    return;
  }

  level.sb = sb;
  level.bb = bb;
  level.durationMinutes = durationMinutes;
  if (selectedLevelIndex === blindsState.currentLevelIndex && !blindsState.running) {
    blindsState.remainingMs = levelDurationMs(level);
  }
  saveState();
  renderBlindsClock({ updateEditor: true });
}

async function confirmDeleteSelectedLevel() {
  if (
    blindsState.running ||
    isCompletedLevel(selectedLevelIndex) ||
    blindsState.levels.length <= 1 ||
    !blindsState.levels[selectedLevelIndex]
  ) {
    return;
  }

  const confirmed = await openModal({
    title: t("blinds.deleteLevelTitle"),
    description: t("blinds.deleteLevelDescription", {
      level: selectedLevelIndex + 1,
    }),
    confirmText: t("blinds.deleteLevel"),
  });
  if (!confirmed) return;

  deleteSelectedLevel();
}

function deleteSelectedLevel() {
  blindsState.levels.splice(selectedLevelIndex, 1);
  if (blindsState.currentLevelIndex >= blindsState.levels.length) {
    blindsState.currentLevelIndex = Math.max(blindsState.levels.length - 1, 0);
  }
  selectedLevelIndex = clampIndex(selectedLevelIndex, blindsState.levels.length);
  if (blindsState.levels.length) {
    blindsState.remainingMs = levelDurationMs(blindsState.levels[blindsState.currentLevelIndex]);
  } else {
    blindsState.remainingMs = 0;
  }
  saveState();
  renderBlindsClock({ updateEditor: true });
}

async function confirmDeleteAllLevels() {
  if (!blindsState.levels.length) return;

  const confirmed = await openModal({
    title: t("blinds.deleteAllLevelsTitle"),
    description: t("blinds.deleteAllLevelsDescription"),
    confirmText: t("blinds.deleteAllLevels"),
  });
  if (!confirmed) return;

  blindsState.levels = [];
  blindsState.currentLevelIndex = 0;
  blindsState.remainingMs = 0;
  blindsState.running = false;
  blindsState.lastTickAt = null;
  selectedLevelIndex = 0;
  saveState();
  renderBlindsClock({ updateEditor: true });
}

function syncRuntime() {
  if (!blindsState.running || !blindsState.lastTickAt) return;

  const now = Date.now();
  const elapsed = Math.max(now - blindsState.lastTickAt, 0);
  if (elapsed <= 0) return;

  blindsState.lastTickAt = now;
  applyElapsed(elapsed);
  saveState();
}

function applyElapsed(elapsedMs) {
  let remainingElapsed = elapsedMs;

  while (remainingElapsed > 0 && blindsState.levels.length) {
    const level = blindsState.levels[blindsState.currentLevelIndex];
    if (!level) {
      blindsState.running = false;
      blindsState.remainingMs = 0;
      blindsState.lastTickAt = null;
      return;
    }

    if (blindsState.remainingMs <= 0) {
      blindsState.remainingMs = levelDurationMs(level);
    }

    if (remainingElapsed < blindsState.remainingMs) {
      blindsState.remainingMs -= remainingElapsed;
      return;
    }

    remainingElapsed -= blindsState.remainingMs;
    const nextIndex = blindsState.currentLevelIndex + 1;
    if (nextIndex >= blindsState.levels.length) {
      blindsState.currentLevelIndex = blindsState.levels.length - 1;
      blindsState.remainingMs = 0;
      blindsState.running = false;
      blindsState.lastTickAt = null;
      return;
    }

    blindsState.currentLevelIndex = nextIndex;
    blindsState.remainingMs = levelDurationMs(blindsState.levels[nextIndex]);
  }
}

function loadState() {
  const fallbackLevels = defaultLevels();
  const fallback = {
    levels: fallbackLevels,
    currentLevelIndex: 0,
    remainingMs: levelDurationMs(fallbackLevels[0]),
    running: false,
    lastTickAt: null,
  };

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;

    const parsed = JSON.parse(raw);
    const levels = Array.isArray(parsed.levels)
      ? parsed.levels.map(normalizeLevel).filter(Boolean)
      : [];
    if (!levels.length) return fallback;

    return {
      levels,
      currentLevelIndex: clampIndex(parsed.currentLevelIndex, levels.length),
      remainingMs: Number.isFinite(parsed.remainingMs)
        ? Math.max(parsed.remainingMs, 0)
        : levelDurationMs(levels[0]),
      running: Boolean(parsed.running),
      lastTickAt: Number.isFinite(parsed.lastTickAt) ? parsed.lastTickAt : null,
    };
  } catch {
    return fallback;
  }
}

function saveState() {
  try {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({
        levels: blindsState.levels,
        currentLevelIndex: blindsState.currentLevelIndex,
        remainingMs: blindsState.remainingMs,
        running: blindsState.running,
        lastTickAt: blindsState.lastTickAt,
      }),
    );
  } catch {}
}

function normalizeLevel(level) {
  const sb = Number(level?.sb);
  const bb = Number(level?.bb);
  const durationMinutes = Number(level?.durationMinutes);
  if (
    !Number.isFinite(sb) ||
    sb <= 0 ||
    !Number.isFinite(bb) ||
    bb <= 0 ||
    !Number.isFinite(durationMinutes) ||
    durationMinutes <= 0
  ) {
    return null;
  }

  return { sb, bb, durationMinutes };
}

function clampIndex(value, levelsLength) {
  if (levelsLength <= 0) return 0;
  const index = Number(value);
  if (!Number.isInteger(index) || index < 0) return 0;
  return Math.min(index, Math.max(levelsLength - 1, 0));
}

function currentLevel() {
  return blindsState.levels[blindsState.currentLevelIndex] || null;
}

function nextLevel() {
  return blindsState.levels[blindsState.currentLevelIndex + 1] || null;
}

function isCompletedLevel(index) {
  return index < blindsState.currentLevelIndex;
}

function levelDurationMs(level) {
  return Math.max(Number(level?.durationMinutes) || 0, 1) * 60 * 1000;
}

function formatDuration(ms) {
  const totalSeconds = Math.max(Math.ceil(ms / 1000), 0);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;
}

function statusLabel() {
  if (!blindsState.levels.length) return t("blinds.statusStopped");
  if (blindsState.running) return t("blinds.statusRunning");
  if (
    blindsState.remainingMs <= 0 &&
    blindsState.currentLevelIndex >= blindsState.levels.length - 1
  ) {
    return t("blinds.statusFinished");
  }
  if (
    blindsState.currentLevelIndex === 0 &&
    blindsState.remainingMs === levelDurationMs(blindsState.levels[0])
  ) {
    return t("blinds.statusStopped");
  }
  return t("blinds.statusPaused");
}
