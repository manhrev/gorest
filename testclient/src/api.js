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

// decodeJwtSegment: base64url JSON segment (header or payload) → object.
function decodeJwtSegment(b64url) {
  const b64 = b64url.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  return JSON.parse(atob(padded));
}

// jwtPayload: decodes a JWT's middle segment for display — no signature
// verification, the server already did that. Display only.
export function jwtPayload(token) {
  try {
    return decodeJwtSegment(token.split(".")[1]);
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

// base64urlToBytes: RFC 7515 §2's base64url (no padding) → raw bytes, used
// for both JWK "x" and the JWT signature segment.
function base64urlToBytes(b64url) {
  const b64 = b64url.replace(/-/g, "+").replace(/_/g, "/");
  const padded = b64 + "=".repeat((4 - (b64.length % 4)) % 4);
  const bin = atob(padded);
  return Uint8Array.from(bin, (c) => c.charCodeAt(0));
}

// verifyJwtWithJwks: verifies a compact JWT's EdDSA signature against a JWKS
// response, entirely client-side via WebCrypto's native Ed25519 support
// (Chrome 113+/Firefox 130+/Safari 17+ — no jose/jsrsasign dependency
// needed). Picks the JWK matching the token's kid header, falling back to
// the first key if the token has none. Returns { valid, reason?, payload }.
export async function verifyJwtWithJwks(token, jwks) {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return { valid: false, reason: "not a compact JWT (want 3 dot-separated parts)" };
  }
  const [headerB64, payloadB64, sigB64] = parts;

  let header, payload;
  try {
    header = decodeJwtSegment(headerB64);
    payload = decodeJwtSegment(payloadB64);
  } catch (e) {
    return { valid: false, reason: "malformed header/payload: " + String(e) };
  }

  if (header.alg !== "EdDSA") {
    return { valid: false, reason: `unsupported alg ${header.alg}, only EdDSA is checked here`, payload };
  }

  const jwk = header.kid ? jwks.keys.find((k) => k.kid === header.kid) : jwks.keys[0];
  if (!jwk) {
    return { valid: false, reason: `no JWK with kid ${header.kid} in JWKS`, payload };
  }

  let cryptoKey;
  try {
    cryptoKey = await crypto.subtle.importKey("raw", base64urlToBytes(jwk.x), { name: "Ed25519" }, false, ["verify"]);
  } catch (e) {
    return { valid: false, reason: "browser can't import Ed25519 key (needs a recent Chrome/Firefox/Safari): " + String(e), payload };
  }

  const signingInput = new TextEncoder().encode(`${headerB64}.${payloadB64}`);
  const signature = base64urlToBytes(sigB64);

  let valid;
  try {
    valid = await crypto.subtle.verify({ name: "Ed25519" }, cryptoKey, signature, signingInput);
  } catch (e) {
    return { valid: false, reason: "verify() threw: " + String(e), payload };
  }

  return { valid, kid: jwk.kid, payload, reason: valid ? undefined : "signature does not match" };
}
