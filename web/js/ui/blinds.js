import {
  getBlindClock,
  getBlindClockPushStatus,
  getPushConfig,
  nextBlindClockLevel,
  pauseBlindClock,
  previousBlindClockLevel,
  resetBlindClock,
  resetBlindClockToDefault,
  resumeBlindClock,
  sendBlindClockPushTest,
  startBlindClock,
  subscribeBlindClockPush,
  unsubscribeBlindClockPush,
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
  setBlindsMode,
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
let editorOpen = false;
let audioWarmupDone = false;
let heroEventHideId = null;
let heroEventCleanupId = null;
let pushConfig = null;
let pushBusy = false;
let pushSubscribed = false;
let pushSupported = false;
let pushSettings = {
  notifyWarning60: true,
  notifyWarning10: true,
};
const PUSH_SETTINGS_STORAGE_KEY = "blindsPushSettings";

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
      void unlockAudio();
      await refreshBlindClock({ silent: true });
    }
  });

  document.addEventListener("pointerdown", () => {
    if (document.body.dataset.screen === "blinds") {
      void unlockAudio();
    }
  });

  document.getElementById("open-blinds-clock-btn")?.addEventListener("click", async () => {
    await openBlindsClock();
  });

  document.getElementById("blinds-open-presentation-btn")?.addEventListener("click", async () => {
    await openBlindsClock({ mode: "presentation" });
  });

  document.getElementById("blinds-exit-presentation-btn")?.addEventListener("click", async () => {
    await openBlindsClock({ mode: "default" });
  });

  document.getElementById("blinds-push-toggle-btn")?.addEventListener("click", async () => {
    await togglePushSubscription();
  });

  document.getElementById("blinds-push-test-btn")?.addEventListener("click", async () => {
    await sendPushTest();
  });

  document.getElementById("blinds-push-warning-60")?.addEventListener("change", async (event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLInputElement)) return;
    await updatePushSettings({ notifyWarning60: target.checked });
  });

  document.getElementById("blinds-push-warning-10")?.addEventListener("change", async (event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLInputElement)) return;
    await updatePushSettings({ notifyWarning10: target.checked });
  });

  document.getElementById("blinds-back-home-btn")?.addEventListener("click", () => {
    setScreen("lobby");
    pushRoute("/");
  });

  document.getElementById("blinds-toggle-editor-btn")?.addEventListener("click", () => {
    editorOpen = !editorOpen;
    renderBlindsClock({ updateEditor: true });
  });

  document.getElementById("blinds-toggle-btn")?.addEventListener("click", async () => {
    const action = currentToggleAction();
    if (!action) return;

    if (action === "start" || action === "resume") {
      await unlockAudio();
    }

    const res =
      action === "start"
        ? await startBlindClock()
        : action === "pause"
          ? await pauseBlindClock()
          : await resumeBlindClock();

    handleMutationResult(res, {
      eventMessage:
        action === "start"
          ? t("blinds.eventStarted")
          : action === "pause"
            ? t("blinds.eventPaused")
            : t("blinds.eventResumed"),
      eventTone: action === "pause" ? "warning" : "success",
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-reset-btn")?.addEventListener("click", async () => {
    const confirmed = await openModal({
      title: t("blinds.resetTitle"),
      description: t("blinds.resetDescription"),
      confirmText: t("blinds.resetTimer"),
    });
    if (!confirmed) return;

    const res = await resetBlindClock();
    handleMutationResult(res, {
      eventMessage: t("blinds.eventReset"),
      eventTone: "warning",
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-reset-default-btn")?.addEventListener("click", async () => {
    const confirmed = await openModal({
      title: t("blinds.resetDefaultTitle"),
      description: t("blinds.resetDefaultDescription"),
      confirmText: t("blinds.resetDefault"),
    });
    if (!confirmed) return;

    const res = await resetBlindClockToDefault();
    handleMutationResult(res, {
      eventMessage: t("blinds.eventResetDefault"),
      eventTone: "warning",
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-previous-level-btn")?.addEventListener("click", async () => {
    await unlockAudio();
    const res = await previousBlindClockLevel();
    handleMutationResult(res, {
      eventMessage: t("blinds.eventLevelMoved"),
      eventTone: "warning",
      errorMessage: t("error.internal_error"),
    });
  });

  document.getElementById("blinds-next-level-btn")?.addEventListener("click", async () => {
    await unlockAudio();
    const res = await nextBlindClockLevel();
    handleMutationResult(res, {
      eventMessage: t("blinds.eventLevelMoved"),
      eventTone: "warning",
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
    const applyDurationInput = document.getElementById("blinds-apply-duration-next");
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
    if (applyDurationInput instanceof HTMLInputElement && applyDurationInput.checked) {
      for (let index = selectedLevelIndex + 1; index < levels.length; index += 1) {
        levels[index] = {
          ...levels[index],
          duration_minutes: durationMinutes,
        };
      }
    }

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

export async function openBlindsClock({ replace = false, mode = "default" } = {}) {
  setScreen("blinds");
  setBlindsMode(mode);
  await refreshBlindClock({ silent: false, announceLevelChange: false });
  await refreshPushState();

  if (replace) {
    replaceRoute(routeToBlinds(mode));
  } else {
    pushRoute(routeToBlinds(mode));
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
  const toggleButton = document.getElementById("blinds-toggle-btn");
  const toggleEditorButton = document.getElementById("blinds-toggle-editor-btn");
  const prevButton = document.getElementById("blinds-previous-level-btn");
  const nextButton = document.getElementById("blinds-next-level-btn");
  const pushButton = document.getElementById("blinds-push-toggle-btn");
  const pushTestButton = document.getElementById("blinds-push-test-btn");
  const pushWarning60 = document.getElementById("blinds-push-warning-60");
  const pushWarning10 = document.getElementById("blinds-push-warning-10");
  const pushSettingsWrap = document.getElementById("blinds-push-settings");

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

  if (toggleButton) {
    const action = currentToggleAction();
    toggleButton.disabled = !action;
    toggleButton.textContent =
      action === "pause"
        ? t("blinds.pause")
        : action === "resume"
          ? t("blinds.resume")
          : t("blinds.start");
    toggleButton.classList.toggle("secondary", action === "pause");
  }
  if (toggleEditorButton) {
    toggleEditorButton.textContent = editorOpen ? t("blinds.closeEditor") : t("blinds.editStructure");
  }
  if (prevButton) prevButton.disabled = runtimeLevelIndex <= 0 || levels.length === 0;
  if (nextButton) nextButton.disabled = runtimeLevelIndex < 0 || runtimeLevelIndex >= levels.length - 1;
  if (pushButton) {
    pushButton.hidden = !pushConfig?.enabled;
    pushButton.disabled = pushBusy || !pushConfig?.enabled;
    pushButton.textContent = !pushConfig?.enabled
      ? t("blinds.alertsUnsupported")
      : pushSubscribed
        ? t("blinds.disableAlerts")
        : t("blinds.enableAlerts");
  }
  if (pushTestButton) {
    pushTestButton.hidden = !pushConfig?.enabled;
    pushTestButton.disabled = pushBusy || !pushConfig?.enabled || !pushSubscribed;
  }
  if (pushSettingsWrap) {
    pushSettingsWrap.hidden = !pushConfig?.enabled;
  }
  if (pushWarning60 instanceof HTMLInputElement) {
    pushWarning60.checked = pushSettings.notifyWarning60;
    pushWarning60.disabled = pushBusy || !pushConfig?.enabled;
  }
  if (pushWarning10 instanceof HTMLInputElement) {
    pushWarning10.checked = pushSettings.notifyWarning10;
    pushWarning10.disabled = pushBusy || !pushConfig?.enabled;
  }

  if (updateEditor) {
    renderLevelEditor();
  }
}

function renderLevelEditor() {
  const structurePanel = document.getElementById("blinds-structure-panel");
  const shell = document.getElementById("blinds-structure-shell");
  const collapsedWrap = document.getElementById("blinds-structure-collapsed");
  const collapsedSelect = document.getElementById("blinds-level-select-collapsed");
  const collapsedSummary = document.getElementById("blinds-collapsed-summary");
  const select = document.getElementById("blinds-level-select");
  const lockHint = document.getElementById("blinds-level-lock-hint");
  const sbInput = document.getElementById("blinds-level-sb");
  const bbInput = document.getElementById("blinds-level-bb");
  const durationInput = document.getElementById("blinds-level-duration");
  const addButton = document.getElementById("blinds-add-level-btn");
  const saveButton = document.getElementById("blinds-save-level-btn");
  const deleteButton = document.getElementById("blinds-delete-level-btn");
  const deleteAllButton = document.getElementById("blinds-delete-all-levels-btn");
  const resetDefaultButton = document.getElementById("blinds-reset-default-btn");
  const applyDurationInput = document.getElementById("blinds-apply-duration-next");

  if (
    !(collapsedSelect instanceof HTMLSelectElement) ||
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
  const canEdit = editorOpen && runtimeStatus !== "finished";

  if (structurePanel) structurePanel.classList.toggle("is-editing", editorOpen);
  if (shell) shell.hidden = !editorOpen;
  if (collapsedWrap) collapsedWrap.hidden = editorOpen;

  const optionsMarkup = levels.length
    ? levels
        .map((level, index) => {
          const suffix =
            index < runtimeLevelIndex
              ? t("blinds.levelLocked")
              : index === runtimeLevelIndex
                ? t("blinds.levelCurrent")
                : "";
          const label = `${t("blinds.levelValue", { level: index + 1 })} - ${formatNumber(level.small_blind)}/${formatNumber(level.big_blind)} · ${formatNumber(level.duration_minutes)} ${t("blinds.minutesShort")}${suffix ? ` · ${suffix}` : ""}`;
          return `<option value="${index}">${escapeHtml(label)}</option>`;
        })
        .join("")
    : `<option value="">${escapeHtml(t("blinds.noLevels"))}</option>`;

  select.innerHTML = optionsMarkup;
  collapsedSelect.innerHTML = optionsMarkup;

  if (levels.length) {
    select.value = String(selectedLevelIndex);
    collapsedSelect.value = String(selectedLevelIndex);
  }
  select.disabled = levels.length === 0;
  collapsedSelect.disabled = levels.length === 0;

  collapsedSelect.onchange = (event) => {
    const target = event.currentTarget;
    if (!(target instanceof HTMLSelectElement)) return;
    selectedLevelIndex = clampLevelIndex(Number(target.value), levels.length);
    renderBlindsClock({ updateEditor: true });
  };

  sbInput.value = selectedLevel ? String(selectedLevel.small_blind) : "";
  bbInput.value = selectedLevel ? String(selectedLevel.big_blind) : "";
  durationInput.value = selectedLevel ? String(selectedLevel.duration_minutes) : "";

  sbInput.disabled = locked || !canEdit;
  bbInput.disabled = locked || !canEdit;
  durationInput.disabled = locked || !canEdit;
  if (applyDurationInput instanceof HTMLInputElement) {
    applyDurationInput.disabled = locked || !canEdit || selectedLevelIndex >= levels.length - 1;
  }
  if (saveButton) saveButton.disabled = locked || !canEdit;
  if (deleteButton) deleteButton.disabled = locked || levels.length <= 1 || !canEdit;
  if (deleteAllButton) deleteAllButton.disabled = levels.length === 0 || runtimeStatus !== "idle";
  if (resetDefaultButton) resetDefaultButton.disabled = false;
  if (addButton) {
    addButton.disabled = runtimeStatus === "finished";
    addButton.hidden = !editorOpen;
  }
  if (deleteAllButton) {
    deleteAllButton.hidden = !editorOpen;
  }

  if (collapsedSummary) {
    if (!selectedLevel) {
      collapsedSummary.textContent = t("blinds.noLevels");
    } else {
      collapsedSummary.textContent = `${t("blinds.levelValue", { level: selectedLevelIndex + 1 })} · ${formatNumber(selectedLevel.small_blind)}/${formatNumber(selectedLevel.big_blind)} · ${selectedLevel.duration_minutes} ${t("blinds.minutesShort")}`;
    }
  }

  if (lockHint) {
    if (!selectedLevel) {
      lockHint.hidden = false;
      lockHint.textContent = t("blinds.noLevels");
    } else if (locked && runtimeStatus === "running") {
      lockHint.hidden = false;
      lockHint.textContent =
        selectedLevelIndex === runtimeLevelIndex
          ? t("blinds.lockedCurrentLevelWhileRunning")
          : t("blinds.lockedCompletedLevel");
    } else if (locked && runtimeStatus !== "idle") {
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
    showHeroEvent({
      label: t("blinds.levelStarted"),
      tone: "success",
      levelIndex: runtimeLevelIndex,
    });
    playLevelChangeAlert();
  } else if (previousStatus === "paused" && runtimeStatus === "running") {
    void unlockAudio();
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
    showHeroEvent({
      label: t("blinds.levelStarted"),
      tone: "success",
      levelIndex,
    });
    playLevelChangeAlert();
  }

  if (remaining <= 0 && levelIndex === levels.length - 1) {
    runtimeStatus = "finished";
    runtimeRemainingSeconds = 0;
    runtimeLevelIndex = levelIndex;
    showHeroEvent({
      label: t("blinds.eventFinished"),
      tone: "warning",
      levelIndex,
    });
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
  if (runtimeStatus === "idle") return false;
  if (runtimeStatus === "running") return selectedLevelIndex <= runtimeLevelIndex;
  if (runtimeStatus === "paused") return selectedLevelIndex < runtimeLevelIndex;
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
  {
    successMessage = "",
    successMessageFromBody = null,
    eventMessage = "",
    eventMessageFromBody = null,
    eventTone = "success",
    errorMessage = t("error.internal_error"),
    onSuccess = null,
  } = {},
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

  const eventLabel = typeof eventMessageFromBody === "function"
    ? eventMessageFromBody(res.body)
    : eventMessage;
  if (eventLabel) {
    showHeroEvent({
      label: eventLabel,
      tone: eventTone,
      levelIndex: runtimeLevelIndex,
    });
    return;
  }

  const noticeMessage = typeof successMessageFromBody === "function"
    ? successMessageFromBody(res.body)
    : successMessage;
  if (noticeMessage) {
    showNotice(noticeMessage, "success");
  }
}

function currentToggleAction() {
  if (!clockState || !Array.isArray(clockState.levels) || clockState.levels.length === 0) {
    return null;
  }
  if (runtimeStatus === "running") return "pause";
  if (runtimeStatus === "paused") return "resume";
  if (runtimeStatus === "finished") return null;
  return "start";
}

async function refreshPushState() {
  pushSupported = supportsWebPush();
  pushSettings = loadStoredPushSettings();

  const configRes = await getPushConfig();
  pushConfig = configRes.ok && configRes.body ? configRes.body : { enabled: false };

  if (!pushSupported || !pushConfig.enabled) {
    pushSubscribed = false;
    renderBlindsClock({ updateEditor: false });
    return;
  }

  try {
    const registration = await navigator.serviceWorker.register("/sw.js");
    const subscription = await registration.pushManager.getSubscription();
    pushSubscribed = Boolean(subscription);
    if (subscription) {
      await loadPushSettings(subscription.endpoint);
    }
  } catch {
    pushSubscribed = false;
  }

  renderBlindsClock({ updateEditor: false });
}

async function togglePushSubscription() {
  if (pushBusy) return;

  if (!supportsWebPush()) {
    showNotice(t("notice.pushUnsupported"), "error");
    return;
  }
  if (isIOS() && !isStandaloneDisplayMode()) {
    showNotice(t("notice.pushRequiresHomeScreen"), "error");
    return;
  }

  if (!pushConfig) {
    const configRes = await getPushConfig();
    pushConfig = configRes.ok && configRes.body ? configRes.body : { enabled: false };
  }
  if (!pushConfig?.enabled || !pushConfig?.public_key) {
    showNotice(t("notice.pushUnavailable"), "error");
    return;
  }

  pushBusy = true;
  renderBlindsClock({ updateEditor: false });

  try {
    const registration = await navigator.serviceWorker.register("/sw.js");
    let subscription = await registration.pushManager.getSubscription();

    if (subscription) {
      await unsubscribeBlindClockPush(subscription.endpoint);
      await subscription.unsubscribe();
      pushSubscribed = false;
      storePushSettings(pushSettings);
      showNotice(t("notice.pushDisabled"), "success");
      return;
    }

    const permission = await Notification.requestPermission();
    if (permission !== "granted") {
      showNotice(t("notice.pushPermissionDenied"), "error");
      return;
    }

    subscription = await registration.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: base64URLToUint8Array(pushConfig.public_key),
    });

    const subscriptionJSON = subscription.toJSON();
    const res = await subscribeBlindClockPush(
      subscriptionJSON,
      navigator.userAgent || "",
      pushSettings,
    );
    if (!res.ok) {
      showNotice(describeError(res, t("error.internal_error")), "error");
      return;
    }

    pushSubscribed = true;
    await loadPushSettings(subscription.endpoint);
    showNotice(t("notice.pushEnabled"), "success");
  } catch (error) {
    showNotice(String(error || t("error.internal_error")), "error");
  } finally {
    pushBusy = false;
    renderBlindsClock({ updateEditor: false });
  }
}

async function loadPushSettings(endpoint) {
  const res = await getBlindClockPushStatus(endpoint);
  if (!res.ok || !res.body) {
    pushSettings = loadStoredPushSettings();
    return;
  }

  pushSettings = {
    notifyWarning60: res.body.notify_warning_60 !== false,
    notifyWarning10: res.body.notify_warning_10 !== false,
  };
  storePushSettings(pushSettings);
}

async function updatePushSettings(nextSettings) {
  if (pushBusy) {
    return;
  }

  const previousSettings = { ...pushSettings };
  pushSettings = {
    ...pushSettings,
    ...nextSettings,
  };
  storePushSettings(pushSettings);
  renderBlindsClock({ updateEditor: false });

  if (!pushSubscribed) {
    return;
  }

  try {
    const registration = await navigator.serviceWorker.register("/sw.js");
    const subscription = await registration.pushManager.getSubscription();
    if (!subscription) {
      pushSubscribed = false;
      pushSettings = loadStoredPushSettings();
      showNotice(t("notice.pushSubscriptionMissing"), "error");
      return;
    }

    pushBusy = true;
    renderBlindsClock({ updateEditor: false });

    const res = await subscribeBlindClockPush(
      subscription.toJSON(),
      navigator.userAgent || "",
      pushSettings,
    );
    if (!res.ok) {
      pushSettings = previousSettings;
      storePushSettings(pushSettings);
      showNotice(describeError(res, t("error.internal_error")), "error");
      return;
    }

    showNotice(t("notice.pushSettingsSaved"), "success");
  } catch (error) {
    pushSettings = previousSettings;
    storePushSettings(pushSettings);
    showNotice(String(error || t("error.internal_error")), "error");
  } finally {
    pushBusy = false;
    renderBlindsClock({ updateEditor: false });
  }
}

async function sendPushTest() {
  if (!pushConfig?.enabled || !pushSubscribed || pushBusy) {
    return;
  }

  pushBusy = true;
  renderBlindsClock({ updateEditor: false });

  try {
    const res = await sendBlindClockPushTest();
    if (!res.ok) {
      showNotice(describeError(res, t("error.internal_error")), "error");
      return;
    }

    const delivered = Number(res.body?.delivered) || 0;
    const failed = Number(res.body?.failed) || 0;
    if (failed > 0) {
      const firstError = Array.isArray(res.body?.errors) ? res.body.errors[0] : "";
      showNotice(
        firstError
          ? `${t("blinds.testAlertFailed")} ${firstError}`
          : t("blinds.testAlertFailed"),
        "error",
      );
      return;
    }

    showNotice(t("blinds.testAlertSent", { count: delivered }), "success");
  } finally {
    pushBusy = false;
    renderBlindsClock({ updateEditor: false });
  }
}

function loadStoredPushSettings() {
  try {
    const raw = window.localStorage.getItem(PUSH_SETTINGS_STORAGE_KEY);
    if (!raw) {
      return { notifyWarning60: true, notifyWarning10: true };
    }

    const parsed = JSON.parse(raw);
    return {
      notifyWarning60: parsed?.notifyWarning60 !== false,
      notifyWarning10: parsed?.notifyWarning10 !== false,
    };
  } catch {
    return { notifyWarning60: true, notifyWarning10: true };
  }
}

function storePushSettings(settings) {
  try {
    window.localStorage.setItem(
      PUSH_SETTINGS_STORAGE_KEY,
      JSON.stringify({
        notifyWarning60: settings.notifyWarning60 !== false,
        notifyWarning10: settings.notifyWarning10 !== false,
      }),
    );
  } catch {}
}

function capitalize(value) {
  const str = String(value || "");
  return str ? str.charAt(0).toUpperCase() + str.slice(1) : "";
}

async function unlockAudio() {
  try {
    if (!audioContext) {
      const Ctx = window.AudioContext || window.webkitAudioContext;
      if (!Ctx) return false;
      audioContext = new Ctx();
    }
    if (audioContext.state === "suspended" || audioContext.state === "interrupted") {
      await audioContext.resume();
    }
    if (!audioWarmupDone && audioContext.state === "running") {
      const osc = audioContext.createOscillator();
      const gain = audioContext.createGain();
      gain.gain.value = 0.00001;
      osc.connect(gain);
      gain.connect(audioContext.destination);
      osc.start();
      osc.stop(audioContext.currentTime + 0.01);
      audioWarmupDone = true;
    }
    return audioContext.state === "running";
  } catch {
    return false;
  }
}

function supportsWebPush() {
  return typeof window !== "undefined" &&
    "serviceWorker" in navigator &&
    "PushManager" in window &&
    "Notification" in window;
}

function isStandaloneDisplayMode() {
  return window.matchMedia?.("(display-mode: standalone)")?.matches || window.navigator.standalone === true;
}

function isIOS() {
  const ua = window.navigator.userAgent || "";
  return /iPad|iPhone|iPod/.test(ua) ||
    (navigator.platform === "MacIntel" && navigator.maxTouchPoints > 1);
}

function base64URLToUint8Array(value) {
  const padding = "=".repeat((4 - (value.length % 4)) % 4);
  const base64 = (value + padding).replaceAll("-", "+").replaceAll("_", "/");
  const raw = atob(base64);
  const out = new Uint8Array(raw.length);
  for (let idx = 0; idx < raw.length; idx += 1) {
    out[idx] = raw.charCodeAt(idx);
  }
  return out;
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

function showHeroEvent({ label, tone = "success", levelIndex = runtimeLevelIndex } = {}) {
  const flash = document.getElementById("blinds-level-flash");
  const flashLabel = document.getElementById("blinds-level-flash-label");
  const flashLevel = document.getElementById("blinds-level-flash-level");
  const flashBlinds = document.getElementById("blinds-level-flash-blinds");
  const hero = document.querySelector(".blinds-hero-panel");

  if (!flash || !flashLabel || !flashLevel || !flashBlinds || !hero) {
    return;
  }

  const level = Array.isArray(clockState?.levels) ? clockState.levels[levelIndex] : null;
  flashLabel.textContent = label || "";
  flashLevel.textContent =
    Number.isInteger(levelIndex) && levelIndex >= 0
      ? t("blinds.levelValue", { level: levelIndex + 1 })
      : runtimeStatus === "idle"
        ? t("blinds.statusStopped")
        : t(`blinds.status${capitalize(runtimeStatus || "idle")}`);
  flashBlinds.textContent = level
    ? `${formatNumber(level.small_blind)} / ${formatNumber(level.big_blind)}`
    : runtimeStatus === "finished"
      ? t("blinds.statusFinished")
      : "";

  if (heroEventHideId) window.clearTimeout(heroEventHideId);
  if (heroEventCleanupId) window.clearTimeout(heroEventCleanupId);

  hero.classList.remove("is-event-active", "is-event-warning");
  flash.classList.remove("is-visible", "is-warning");
  flash.hidden = false;

  void flash.offsetWidth;

  hero.classList.add(tone === "warning" ? "is-event-warning" : "is-event-active");
  flash.classList.add("is-visible");
  flash.classList.toggle("is-warning", tone === "warning");

  heroEventHideId = window.setTimeout(() => {
    flash.classList.remove("is-visible");
    hero.classList.remove("is-event-active", "is-event-warning");
    heroEventCleanupId = window.setTimeout(() => {
      flash.hidden = true;
    }, 260);
  }, 2200);
}
