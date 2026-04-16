const API = window.location.origin;

export async function apiGet(path) {
  try {
    const res = await fetch(API + path);
    return await normalize(res);
  } catch (e) {
    return { ok: false, text: String(e) };
  }
}

export async function apiPost(path, body) {
  try {
    const res = await fetch(API + path, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
    return await normalize(res);
  } catch (e) {
    return { ok: false, text: String(e) };
  }
}

async function normalize(res) {
  const text = await res.text();

  let body = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {}

  return {
    ok: res.ok,
    status: res.status,
    body,
    text,
  };
}
