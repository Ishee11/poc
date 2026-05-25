import { t } from "./i18n.js";

export function value(id) {
  const el = document.getElementById(id);
  return el ? el.value : "";
}

export function setValue(id, val) {
  const el = document.getElementById(id);
  if (el) el.value = val ?? "";
}

export function formatNumber(v) {
  const n = Number(v);
  return Number.isFinite(n) ? n.toLocaleString() : "-";
}

export function formatMoney(value, currency) {
  return `${formatNumber(value)} ${currencySymbol(currency)}`;
}

export function currencySymbol(currency) {
  switch (String(currency || "RUB").toUpperCase()) {
    case "USD":
      return "$";
    case "RUB":
    default:
      return "₽";
  }
}

export function formatDate(v, { seconds = false } = {}) {
  if (!v) return "-";
  const d = new Date(v);
  return isNaN(d.getTime())
    ? String(v)
    : d.toLocaleString([], {
        year: "numeric",
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        ...(seconds ? { second: "2-digit" } : {}),
      });
}

export function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function generateRequestId(prefix = "req") {
  return `${prefix}-${Date.now().toString(36)}-${Math.random()
    .toString(36)
    .slice(2, 8)}`;
}

export function showNotice(message, kind = "info") {
  const el = document.getElementById("page-notice");
  if (!el) return;

  if (!message) {
    el.hidden = true;
    el.textContent = "";
    el.className = "notice";
    return;
  }

  el.hidden = false;
  el.textContent = message;
  el.className = `notice ${kind}`;
}

export function describeError(res, fallback = t("error.fallback")) {
  if (!res) return fallback;

  const details = res.body?.details;
  const errorCode = res.body?.error;

  if (
    errorCode === "session_not_balanced" &&
    typeof details?.remaining_chips !== "undefined"
  ) {
    return t("error.sessionNotBalanced", {
      chips: formatNumber(details.remaining_chips),
    });
  }

  if (errorCode) {
    const message = t(`error.${errorCode}`);
    return message === `error.${errorCode}` ? errorCode.replaceAll("_", " ") : message;
  }

  return res.text || fallback;
}

export function setScreen(name) {
  showNotice("");
  if (name !== "session") {
    const appTitle = document.querySelector(".app-brand h1");
    if (appTitle && name !== "results") appTitle.textContent = t("app.title");
  }

  document
    .getElementById("screen-lobby")
    ?.classList.toggle("active", name === "lobby");

  document
    .getElementById("screen-session")
    ?.classList.toggle("active", name === "session");

  document
    .getElementById("screen-results")
    ?.classList.toggle("active", name === "results");

  document
    .getElementById("screen-player")
    ?.classList.toggle("active", name === "player");

  document
    .getElementById("screen-players-stats")
    ?.classList.toggle("active", name === "players-stats");

  document
    .getElementById("screen-account")
    ?.classList.toggle("active", name === "account");

  document
    .getElementById("screen-blinds")
    ?.classList.toggle("active", name === "blinds");

  if (name !== "blinds") {
    document.body.dataset.blindsMode = "default";
  } else if (!document.body.dataset.blindsMode) {
    document.body.dataset.blindsMode = "default";
  }

  document.body.dataset.screen = name;
  window.scrollTo({ top: 0, left: 0 });
}

export function setBlindsMode(mode = "default") {
  document.body.dataset.blindsMode = mode === "presentation" ? "presentation" : "default";
}

export function routeToSession(sessionId) {
  return `/session/${encodeURIComponent(sessionId)}`;
}

export function routeToSessionResults(sessionId) {
  return `/session/${encodeURIComponent(sessionId)}/results`;
}

export function routeToPlayer(playerId) {
  return `/player/${encodeURIComponent(playerId)}`;
}

export function routeToPlayersStats() {
  return "/players/stats";
}

export function routeToAccount() {
  return "/account";
}

export function routeToBlinds(mode = "default") {
  return mode === "presentation" ? "/blinds/presentation" : "/blinds";
}

export function pushRoute(path) {
  if (currentPath() !== path) {
    window.history.pushState({}, "", path);
  }
}

export function replaceRoute(path) {
  if (currentPath() !== path) {
    window.history.replaceState({}, "", path);
  }
}

export function routeToHome() {
  return "/";
}

function currentPath() {
  return `${window.location.pathname}${window.location.search}`;
}

export function openModal({
  title,
  description = "",
  fields = [],
  confirmText = t("common.confirm"),
  cancelText = t("common.cancel"),
  showCancel = true,
  confirmClass = "",
}) {
  const root = document.getElementById("modal-root");
  if (!root) {
    return Promise.resolve(null);
  }

  root.hidden = false;

  const fieldMarkup = fields
    .map((field) => {
      const showWhen = field.showWhen;
      const conditionalAttrs = showWhen
        ? ` data-modal-show-when-name="${escapeHtml(showWhen.name)}" data-modal-show-when-value="${escapeHtml(showWhen.value)}"`
        : "";
      if (field.type === "select") {
        const options = (field.options || [])
          .map(
            (option) =>
              `<option value="${escapeHtml(option.value)}"${option.value === field.value ? " selected" : ""}>${escapeHtml(option.label)}</option>`,
          )
          .join("");

        return `
          <label data-modal-field-wrap="${escapeHtml(field.name)}"${conditionalAttrs}>
            ${escapeHtml(field.label)}
            <select name="${escapeHtml(field.name)}">${options}</select>
          </label>
        `;
      }

      return `
        <label data-modal-field-wrap="${escapeHtml(field.name)}"${conditionalAttrs}>
          ${escapeHtml(field.label)}
          <span class="${field.step ? "modal-number-stepper" : ""}">
            ${
              field.step
                ? `<button type="button" class="secondary" data-modal-step="${escapeHtml(String(-Math.abs(Number(field.step))))}" data-modal-step-target="${escapeHtml(field.name)}">-${escapeHtml(String(Math.abs(Number(field.step))))}</button>`
                : ""
            }
            <input
              name="${escapeHtml(field.name)}"
              type="${escapeHtml(field.type || "text")}"
              value="${escapeHtml(field.value ?? "")}"
              ${field.min != null ? `min="${escapeHtml(field.min)}"` : ""}
              ${field.max != null ? `max="${escapeHtml(field.max)}"` : ""}
              ${field.placeholder ? `placeholder="${escapeHtml(field.placeholder)}"` : ""}
            />
            ${
              field.step
                ? `<button type="button" class="secondary" data-modal-step="${escapeHtml(String(Math.abs(Number(field.step))))}" data-modal-step-target="${escapeHtml(field.name)}">+${escapeHtml(String(Math.abs(Number(field.step))))}</button>`
                : ""
            }
          </span>
        </label>
      `;
    })
    .join("");

  root.innerHTML = `
    <div class="modal">
      <h3>${escapeHtml(title)}</h3>
      ${description ? `<p>${escapeHtml(description)}</p>` : ""}
      <form id="modal-form">
        ${fieldMarkup}
        <div class="modal-actions">
          ${
            showCancel
              ? `<button type="button" class="secondary" id="modal-cancel-btn">${escapeHtml(cancelText)}</button>`
              : ""
          }
          <button type="submit" id="modal-confirm-btn" class="${escapeHtml(confirmClass)}">${escapeHtml(confirmText)}</button>
        </div>
      </form>
    </div>
  `;

  return new Promise((resolve) => {
    const close = (result) => {
      root.hidden = true;
      root.innerHTML = "";
      resolve(result);
    };

    root.addEventListener(
      "click",
      (event) => {
        if (event.target === root) {
          close(null);
        }
      },
      { once: true },
    );

    const syncConditionalFields = () => {
      root.querySelectorAll("[data-modal-show-when-name]").forEach((wrap) => {
        const controlName = wrap.getAttribute("data-modal-show-when-name");
        const expectedValue = wrap.getAttribute("data-modal-show-when-value");
        const control = controlName
          ? Array.from(root.querySelectorAll("input, select, textarea")).find((item) => item.name === controlName)
          : null;
        const visible = control && control.value === expectedValue;
        wrap.hidden = !visible;
        wrap.querySelectorAll("input, select, textarea").forEach((input) => {
          input.disabled = !visible;
        });
      });
    };

    root.querySelector("#modal-form")?.addEventListener("input", syncConditionalFields);
    root.querySelector("#modal-form")?.addEventListener("change", syncConditionalFields);
    syncConditionalFields();

    root.querySelector("#modal-cancel-btn")?.addEventListener("click", () => {
      close(null);
    });

    root.querySelectorAll("[data-modal-step]").forEach((button) => {
      button.addEventListener("click", () => {
        const targetName = button.getAttribute("data-modal-step-target");
        const delta = Number(button.getAttribute("data-modal-step"));
        const input = targetName
          ? Array.from(root.querySelectorAll("input")).find((item) => item.name === targetName)
          : null;
        if (!input || !Number.isFinite(delta)) return;

        const current = Number(input.value) || 0;
        const min = input.getAttribute("min") === null ? 0 : Number(input.getAttribute("min"));
        const max = input.getAttribute("max") === null ? Infinity : Number(input.getAttribute("max"));
        const next = Math.max(min, Math.min(max, current + delta));
        input.value = String(next);
        input.dispatchEvent(new Event("input", { bubbles: true }));
      });
    });

    root.querySelector("#modal-form")?.addEventListener("submit", (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const values = Object.fromEntries(form.entries());
      close(values);
    });
  });
}
