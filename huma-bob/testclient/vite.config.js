import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

const API_BASE = "http://localhost:8080";

// Real confidential-client secrets — held server-side only, in this file.
// vite.config.js runs in Node (Vite's own dev-server process), never
// bundled/shipped to the browser — this makes the OAuth token exchange
// step genuinely confidential, unlike holding client_secret in browser JS
// (which OAuthPage.jsx used to do — a real page's source is always
// readable by whoever's running it, so nothing there can ever be a secret).
const CLIENT_SECRETS = { "internal-service": "dev-secret", "partner-app": "dev-secret" };

// oauthExchangeProxy: adds POST /api/exchange to Vite's dev server — the
// browser calls this same-origin route (no client_secret in that request
// at all) instead of gorest's /oauth/token directly. This route is this
// test client's stand-in "backend": it's the only thing that ever attaches
// client_secret, and it does so server-to-server. A real confidential
// client (a web app with a server component) works the same way, just
// with its backend being a whole separate service instead of a dev-server
// plugin.
//
// The response includes both the request this backend made to gorest and
// the response it got back, so OAuthPage.jsx can render that hop
// explicitly too — not just the browser→backend hop apiCall() already
// captures.
function oauthExchangeProxy() {
  return {
    name: "oauth-exchange-proxy",
    configureServer(server) {
      server.middlewares.use("/api/exchange", async (req, res) => {
        if (req.method !== "POST") {
          res.statusCode = 405;
          res.end();
          return;
        }

        let raw = "";
        for await (const chunk of req) raw += chunk;
        const { code, redirect_uri, client_id, code_verifier } = JSON.parse(raw);

        const serverRequestUrl = API_BASE + "/oauth/token";
        const serverRequestHeaders = { "Content-Type": "application/json" };
        const serverRequestBody = {
          grant_type: "authorization_code",
          code,
          redirect_uri,
          client_id,
          client_secret: CLIENT_SECRETS[client_id],
          code_verifier,
        };

        const upstream = await fetch(serverRequestUrl, {
          method: "POST",
          headers: serverRequestHeaders,
          body: JSON.stringify(serverRequestBody),
        });
        const text = await upstream.text();
        let upstreamBody = null;
        try {
          upstreamBody = JSON.parse(text);
        } catch {
          // not JSON, fall through to raw text below
        }

        const payload = {
          server: {
            request: { method: "POST", url: serverRequestUrl, headers: serverRequestHeaders, body: serverRequestBody },
            response: {
              status: upstream.status,
              statusText: upstream.statusText,
              url: upstream.url,
              headers: Object.fromEntries(upstream.headers.entries()),
              body: upstreamBody ?? text,
            },
          },
        };

        res.statusCode = upstream.status;
        res.setHeader("Content-Type", "application/json");
        res.end(JSON.stringify(payload));
      });
    },
  };
}

// strictPort matters here: the API server's stub OAuth clients
// (internal/server/oauth_stub.go) have http://localhost:5173/callback
// hardcoded as a registered redirect_uri. If Vite silently jumped to 5174
// because 5173 was busy, the OAuth flow would fail redirect_uri validation
// with no obvious reason why.
export default defineConfig({
  plugins: [react(), oauthExchangeProxy()],
  server: {
    port: 5173,
    strictPort: true,
    // Needed for OAuthPage's fetch(...) to follow the 302 from
    // /oauth/authorize back to /callback: once a fetch starts
    // cross-origin, every hop of its redirect chain is CORS-checked —
    // even the final one landing back on this app's own origin. Without
    // this, Vite's dev server sends no Access-Control-Allow-Origin on its
    // own responses and the browser blocks that last hop.
    cors: true,
  },
});
