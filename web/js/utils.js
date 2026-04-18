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

export function formatDate(v) {
  if (!v) return "-";
  const d = new Date(v);
  return isNaN(d.getTime()) ? String(v) : d.toLocaleString();
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

export function describeError(res, fallback = "Request failed") {
  if (!res) return fallback;

  const details = res.body?.details;
  if (
    res.body?.error === "session_not_balanced" &&
    typeof details?.remaining_chips !== "undefined"
  ) {
    return `Session is not balanced yet. Remaining chips on table: ${formatNumber(details.remaining_chips)}.`;
  }

  const errorCode = res.body?.error;
  if (errorCode) {
    return errorCode.replaceAll("_", " ");
  }

  return res.text || fallback;
}

export function setScreen(name) {
  document
    .getElementById("screen-lobby")
    ?.classList.toggle("active", name === "lobby");

  document
    .getElementById("screen-session")
    ?.classList.toggle("active", name === "session");

  document
    .getElementById("screen-player")
    ?.classList.toggle("active", name === "player");
}

export function openModal({
  title,
  description = "",
  fields = [],
  confirmText = "Confirm",
  cancelText = "Cancel",
}) {
  const root = document.getElementById("modal-root");
  if (!root) {
    return Promise.resolve(null);
  }

  root.hidden = false;

  const fieldMarkup = fields
    .map((field) => {
      if (field.type === "select") {
        const options = (field.options || [])
          .map(
            (option) =>
              `<option value="${escapeHtml(option.value)}"${option.value === field.value ? " selected" : ""}>${escapeHtml(option.label)}</option>`,
          )
          .join("");

        return `
          <label>
            ${escapeHtml(field.label)}
            <select name="${escapeHtml(field.name)}">${options}</select>
          </label>
        `;
      }

      return `
        <label>
          ${escapeHtml(field.label)}
          <input
            name="${escapeHtml(field.name)}"
            type="${escapeHtml(field.type || "text")}"
            value="${escapeHtml(field.value ?? "")}"
            ${field.min != null ? `min="${escapeHtml(field.min)}"` : ""}
            ${field.placeholder ? `placeholder="${escapeHtml(field.placeholder)}"` : ""}
          />
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
          <button type="button" class="secondary" id="modal-cancel-btn">${escapeHtml(cancelText)}</button>
          <button type="submit" id="modal-confirm-btn">${escapeHtml(confirmText)}</button>
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

    root.querySelector("#modal-cancel-btn")?.addEventListener("click", () => {
      close(null);
    });

    root.querySelector("#modal-form")?.addEventListener("submit", (event) => {
      event.preventDefault();
      const form = new FormData(event.currentTarget);
      const values = Object.fromEntries(form.entries());
      close(values);
    });
  });
}
