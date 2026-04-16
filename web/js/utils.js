export function value(id) {
  return document.getElementById(id).value;
}

export function setValue(id, val) {
  document.getElementById(id).value = val;
}

export function formatNumber(v) {
  return Number.isFinite(Number(v)) ? Number(v).toLocaleString() : "-";
}

export function formatDate(v) {
  if (!v) return "-";
  const d = new Date(v);
  return isNaN(d.getTime()) ? v : d.toLocaleString();
}

export function escapeHtml(str) {
  return String(str ?? "")
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

export function generateRequestId(prefix) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}
