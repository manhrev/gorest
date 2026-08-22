// Cross-origin by design (this app runs on :5173, the API on :8080) — the
// API's CORS middleware (pkg/middleware/middleware.go) already allows any
// origin + Authorization/Content-Type headers + preflight, no server change
// needed for that. The gorest server must be running for any of this to work.
export const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080";

// apiCall: fetch wrapper that returns both the request it made and the
// response it got, shaped for direct display (not just the parsed body) —
// the whole point of this test client is to show the actual HTTP traffic.
export async function apiCall(method, path, headers = {}, body) {
  const url = path.startsWith("http") ? path : API_BASE + path;
  const opts = { method, headers: { ...headers } };
  if (body !== undefined) {
    opts.headers["Content-Type"] = "application/json";
    opts.body = JSON.stringify(body);
  }

  const request = { method, url, headers: opts.headers, body };

  let res, text;
  try {
    res = await fetch(url, opts);
    text = await res.text();
  } catch (e) {
    return { request, response: { error: String(e) }, ok: false };
  }

  let parsed = null;
  try {
    parsed = JSON.parse(text);
  } catch {
    // not JSON, fall through to raw text below
  }

  const response = {
    status: res.status,
    statusText: res.statusText,
    url: res.url,
    redirected: res.redirected,
    headers: Object.fromEntries(res.headers.entries()),
    body: parsed ?? text,
  };

  return { request, response, ok: res.ok };
}

// jwtPayload: decodes a JWT's middle segment for display — no signature
// verification, the server already did that. Display only.
export function jwtPayload(token) {
  try {
    const b64 = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
    return JSON.parse(atob(padded));
  } catch (e) {
    return { error: String(e) };
  }
}

// sha256Base64Url: RFC 7636 §4.2's code_challenge transform, computed
// client-side the same way a real OAuth client would, via WebCrypto.
export async function sha256Base64Url(str) {
  const data = new TextEncoder().encode(str);
  const digest = await crypto.subtle.digest("SHA-256", data);
  const bytes = new Uint8Array(digest);
  let bin = "";
  for (const b of bytes) bin += String.fromCharCode(b);
  return btoa(bin).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

export function randomString(len) {
  const bytes = crypto.getRandomValues(new Uint8Array(len));
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}
