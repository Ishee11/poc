import {
  getBlindClock,
  nextBlindClockLevel,
  pauseBlindClock,
  previousBlindClockLevel,
  resetBlindClock,
  resumeBlindClock,
  startBlindClock,
  updateBlindClockLevels,
} from "../api.js";
import { t } from "../i18n.js";
import {
  describeError,
  escapeHtml,
  formatNumber,
  openModal,
  pushRoute,
  replaceRoute,
  routeToBlinds,
  setScreen,
  showNotice,
} from "../utils.js";

let clockState = null;
let selectedLevelIndex = 0;
let runtimeStatus = "idle";
let runtimeLevelIndex = -1;
let runtimeRemainingSeconds = 0;
let runtimeTickAtMs = 0;
let tickerId = null;
let resyncId = null;
let lastAlertedLevel = null;
let lastCountdownAlertKey = "";
let audioContext = null;

export function initBlindsClock() {
  if (!tickerId) {
    tickerId = window.setInterval(() => {
      if (document.body.dataset.screen !== "blinds") return;
      tickRuntime();
      renderBlindsClock({ updateEditor: false });
    }, 1000);
  }

  if (!resyncId) {
    resyncId = window.setInterval(async () => {
      if (document.body.dataset.screen !== "blinds") return;
      await refreshBlindClock({ silent: true });
    }, 15000);
  }

  document.addEventListener("visibilitychange", async () => {
    if (document.visibilityState === "visible" && document.body.dataset.screen === "blinds") {
      await refreshBlindClock({ silent: true });
    }
  });

  document.getElementById("open-blinds-clock-btn")?.addEventListener("click", async () => {
    await openBlindsClock();
  });

  document.getElementById("blinds-back-home-btn")?.addEventListener("click", () => {
    setScreen("lobby");
    pushRoute("/");
  });

  document.getElementById("blinds-start-btn")?.addEventListener("click", async () => {
    unlockAudio();
    const res = await startBlindClock();
    handleMutationResult(res, {
      successMessage: t("notice.blindsStarted"),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-pause-btn")?.addEventListener("click", async () => {
    const res = await pauseBlindClock();
    handleMutationResult(res, {
      successMessage: t("notice.blindsPaused"),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-resume-btn")?.addEventListener("click", async () => {
    unlockAudio();
    const res = await resumeBlindClock();
    handleMutationResult(res, {
      successMessage: t("notice.blindsResumed"),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-reset-btn")?.addEventListener("click", async () => {
    const confirmed = await openModal({
      title: t("blinds.resetTitle"),
      description: t("blinds.resetDescription"),
      confirmText: t("blinds.reset"),
    });
    if (!confirmed) return;

    const res = await resetBlindClock();
    handleMutationResult(res, {
      successMessage: t("notice.blindsReset"),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-previous-level-btn")?.addEventListener("click", async () => {
    unlockAudio();
    const res = await previousBlindClockLevel();
    handleMutationResult(res, {
      successMessageFromBody: (body) =>
        t("notice.blindsLevelMoved", { level: Number(body?.current_level) || 0 }),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-next-level-btn")?.addEventListener("click", async () => {
    unlockAudio();
    const res = await nextBlindClockLevel();
    handleMutationResult(res, {
      successMessageFromBody: (body) =>
        t("notice.blindsLevelMoved", { level: Number(body?.current_level) || 0 }),
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-add-level-btn")?.addEventListener("click", async () => {
    if (!clockState) return;

    const levels = clockState.levels.map(cloneLevelInput);
    const last = levels.at(-1) || { small_blind: 10, big_blind: 20, duration_minutes: 30 };
    levels.push({
      small_blind: Number(last.small_blind) || 10,
      big_blind: Number(last.big_blind) || 20,
      duration_minutes: Number(last.duration_minutes) || 30,
    });

    const res = await updateBlindClockLevels(levels);
    handleMutationResult(res, {
      successMessageFromBody: (body) =>
        t("notice.blindsLevelAdded", { level: body?.levels?.length || levels.length }),
      errorMessage: t("error.invalid_blind_clock_level"),
      onSuccess: (body) => {
        selectedLevelIndex = Math.max((body?.levels?.length || levels.length) - 1, 0);
      },
    });
  });

  document.getElementById("blinds-delete-level-btn")?.addEventListener("click", async () => {
    if (!clockState?.levels?.[selectedLevelIndex]) return;

    const confirmed = await openModal({
      title: t("blinds.deleteLevelTitle"),
      description: t("blinds.deleteLevelDescription", {
        level: selectedLevelIndex + 1,
      }),
      confirmText: t("blinds.deleteLevel"),
    });
    if (!confirmed) return;

    const levels = clockState.levels
      .map(cloneLevelInput)
      .filter((_, index) => index !== selectedLevelIndex);

    const deletedLevelNumber = selectedLevelIndex + 1;
    const res = await updateBlindClockLevels(levels);
    handleMutationResult(res, {
      successMessage: t("notice.blindsLevelDeleted", { level: deletedLevelNumber }),
      errorMessage: t("error.invalid_blind_clock_level"),
      onSuccess: () => {
        selectedLevelIndex = Math.max(Math.min(selectedLevelIndex, levels.length - 1), 0);
      },
    });
  });

  document.getElementById("blinds-delete-all-levels-btn")?.addEventListener("click", async () => {
    const confirmed = await openModal({
      title: t("blinds.deleteAllLevelsTitle"),
      description: t("blinds.deleteAllLevelsDescription"),
      confirmText: t("blinds.deleteAllLevels"),
    });
    if (!confirmed) return;

    const res = await updateBlindClockLevels([]);
    handleMutationResult(res, {
      successMessage: t("notice.blindsAllLevelsDeleted"),
      errorMessage: t("error.blind_clock_has_no_levels"),
    });
  });

  document.getElementById("blinds-save-level-btn")?.addEventListener("click", async () => {
    if (!clockState?.levels?.[selectedLevelIndex]) return;

    const sbInput = document.getElementById("blinds-level-sb");
    const bbInput = document.getElementById("blinds-level-bb");
    const durationInput = document.getElementById("blinds-level-duration");
    if (!(sbInput instanceof HTMLInputElement) || !(bbInput instanceof HTMLInputElement) || !(durationInput instanceof HTMLInputElement)) {
      return;
    }

    const smallBlind = Number(sbInput.value);
    const bigBlind = Number(bbInput.value);
    const durationMinutes = Number(durationInput.value);
    if (
      !Number.isFinite(smallBlind) ||
      smallBlind <= 0 ||
      !Number.isFinite(bigBlind) ||
      bigBlind <= 0 ||
      !Number.isFinite(durationMinutes) ||
      durationMinutes <= 0
    ) {
      showNotice(t("error.invalid_blind_clock_level"), "error");
      return;
    }

    const levels = clockState.levels.map(cloneLevelInput);
    levels[selectedLevelIndex] = {
      small_blind: smallBlind,
      big_blind: bigBlind,
      duration_minutes: durationMinutes,
    };

    const res = await updateBlindClockLevels(levels);
    handleMutationResult(res, {
      successMessage: t("notice.blindsLevelSaved", { level: selectedLevelIndex + 1 }),
      errorMessage: t("error.invalid_blind_clock_level"),
    });
  });

  document.getElementById("blinds-level-select")?.addEventListener("change", (event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLSelectElement)) return;
    selectedLevelIndex = clampLevelIndex(Number(target.value), clockState?.levels?.length || 0);
    renderBlindsClock({ updateEditor: true });
  });
}

export async function openBlindsClock({ replace = false } = {}) {
  setScreen("blinds");
  await refreshBlindClock({ silent: false, announceLevelChange: false });

  if (replace) {
    replaceRoute(routeToBlinds());
  } else {
    pushRoute(routeToBlinds());
  }
}

export function renderBlindsClock({ updateEditor = true } = {}) {
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
  const prevButton = document.getElementById("blinds-previous-level-btn");
  const nextButton = document.getElementById("blinds-next-level-btn");

  const levels = Array.isArray(clockState?.levels) ? clockState.levels : [];
  const currentLevel = levels[runtimeLevelIndex] || null;
  const nextLevel = levels[runtimeLevelIndex + 1] || null;
  const status = runtimeStatus || clockState?.status || "idle";

  if (currentLevelEl) {
    currentLevelEl.textContent =
      runtimeLevelIndex >= 0 ? t("blinds.levelValue", { level: runtimeLevelIndex + 1 }) : "-";
  }
  if (currentBlindEl) {
    currentBlindEl.textContent = currentLevel
      ? `${formatNumber(currentLevel.small_blind)} / ${formatNumber(currentLevel.big_blind)}`
      : "-";
  }
  if (nextBlindEl) {
    nextBlindEl.textContent = nextLevel
      ? `${formatNumber(nextLevel.small_blind)} / ${formatNumber(nextLevel.big_blind)}`
      : t("blinds.noNextLevel");
  }
  if (timerEl) {
    timerEl.textContent = formatDuration(runtimeRemainingSeconds);
    timerEl.classList.toggle("is-warning", status === "running" && runtimeRemainingSeconds <= 60 && runtimeRemainingSeconds > 10);
    timerEl.classList.toggle("is-danger", status === "running" && runtimeRemainingSeconds <= 10);
  }
  if (statusEl) {
    const label =
      status === "idle"
        ? t("blinds.statusStopped")
        : t(`blinds.status${capitalize(status)}`);
    statusEl.textContent = label;
    statusEl.classList.toggle("status-finished", status === "finished");
    statusEl.classList.toggle("status-active", status !== "finished");
  }
  if (totalLevelsEl) {
    totalLevelsEl.textContent = formatNumber(levels.length);
  }
  if (upcomingLevelsEl) {
    upcomingLevelsEl.textContent = formatNumber(Math.max(levels.length - runtimeLevelIndex - 1, 0));
  }

  if (startButton) startButton.disabled = status === "running" || levels.length === 0 || status === "finished";
  if (pauseButton) pauseButton.disabled = status !== "running";
  if (resumeButton) resumeButton.disabled = status !== "paused";
  if (prevButton) prevButton.disabled = runtimeLevelIndex <= 0 || levels.length === 0;
  if (nextButton) nextButton.disabled = runtimeLevelIndex < 0 || runtimeLevelIndex >= levels.length - 1;

  if (updateEditor) {
    renderLevelEditor();
  }
}

function renderLevelEditor() {
  const select = document.getElementById("blinds-level-select");
  const lockHint = document.getElementById("blinds-level-lock-hint");
  const sbInput = document.getElementById("blinds-level-sb");
  const bbInput = document.getElementById("blinds-level-bb");
  const durationInput = document.getElementById("blinds-level-duration");
  const addButton = document.getElementById("blinds-add-level-btn");
  const saveButton = document.getElementById("blinds-save-level-btn");
  const deleteButton = document.getElementById("blinds-delete-level-btn");
  const deleteAllButton = document.getElementById("blinds-delete-all-levels-btn");

  if (
    !(select instanceof HTMLSelectElement) ||
    !(sbInput instanceof HTMLInputElement) ||
    !(bbInput instanceof HTMLInputElement) ||
    !(durationInput instanceof HTMLInputElement)
  ) {
    return;
  }

  const levels = Array.isArray(clockState?.levels) ? clockState.levels : [];
  selectedLevelIndex = clampLevelIndex(selectedLevelIndex, levels.length);
  const selectedLevel = levels[selectedLevelIndex] || null;
  const locked = isSelectedLevelLocked();

  select.innerHTML = levels.length
    ? levels
        .map((level, index) => {
          const suffix =
            index < runtimeLevelIndex
              ? t("blinds.levelLocked")
              : index === runtimeLevelIndex
                ? t("blinds.levelCurrent")
                : "";
          const label = `${t("blinds.levelValue", { level: index + 1 })} - ${formatNumber(level.small_blind)}/${formatNumber(level.big_blind)}${suffix ? ` · ${suffix}` : ""}`;
          return `<option value="${index}">${escapeHtml(label)}</option>`;
        })
        .join("")
    : `<option value="">${escapeHtml(t("blinds.noLevels"))}</option>`;

  if (levels.length) {
    select.value = String(selectedLevelIndex);
  }
  select.disabled = levels.length === 0;

  sbInput.value = selectedLevel ? String(selectedLevel.small_blind) : "";
  bbInput.value = selectedLevel ? String(selectedLevel.big_blind) : "";
  durationInput.value = selectedLevel ? String(selectedLevel.duration_minutes) : "";

  sbInput.disabled = locked;
  bbInput.disabled = locked;
  durationInput.disabled = locked;
  if (saveButton) saveButton.disabled = locked;
  if (deleteButton) deleteButton.disabled = locked || levels.length <= 1;
  if (deleteAllButton) deleteAllButton.disabled = levels.length === 0 || runtimeStatus !== "idle";
  if (addButton) addButton.disabled = runtimeStatus === "running";

  if (lockHint) {
    if (!selectedLevel) {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.noLevels");
    } else if (runtimeStatus === "running") {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.lockedWhileRunning");
    } else if (selectedLevelIndex <= runtimeLevelIndex && runtimeStatus !== "idle") {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.lockedCompletedLevel");
    } else {
      lockHint.hidden = true;
      lockHint.textContent = "";
    }
  }
}

async function refreshBlindClock({ silent = false, announceLevelChange = true } = {}) {
  const res = await getBlindClock();
  if (!res.ok || !res.body) {
    if (!silent) {
      showNotice(describeError(res, t("error.internal_error")), "error");
    }
    return;
  }

  applyClockState(res.body, { announceLevelChange });
  renderBlindsClock({ updateEditor: true });
}

function applyClockState(body, { announceLevelChange = true } = {}) {
  const previousLevel = runtimeLevelIndex;
  const previousStatus = runtimeStatus;

  clockState = body;
  runtimeStatus = body.status || "idle";
  runtimeLevelIndex = Number.isInteger(body.current_level_index) ? body.current_level_index : -1;
  runtimeRemainingSeconds = Math.max(Number(body.remaining_seconds) || 0, 0);
  runtimeTickAtMs = Date.now();
  selectedLevelIndex = clampLevelIndex(selectedLevelIndex, clockState?.levels?.length || 0);

  if (
    announceLevelChange &&
    previousLevel >= 0 &&
    runtimeLevelIndex >= 0 &&
    runtimeLevelIndex !== previousLevel
  ) {
    showNotice(t("notice.blindsLevelAutoChanged", { level: runtimeLevelIndex + 1 }), "success");
    playLevelChangeAlert();
  } else if (previousStatus === "paused" && runtimeStatus === "running") {
    unlockAudio();
  }

  lastAlertedLevel = runtimeLevelIndex;
  lastCountdownAlertKey = "";
}

function tickRuntime() {
  if (!clockState) return;

  const now = Date.now();
  const elapsedSeconds = Math.floor((now - runtimeTickAtMs) / 1000);
  if (elapsedSeconds <= 0) return;

  runtimeTickAtMs += elapsedSeconds * 1000;
  if (runtimeStatus !== "running") return;

  const levels = Array.isArray(clockState.levels) ? clockState.levels : [];
  let remaining = runtimeRemainingSeconds - elapsedSeconds;
  let levelIndex = runtimeLevelIndex;

  while (remaining <= 0 && levelIndex >= 0 && levelIndex < levels.length - 1) {
    levelIndex += 1;
    remaining += Number(levels[levelIndex].duration_minutes || 0) * 60;
    showNotice(t("notice.blindsLevelAutoChanged", { level: levelIndex + 1 }), "success");
    playLevelChangeAlert();
  }

  if (remaining <= 0 && levelIndex === levels.length - 1) {
    runtimeStatus = "finished";
    runtimeRemainingSeconds = 0;
    runtimeLevelIndex = levelIndex;
    playLevelChangeAlert();
    return;
  }

  runtimeRemainingSeconds = Math.max(remaining, 0);
  runtimeLevelIndex = levelIndex;
  maybePlayCountdownWarning();
}

function maybePlayCountdownWarning() {
  if (runtimeStatus !== "running" || runtimeLevelIndex < 0) return;

  const warningKey =
    runtimeRemainingSeconds <= 10
      ? `${runtimeLevelIndex}:danger:${runtimeRemainingSeconds}`
      : runtimeRemainingSeconds <= 60
        ? `${runtimeLevelIndex}:warning`
        : `${runtimeLevelIndex}:clear`;
  if (warningKey === lastCountdownAlertKey) return;
  lastCountdownAlertKey = warningKey;

  if (runtimeRemainingSeconds <= 10) {
    playWarningAlert();
    return;
  }

  if (runtimeRemainingSeconds === 60) {
    playWarningAlert();
  }
}

function isSelectedLevelLocked() {
  if (!clockState?.levels?.[selectedLevelIndex]) return true;
  if (runtimeStatus === "running") return true;
  if (runtimeStatus === "idle") return false;
  return selectedLevelIndex <= runtimeLevelIndex;
}

function cloneLevelInput(level) {
  return {
    small_blind: Number(level.small_blind) || 0,
    big_blind: Number(level.big_blind) || 0,
    duration_minutes: Number(level.duration_minutes) || 0,
  };
}

function clampLevelIndex(value, levelsLength) {
  if (levelsLength <= 0) return 0;
  const index = Number(value);
  if (!Number.isInteger(index) || index < 0) return 0;
  return Math.min(index, levelsLength - 1);
}

function formatDuration(totalSeconds) {
  const seconds = Math.max(Number(totalSeconds) || 0, 0);
  const minutes = Math.floor(seconds / 60);
  const restSeconds = seconds % 60;
  return `${String(minutes).padStart(2, "0")}:${String(restSeconds).padStart(2, "0")}`;
}

function handleMutationResult(
  res,
  { successMessage = "", successMessageFromBody = null, errorMessage = t("error.internal_error"), onSuccess = null } = {},
) {
  if (!res.ok || !res.body) {
    showNotice(describeError(res, errorMessage), "error");
    return;
  }

  if (typeof onSuccess === "function") {
    onSuccess(res.body);
  }
  applyClockState(res.body, { announceLevelChange: false });
  renderBlindsClock({ updateEditor: true });

  const message = typeof successMessageFromBody === "function"
    ? successMessageFromBody(res.body)
    : successMessage;
  if (message) {
    showNotice(message, "success");
  }
}

function capitalize(value) {
  const str = String(value || "");
  return str ? str.charAt(0).toUpperCase() + str.slice(1) : "";
}

function unlockAudio() {
  try {
    if (!audioContext) {
      const Ctx = window.AudioContext || window.webkitAudioContext;
      if (!Ctx) return;
      audioContext = new Ctx();
    }
    if (audioContext.state === "suspended") {
      audioContext.resume();
    }
  } catch {}
}

function playLevelChangeAlert() {
  playTone(880, 0.2, 0.16, "square");
  window.setTimeout(() => playTone(1174, 0.22, 0.18, "square"), 180);
}

function playWarningAlert() {
  playTone(1046, 0.14, 0.22, "square");
}

function playTone(frequency, durationSeconds, gainValue, type = "sine") {
  try {
    if (!audioContext || audioContext.state !== "running") return;
    const osc = audioContext.createOscillator();
    const gain = audioContext.createGain();
    osc.type = type;
    osc.frequency.value = frequency;
    gain.gain.setValueAtTime(0.0001, audioContext.currentTime);
    gain.gain.exponentialRampToValueAtTime(gainValue, audioContext.currentTime + 0.02);
    gain.gain.exponentialRampToValueAtTime(0.0001, audioContext.currentTime + durationSeconds);
    osc.connect(gain);
    gain.connect(audioContext.destination);
    osc.start();
    osc.stop(audioContext.currentTime + durationSeconds);
  } catch {}
}
