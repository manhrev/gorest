// Callback: the redirect_uri target both stub clients accept
// (http://localhost:5173/callback). Its only real job is existing at a
// same-origin URL so OAuthPage's fetch(...) call can follow the 302 here
// and read response.url — browsers make a 302's Location header itself
// unreadable to JS by spec (redirect:"manual" returns a fully opaque
// response, status 0, no headers, for any caller — not a restriction
// specific to this app). Also renders the query string if you land on it
// directly, e.g. testing a real top-level browser redirect.
export default function Callback() {
  return (
    <div>
      <h1>OAuth callback landing page</h1>
      <p className="hint">
        This is redirect_uri — a real client's backend would read code/state (or error) off this URL
        and call /oauth/token itself.
      </p>
      <pre>{window.location.search || "(no query string)"}</pre>
    </div>
  );
}
