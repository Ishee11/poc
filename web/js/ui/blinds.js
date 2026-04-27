import { t } from "../i18n.js";
import { escapeHtml, formatNumber, pushRoute, replaceRoute, routeToBlinds, setScreen } from "../utils.js";

const STORAGE_KEY = "poker-blinds-clock-v1";

const defaultLevels = () => [
  { sb: 50, bb: 100, durationMinutes: 15 },
  { sb: 100, bb: 200, durationMinutes: 15 },
  { sb: 200, bb: 400, durationMinutes: 15 },
  { sb: 300, bb: 600, durationMinutes: 15 },
];

let blindsState = loadState();
let tickerId = null;

export function initBlindsClock() {
  if (!tickerId) {
    tickerId = window.setInterval(() => {
      syncRuntime();
      if (document.body.dataset.screen === "blinds") {
        renderBlindsClock();
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

  document.getElementById("blinds-reset-btn")?.addEventListener("click", () => {
    resetClock();
  });

  document.getElementById("blinds-add-level-btn")?.addEventListener("click", () => {
    addLevel();
  });
}

export function openBlindsClock({ replace = false } = {}) {
  syncRuntime();
  setScreen("blinds");
  renderBlindsClock();

  if (replace) {
    replaceRoute(routeToBlinds());
  } else {
    pushRoute(routeToBlinds());
  }
}

export function renderBlindsClock() {
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
  const levelsWrap = document.getElementById("blinds-levels-wrap");
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
    statusEl.textContent = statusLabel();
  }
  if (totalLevelsEl) {
    totalLevelsEl.textContent = formatNumber(blindsState.levels.length);
  }
  if (upcomingLevelsEl) {
    upcomingLevelsEl.textContent = formatNumber(Math.max(blindsState.levels.length - blindsState.currentLevelIndex - 1, 0));
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

  if (levelsWrap) {
    levelsWrap.innerHTML = blindsState.levels.length
      ? blindsState.levels
          .map((level, index) => renderLevelRow(level, index))
          .join("")
      : `<div class="empty-inline">${escapeHtml(t("blinds.noLevels"))}</div>`;

    bindLevelControls(levelsWrap);
  }
}

function renderLevelRow(level, index) {
  const locked = isCompletedLevel(index);
  const editable = !locked && !blindsState.running;

  return `
    <div class="blinds-level-row ${locked ? "is-locked" : ""}">
      <div class="blinds-level-number">${escapeHtml(
        t("blinds.levelValue", { level: index + 1 }),
      )}</div>
      <label>
        <span>${escapeHtml(t("blinds.smallBlind"))}</span>
        <input
          type="number"
          min="1"
          data-level-field="sb"
          data-level-index="${index}"
          value="${escapeHtml(level.sb)}"
          ${editable ? "" : "disabled"}
        />
      </label>
      <label>
        <span>${escapeHtml(t("blinds.bigBlind"))}</span>
        <input
          type="number"
          min="1"
          data-level-field="bb"
          data-level-index="${index}"
          value="${escapeHtml(level.bb)}"
          ${editable ? "" : "disabled"}
        />
      </label>
      <label>
        <span>${escapeHtml(t("blinds.duration"))}</span>
        <input
          type="number"
          min="1"
          data-level-field="durationMinutes"
          data-level-index="${index}"
          value="${escapeHtml(level.durationMinutes)}"
          ${editable ? "" : "disabled"}
        />
      </label>
      <div class="blinds-level-actions">
        <button
          type="button"
          class="secondary"
          data-delete-level="${index}"
          ${editable && blindsState.levels.length > 1 ? "" : "disabled"}
        >
          ${escapeHtml(t("blinds.deleteLevel"))}
        </button>
      </div>
    </div>
  `;
}

function bindLevelControls(root) {
  root.querySelectorAll("[data-level-field]").forEach((input) => {
    input.addEventListener("change", (event) => {
      const target = event.currentTarget;
      if (!(target instanceof HTMLInputElement)) return;

      const index = Number(target.dataset.levelIndex);
      const field = target.dataset.levelField;
      const value = Number(target.value);
      if (!Number.isInteger(index) || !field || !Number.isFinite(value) || value <= 0) {
        renderBlindsClock();
        return;
      }

      updateLevel(index, field, value);
    });
  });

  root.querySelectorAll("[data-delete-level]").forEach((button) => {
    button.addEventListener("click", (event) => {
      const target = event.currentTarget;
      if (!(target instanceof HTMLButtonElement)) return;

      const index = Number(target.dataset.deleteLevel);
      if (!Number.isInteger(index)) return;
      deleteLevel(index);
    });
  });
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
  renderBlindsClock();
}

function pauseClock() {
  if (!blindsState.running) return;

  syncRuntime();
  blindsState.running = false;
  blindsState.lastTickAt = null;
  saveState();
  renderBlindsClock();
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
  renderBlindsClock();
}

function resetClock() {
  blindsState = {
    ...blindsState,
    currentLevelIndex: 0,
    remainingMs: blindsState.levels.length ? levelDurationMs(blindsState.levels[0]) : 0,
    running: false,
    lastTickAt: null,
  };
  saveState();
  renderBlindsClock();
}

function addLevel() {
  if (blindsState.running) return;

  const last = blindsState.levels.at(-1) || { sb: 50, bb: 100, durationMinutes: 15 };
  blindsState.levels.push({
    sb: Number(last.sb) || 50,
    bb: Number(last.bb) || 100,
    durationMinutes: Number(last.durationMinutes) || 15,
  });
  saveState();
  renderBlindsClock();
}

function updateLevel(index, field, value) {
  const level = blindsState.levels[index];
  if (!level || blindsState.running || isCompletedLevel(index)) return;

  level[field] = value;
  if (field === "durationMinutes" && index === blindsState.currentLevelIndex && !blindsState.running) {
    blindsState.remainingMs = levelDurationMs(level);
  }
  saveState();
  renderBlindsClock();
}

function deleteLevel(index) {
  if (
    blindsState.running ||
    isCompletedLevel(index) ||
    blindsState.levels.length <= 1 ||
    index < blindsState.currentLevelIndex
  ) {
    return;
  }

  blindsState.levels.splice(index, 1);
  if (blindsState.currentLevelIndex >= blindsState.levels.length) {
    blindsState.currentLevelIndex = Math.max(blindsState.levels.length - 1, 0);
  }
  if (blindsState.currentLevelIndex === index) {
    blindsState.remainingMs = levelDurationMs(blindsState.levels[blindsState.currentLevelIndex]);
  }
  saveState();
  renderBlindsClock();
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
  const fallback = {
    levels: defaultLevels(),
    currentLevelIndex: 0,
    remainingMs: levelDurationMs(defaultLevels()[0]),
    running: false,
    lastTickAt: null,
  };

  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return fallback;

    const parsed = JSON.parse(raw);
    const levels = Array.isArray(parsed.levels)
      ? parsed.levels
          .map(normalizeLevel)
          .filter(Boolean)
      : [];
    if (!levels.length) return fallback;

    return {
      levels,
      currentLevelIndex: clampIndex(parsed.currentLevelIndex, levels.length),
      remainingMs: Number.isFinite(parsed.remainingMs) ? Math.max(parsed.remainingMs, 0) : levelDurationMs(levels[0]),
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
  if (!Number.isFinite(sb) || sb <= 0 || !Number.isFinite(bb) || bb <= 0 || !Number.isFinite(durationMinutes) || durationMinutes <= 0) {
    return null;
  }

  return {
    sb,
    bb,
    durationMinutes,
  };
}

function clampIndex(value, levelsLength) {
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
  if (blindsState.remainingMs <= 0 && blindsState.currentLevelIndex >= blindsState.levels.length - 1) {
    return t("blinds.statusFinished");
  }
  if (blindsState.currentLevelIndex === 0 && blindsState.remainingMs === levelDurationMs(blindsState.levels[0])) {
    return t("blinds.statusStopped");
  }
  return t("blinds.statusPaused");
}
