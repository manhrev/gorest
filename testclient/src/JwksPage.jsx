import { useState } from "react";
import { apiCall, verifyJwtWithJwks } from "./api.js";
import Exchange from "./Exchange.jsx";

export default function JwksPage() {
  const [fetchResult, setFetchResult] = useState(null);
  const [jwks, setJwks] = useState(null);
  const [token, setToken] = useState("");
  const [verdict, setVerdict] = useState(null);

  async function fetchJwks() {
    const r = await apiCall("GET", "/.well-known/jwks.json");
    setFetchResult(r);
    if (r.ok && r.response.body?.keys) setJwks(r.response.body);
  }

  async function verify() {
    if (!jwks) return;
    setVerdict(await verifyJwtWithJwks(token.trim(), jwks));
  }

  return (
    <div>
      <h1>JWKS</h1>
      <p className="hint">
        Fetches the server's public keys, then verifies a token's EdDSA signature entirely client-side via
        WebCrypto (<code>crypto.subtle.verify</code>) — no server call for the verify step itself.
      </p>

      <fieldset>
        <button onClick={fetchJwks}>Fetch /.well-known/jwks.json</button>
      </fieldset>
      <Exchange result={fetchResult} />

      <fieldset>
        <label>Token to verify</label>
        <input
          value={token}
          onChange={(e) => setToken(e.target.value)}
          placeholder="paste an access token, or leave blank to use localStorage's"
        />
        <button
          onClick={() => {
            if (!token) setToken(localStorage.getItem("accessToken") || "");
          }}
        >
          Fill from localStorage
        </button>
        <button onClick={verify} disabled={!jwks}>
          Verify with JWKS
        </button>
        {!jwks && <p className="hint">Fetch the JWKS above first.</p>}
      </fieldset>

      {verdict && (
        <div className={"panel res" + (verdict.valid ? "" : " err")}>
          <h3>{verdict.valid ? `✓ signature valid (kid ${verdict.kid})` : `✗ invalid — ${verdict.reason}`}</h3>
          {verdict.payload && <pre>{JSON.stringify(verdict.payload, null, 2)}</pre>}
        </div>
      )}
    </div>
  );
}
