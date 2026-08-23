// ActorBadge: labels which actor makes a given HTTP call — the browser
// itself ("client") or this app's confidential backend, vite.config.js's
// /api/exchange route ("server"). Exists so it's visually unambiguous
// which hop could legitimately hold a client_secret (only "server" ever
// can) and which can't (anything labeled "client").
export default function ActorBadge({ actor }) {
  return actor === "server" ? (
    <span className="badge server">🔒 Server (vite.config.js — /api/exchange)</span>
  ) : (
    <span className="badge client">🖥️ Client (browser)</span>
  );
}
