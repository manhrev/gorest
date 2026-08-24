// OAuth 2.0 Client Credentials Grant (RFC 6749 §4.4) — step data
export const actors = ["client", "auth", "resource"];

// static info shown when an actor is clicked
export const actorInfo = {
  auth: {
    label: "Auth Server",
    host: "auth.example.com",
    endpoints: [
      "POST /oauth/token — issue access token",
      "GET /.well-known/jwks.json — public signing keys",
    ],
    holds: ["Registered clients: client_id → hashed client_secret", "Allowed scopes per client"],
  },
  resource: {
    label: "Resource Server",
    host: "api.example.com",
    endpoints: ["GET /resource — requires Bearer token, scope read:resource"],
    holds: ["Auth server's public key / JWKS (to verify token signature)"],
  },
};

// what the client is holding at each step index (grows as the flow progresses)
export function clientHolding(stepIndex) {
  const items = ["client_id", "client_secret"];
  if (stepIndex >= 3) items.push("access_token (Bearer, expires_in 3600s)");
  if (stepIndex >= 6) items.push("resource data (fetched)");
  return items;
}

export const steps = [
  {
    from: null,
    to: null,
    title: "0. Start",
    detail:
      "Client holds a client_id/client_secret issued by the auth server ahead of time. Nothing has been sent yet — press Next to fire the token request.",
    http: `# no request sent yet`,
  },
  {
    from: "client",
    to: "auth",
    title: "1. Client requests token",
    detail:
      "Client authenticates itself directly (no user involved) and asks for an access token, optionally scoped.",
    http: `POST /oauth/token HTTP/1.1
Host: auth.example.com
Authorization: Basic base64(client_id:client_secret)
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&scope=read:resource`,
  },
  {
    from: "auth",
    to: "auth",
    title: "2. Auth server validates client",
    detail:
      "Checks client_id/client_secret, confirms client is allowed the client_credentials grant and the requested scope.",
    http: `# server-side
lookup client by client_id
verify client_secret (hash compare)
check grant_type "client_credentials" allowed
check requested scope <= client's allowed scopes`,
  },
  {
    from: "auth",
    to: "client",
    title: "3. Auth server issues access token",
    detail:
      "No refresh token here — RFC 6749 §4.4.3 says refresh tokens SHOULD NOT be issued for this grant, since the client can just re-authenticate.",
    http: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:resource"
}`,
  },
  {
    from: "client",
    to: "resource",
    title: "4. Client calls resource server",
    detail:
      "Client presents the bearer token on the actual API call — same as any other grant from this point on.",
    http: `GET /resource HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...`,
  },
  {
    from: "resource",
    to: "resource",
    title: "5. Resource server validates token",
    detail:
      "Verifies signature/expiry (locally, or via introspection), checks scope covers the endpoint, then serves data.",
    http: `# server-side
verify JWT signature + exp
check scope "read:resource" permits this endpoint`,
  },
  {
    from: "resource",
    to: "client",
    title: "6. Resource returned",
    detail: "Client gets the data. No user session, no redirect — pure machine-to-machine.",
    http: `HTTP/1.1 200 OK
Content-Type: application/json

{ "data": "..." }`,
  },
];
