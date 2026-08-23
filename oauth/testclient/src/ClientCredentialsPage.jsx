import { useState } from "react";
import { apiCall, jwtPayload } from "./api.js";
import Exchange from "./Exchange.jsx";
import ActorBadge from "./ActorBadge.jsx";

// ClientCredentialsPage: RFC 6749 §4.4 — no user, no redirect, no consent.
// A client authenticates with its own secret and gets a token for itself
// directly. Same "browser never sees client_secret" trick as OAuthPage's
// Exchange step: the request goes to this app's own /api/client-credentials
// (vite.config.js), which is the only place client_secret is attached.
export default function ClientCredentialsPage() {
  const [client, setClient] = useState("internal-service");
  const [scope, setScope] = useState("read:user_password read:user_email");
  const [browserResult, setBrowserResult] = useState(null); // browser -> this app's /api/client-credentials
  const [serverResult, setServerResult] = useState(null); // /api/client-credentials -> gorest
  const [decodedToken, setDecodedToken] = useState(null);

  async function requestToken() {
    setServerResult(null);
    setDecodedToken(null);

    const r = await apiCall(
      "POST",
      window.location.origin + "/api/client-credentials",
      {},
      { client_id: client, scope },
    );
    setBrowserResult(r);

    const serverHop = r.response.body?.server;
    if (serverHop) {
      setServerResult({ request: serverHop.request, response: serverHop.response, ok: serverHop.response.status < 400 });

      if (serverHop.response.body?.access_token) {
        setDecodedToken(jwtPayload(serverHop.response.body.access_token));
      }
    }
  }

  return (
    <div>
      <h1>OAuth2 Client Credentials Grant</h1>
      <p className="hint">
        No user in this flow at all — the client authenticates with its own client_secret and gets a token
        for itself directly (machine-to-machine). No /authorize, no /decision, no code.
      </p>

      <fieldset>
        <label>Client</label>
        <select value={client} onChange={(e) => setClient(e.target.value)}>
          <option value="internal-service">internal-service</option>
          <option value="partner-app">partner-app</option>
        </select>
        <label>scope</label>
        <input value={scope} onChange={(e) => setScope(e.target.value)} />
        <p>
          <ActorBadge actor="client" /> — sends client_id+scope to this app's own backend, no client_secret in
          this request.
        </p>
        <button onClick={requestToken}>Request token</button>
      </fieldset>
      <Exchange result={browserResult} />

      {serverResult && (
        <>
          <p>
            <ActorBadge actor="server" route="/api/client-credentials" /> — the confidential hop: only here does
            client_secret get attached, server-to-server, never in browser JS.
          </p>
          <p className="gap-note">
            ⚠️ Known gap: /api/client-credentials hands the raw access_token straight back to this browser below (so
            this page can display it) — a real confidential-client backend would normally keep the token itself.
          </p>
          <Exchange result={serverResult} />
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
