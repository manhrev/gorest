import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// strictPort matters here: the API server's stub OAuth clients
// (internal/server/oauth_stub.go) have http://localhost:5173/callback
// hardcoded as a registered redirect_uri. If Vite silently jumped to 5174
// because 5173 was busy, the OAuth flow would fail redirect_uri validation
// with no obvious reason why.
export default defineConfig({
  plugins: [react()],
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
