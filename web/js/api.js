const API = window.location.origin;

export async function apiGet(path) {
  try {
    const res = await fetch(API + path, {
      headers: {
        Accept: "application/json",
      },
    });
    return await normalize(res);
  } catch (e) {
    return { ok: false, text: String(e) };
  }
}

export async function apiPost(path, body) {
  try {
    const res = await fetch(API + path, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify(body),
    });
    return await normalize(res);
  } catch (e) {
    return { ok: false, text: String(e) };
  }
}

async function normalize(res) {
  let text = "";

  try {
    text = await res.text();
  } catch (e) {
    return { ok: false, status: 0, body: null, text: String(e) };
  }

  let body = null;

  if (text) {
    try {
      body = JSON.parse(text);
    } catch {
      // оставляем body = null, текст остаётся в text
    }
  }

  return {
    ok: res.ok,
    status: res.status,
    body,
    text,
  };
}
