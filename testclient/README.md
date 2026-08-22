# testclient

Minimal Vite + React app for manually exercising gorest's auth + OAuth2
flows in a browser — dev tooling only, not part of the API.

```sh
pnpm install
pnpm dev   # http://localhost:5173
```

The gorest API server must be running at `http://localhost:8080` (`go run
./cmd` from the repo root) — every request here is cross-origin against it.

Pages:
- **Login** — `alice`/`hunter2` (stub creds, `internal/server/auth_stub.go`) → `/auth/login`, plus Check Auth / Logout.
- **OAuth** — Authorization Code + PKCE flow against the two stub clients (`internal/server/oauth_stub.go`): `internal-service` (auto-approved) and `partner-app` (requires consent).
- **`/callback`** — the `redirect_uri` both stub clients accept. Vite's dev server serves `index.html` for any unmatched path by default, so this "just works" without extra routing config.

`vite.config.js` pins the dev server to port 5173 (`strictPort: true`) —
that exact URL is hardcoded as a registered `redirect_uri` server-side, so a
silent port bump to 5174 would break the OAuth flow with no obvious cause.
