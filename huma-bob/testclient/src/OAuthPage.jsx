import { useState } from "react";
import { apiCall, jwtPayload, randomString, sha256Base64Url } from "./api.js";
import Exchange from "./Exchange.jsx";
import ActorBadge from "./ActorBadge.jsx";

export default function OAuthPage() {
  const [client, setClient] = useState("internal-service");
  const [redirectUri, setRedirectUri] = useState("http://localhost:5173/callback");
  const [scope, setScope] = useState("read:user_password read:user_email");
  const [code, setCode] = useState("");
  const [consent, setConsent] = useState(null); // { consentId } once pending
  const [authorizeResult, setAuthorizeResult] = useState(null);
  const [decisionResult, setDecisionResult] = useState(null);
  const [exchangeResult, setExchangeResult] = useState(null); // client -> this app's /api/exchange
  const [serverExchangeResult, setServerExchangeResult] = useState(null); // /api/exchange -> gorest
  const [decodedToken, setDecodedToken] = useState(null);
  const [pkceVerifier, setPkceVerifier] = useState("");

  const accessToken = localStorage.getItem("accessToken") || "";

  async function authorize() {
    setConsent(null);
    const state = randomString(8);
    const verifier = randomString(32);
    const challenge = await sha256Base64Url(verifier);
    setPkceVerifier(verifier);

    const url =
      "/oauth/authorize?" +
      new URLSearchParams({
        client_id: client,
        redirect_uri: redirectUri,
        response_type: "code",
        scope,
        state,
        code_challenge: challenge,
        code_challenge_method: "S256",
      });

    const r = await apiCall("GET", url, { Authorization: "Bearer " + accessToken });
    setAuthorizeResult(r);

    if (r.response.body?.consentRequired) {
      setConsent({ consentId: r.response.body.consentId });
      return;
    }

    // non-consent path: fetch followed the 302 to redirectUri (this app's
    // own /callback), whose query string (in the final response.url)
    // carries code/state — see Callback.jsx's doc comment for why we can't
    // read the raw 302/Location instead.
    const finalUrl = new URL(r.response.url);
    const gotCode = finalUrl.searchParams.get("code");
    if (gotCode) setCode(gotCode);
  }

  async function decide(approve) {
    const r = await apiCall(
      "POST",
      "/oauth/decision",
      { Authorization: "Bearer " + accessToken },
      { consentId: consent.consentId, approve },
    );
    setDecisionResult(r);

    const finalUrl = new URL(r.response.url);
    const gotCode = finalUrl.searchParams.get("code");
    const error = finalUrl.searchParams.get("error");
    if (gotCode) setCode(gotCode);
    if (error) setCode("(denied: " + error + ")");
  }

  async function exchange() {
    // client -> this app's own backend (vite.config.js's /api/exchange) —
    // note no client_secret in this request at all, the browser never
    // touches it.
    const r = await apiCall(
      "POST",
      window.location.origin + "/api/exchange",
      {},
      { code, redirect_uri: redirectUri, client_id: client, code_verifier: pkceVerifier },
    );
    setExchangeResult(r);

    // /api/exchange -> gorest — the actual confidential hop, reconstructed
    // from what the backend reports doing (see vite.config.js). This is
    // the only place client_secret ever appears, and it never left Node.
    const serverHop = r.response.body?.server;
    if (serverHop) {
      setServerExchangeResult({ request: serverHop.request, response: serverHop.response, ok: serverHop.response.status < 400 });

      if (serverHop.response.body?.access_token) {
        setDecodedToken(jwtPayload(serverHop.response.body.access_token));
      }
    }
  }

  return (
    <div>
      <h1>OAuth2 Authorization Code + PKCE</h1>
      <p className="hint">
        Needs an access token from the Login tab first — that's the "already-logged-in user"
        /oauth/authorize identifies via the Authorization header (see pkg/oauthserver's package doc for
        why that's not a real browser redirect).
      </p>
      <p className="hint">
        {accessToken ? (
          <>Using access token for: <code>{JSON.stringify(jwtPayload(accessToken))}</code></>
        ) : (
          <span className="err-text">No access token in localStorage — log in first.</span>
        )}
      </p>

      <fieldset>
        <label>Client</label>
        <select
          value={client}
          onChange={(e) => {
            setClient(e.target.value);
            setConsent(null);
          }}
        >
          <option value="internal-service">internal-service (auto-approved, no consent)</option>
          <option value="partner-app">partner-app (RequireConsent)</option>
        </select>
        <label>redirect_uri</label>
        <input value={redirectUri} onChange={(e) => setRedirectUri(e.target.value)} />
        <label>scope</label>
        <input value={scope} onChange={(e) => setScope(e.target.value)} />
        <p><ActorBadge actor="client" /> — calls /oauth/authorize directly (no secret needed, only your own bearer token).</p>
        <button onClick={authorize}>1. Authorize</button>
      </fieldset>
      <Exchange result={authorizeResult} />

      {consent && (
        <fieldset>
          <p><ActorBadge actor="client" /> — consent decision, also no secret needed.</p>
          <p>Consent required — approve or deny:</p>
          <button onClick={() => decide(true)}>Approve</button>
          <button onClick={() => decide(false)}>Deny</button>
        </fieldset>
      )}
      <Exchange result={decisionResult} />

      <fieldset>
        <label>code (from Authorize/Decide above, or paste one)</label>
        <input value={code} onChange={(e) => setCode(e.target.value)} />
        <p><ActorBadge actor="client" /> — sends code+code_verifier to this app's own backend, no client_secret in this request.</p>
        <button onClick={exchange}>2. Exchange for token</button>
      </fieldset>
      <Exchange result={exchangeResult} />

      {serverExchangeResult && (
        <>
          <p><ActorBadge actor="server" /> — the confidential hop: only here does client_secret get attached, server-to-server, never in browser JS.</p>
          <p className="gap-note">
            ⚠️ Known gap: /api/exchange hands the raw access_token/refresh_token straight back to this browser
            below (so this page can display them) — a real confidential-client backend would normally keep the
            token itself and give the browser only an httpOnly session cookie instead. What's actually
            confidential here is the exchange call (the secret), not what happens to the token afterward.
          </p>
          <Exchange result={serverExchangeResult} />
        </>
      )}

      {decodedToken && (
        <div className="panel">
          <h3>Decoded access_token payload</h3>
          <pre>{JSON.stringify(decodedToken, null, 2)}</pre>
        </div>
      )}
    </div>
  );
}
