// OAuth 2.0 Authorization Code Grant (RFC 6749 §4.1) for a PUBLIC client — a single-page
// app running entirely in the browser. It cannot hold a client_secret, so PKCE (RFC 7636)
// is mandatory here, not optional. Only 3 actors: the browser IS the client, no backend hop.
import type { ActorId, FlowGraph, Outcome } from '../types'

export const START_STEP = 'start'

export const actorInfo: Record<'auth' | 'resource', { label: string; host: string; endpoints: string[]; holds: string[] }> = {
  auth: {
    label: 'Auth Server',
    host: 'auth.example.com',
    endpoints: [
      'GET /authorize — user login + consent, issues a code',
      'POST /oauth/token — exchange code for tokens (CORS-enabled, no client secret required)',
    ],
    holds: [
      'Registered public clients: client_id (no secret), redirect_uri allow-list, PKCE-required flag',
      'Issued authorization codes (single-use, short-lived) → code_challenge',
      'Issued access/refresh tokens (refresh_token rotates on each use)',
    ],
  },
  resource: {
    label: 'Resource Server',
    host: 'api.example.com',
    endpoints: ['GET /resource — requires Bearer token, scope read:resource'],
    holds: ["Auth server's public key / JWKS (to verify token signature)"],
  },
}

export function clientHolding(opts: {
  pkceGenerated: boolean
  codeReceived: boolean
  tokenIssued: boolean
  resourceFetched: boolean
}): string[] {
  const items = ['client_id (public — embedded in the SPA bundle, not secret)']
  if (opts.pkceGenerated) items.push('code_verifier (in-memory/sessionStorage, mandatory PKCE)')
  if (opts.codeReceived) items.push('authorization_code (single-use)')
  if (opts.tokenIssued) items.push('access_token (Bearer, kept in memory)', 'refresh_token (rotating, single-use)')
  if (opts.resourceFetched) items.push('resource data (fetched)')
  return items
}

function terminalError(id: string, from: ActorId, to: ActorId, detail: string) {
  return {
    id,
    from,
    to,
    outcomes: [
      {
        id: 'terminal',
        label: 'End (error)',
        kind: 'error' as const,
        title: 'Flow terminated',
        detail,
        http: `# no further requests — client should surface the error and stop`,
        next: null,
      },
    ],
  }
}

const validateCodeOutcomes: Outcome[] = [
  {
    id: 'valid',
    label: 'Code + PKCE valid',
    kind: 'happy',
    title: '4. Auth server validates the code',
    detail:
      "Checks the code hasn't been used, matches redirect_uri, and verifies SHA256(code_verifier) equals the stored code_challenge — this substitutes for the client_secret a confidential client would present.",
    http: `# server-side
lookup authorization_code, check unused + not expired
verify redirect_uri matches the one used at /authorize
verify SHA256(code_verifier) base64url == stored code_challenge
=> OK, continue to token issuance`,
    next: 'issue-tokens',
  },
  {
    id: 'invalid_grant',
    label: 'Code expired / already used',
    kind: 'error',
    title: '4. Auth server validates the code — failed',
    detail: 'Authorization codes are single-use and short-lived (~60s). A replay or timeout lands here.',
    http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_grant",
  "error_description": "Authorization code is invalid, expired, or already used"
}`,
    next: 'error-invalid-grant',
  },
  {
    id: 'pkce_mismatch',
    label: 'PKCE verifier mismatch',
    kind: 'error',
    title: '4. Auth server validates the code — failed',
    detail:
      'code_verifier does not hash to the code_challenge stored for this code. Without PKCE this is exactly the "authorization code interception" attack public clients are exposed to.',
    http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_grant",
  "error_description": "PKCE verification failed"
}`,
    next: 'error-pkce-mismatch',
  },
]

export const graph: FlowGraph = {
  start: {
    id: 'start',
    from: null,
    to: null,
    outcomes: [
      {
        id: 'happy',
        label: 'Start',
        kind: 'happy',
        title: '0. Start',
        detail:
          'Public client — a single-page app running entirely in the browser. It cannot hold a client_secret (anyone could extract it from the JS bundle), so PKCE is mandatory here, not optional. Nothing sent yet — press Next.',
        http: `# no request sent yet`,
        next: 'redirect-to-auth',
      },
    ],
  },

  'redirect-to-auth': {
    id: 'redirect-to-auth',
    from: 'client',
    to: 'auth',
    outcomes: [
      {
        id: 'happy',
        label: 'Redirected',
        kind: 'happy',
        title: '1. Client redirects to the authorization endpoint',
        detail:
          'SPA generates a random state (CSRF protection) and a mandatory PKCE code_verifier/code_challenge (S256) pair, stores the verifier in memory/sessionStorage, then navigates the browser — there is no separate backend to do this on its behalf.',
        http: `# browser navigation (window.location.assign)
GET https://auth.example.com/authorize?
  response_type=code&
  client_id=spa-app&
  redirect_uri=https://app.example.com/callback&
  scope=read:resource&
  state=xyzABC123&
  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
  code_challenge_method=S256`,
        next: 'authenticate',
      },
    ],
  },

  authenticate: {
    id: 'authenticate',
    from: 'auth',
    to: 'auth',
    outcomes: [
      {
        id: 'approved',
        label: 'User logs in & approves',
        kind: 'happy',
        title: '2. Auth server authenticates the user and requests consent',
        detail: 'User logs in (if not already) and approves the requested scope.',
        http: `# server-side
show login form (if no session) + consent screen for scope "read:resource"
user approves
=> generate authorization_code, bind it to client_id + redirect_uri + code_challenge`,
        next: 'issue-code',
      },
      {
        id: 'denied',
        label: 'User denies consent',
        kind: 'error',
        title: '2. Auth server authenticates the user and requests consent — denied',
        detail: 'User declines the consent screen. Auth server redirects back with an error instead of a code.',
        http: `HTTP/1.1 302 Found
Location: https://app.example.com/callback?error=access_denied&state=xyzABC123`,
        next: 'error-access-denied',
      },
      {
        id: 'login_failed',
        label: 'Login failed',
        kind: 'error',
        title: '2. Auth server authenticates the user and requests consent — failed',
        detail: 'Wrong username/password. No redirect back to the client yet — user is stuck at the auth server.',
        http: `# server-side
invalid credentials
=> re-render login form with an error (no redirect to client)`,
        next: 'error-login-failed',
      },
    ],
  },

  'issue-code': {
    id: 'issue-code',
    from: 'auth',
    to: 'client',
    outcomes: [
      {
        id: 'happy',
        label: 'Code issued',
        kind: 'happy',
        title: '3. Auth server redirects the browser back with a code',
        detail:
          "Browser navigates back into the SPA's own route — the client-side router picks up ?code=... from the URL, no server-rendered page in between.",
        http: `HTTP/1.1 302 Found
Location: https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyzABC123`,
        next: 'exchange-code',
      },
    ],
  },

  'exchange-code': {
    id: 'exchange-code',
    from: 'client',
    to: 'auth',
    outcomes: [
      {
        id: 'happy',
        label: 'Exchange sent',
        kind: 'happy',
        title: '3b. Client exchanges the code for tokens',
        detail:
          "Called directly from the browser via fetch() — no backend hop. No client_secret: the code_verifier is what proves this is the same app that started the flow. Needs CORS enabled on the auth server's /oauth/token endpoint.",
        http: `POST /oauth/token HTTP/1.1
Host: auth.example.com
Content-Type: application/x-www-form-urlencoded
Origin: https://app.example.com

grant_type=authorization_code&
client_id=spa-app&
code=SplxlOBeZQQYbYS6WxSbIA&
redirect_uri=https://app.example.com/callback&
code_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk`,
        next: 'validate-code',
      },
    ],
  },

  'validate-code': {
    id: 'validate-code',
    from: 'auth',
    to: 'auth',
    outcomes: validateCodeOutcomes,
  },

  'issue-tokens': {
    id: 'issue-tokens',
    from: 'auth',
    to: 'client',
    outcomes: [
      {
        id: 'happy',
        label: 'Tokens issued',
        kind: 'happy',
        title: '5. Auth server issues tokens',
        detail:
          'Access token is short-lived. The refresh_token (if issued at all to a public client) rotates on every use and is single-use, since nothing but PKCE protects this client.',
        http: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 900,
  "refresh_token": "rot_8xLOxBtZp8",
  "scope": "read:resource"
}`,
        next: 'call-resource',
      },
    ],
  },

  'call-resource': {
    id: 'call-resource',
    from: 'client',
    to: 'resource',
    outcomes: [
      {
        id: 'happy',
        label: 'Call sent',
        kind: 'happy',
        title: '6. Client calls the resource server',
        detail: 'Client presents the bearer token on the actual API call — same as any other grant from this point on.',
        http: `GET /resource HTTP/1.1
Host: api.example.com
Authorization: Bearer eyJhbGciOi...`,
        next: 'validate-token',
      },
    ],
  },

  'validate-token': {
    id: 'validate-token',
    from: 'resource',
    to: 'resource',
    outcomes: [
      {
        id: 'valid',
        label: 'Valid token',
        kind: 'happy',
        title: '7. Resource server validates the token',
        detail: 'Verifies signature/expiry (locally, or via introspection), checks scope covers the endpoint, then serves data.',
        http: `# server-side
verify JWT signature + exp
check scope "read:resource" permits this endpoint
=> OK, serve resource`,
        next: 'resource-returned',
      },
      {
        id: 'expired_token',
        label: 'Token expired',
        kind: 'error',
        title: '7. Resource server validates the token — failed',
        detail:
          'Signature is valid but exp has passed. Client uses the rotating refresh_token to get a new access_token — no need to send the user through login again.',
        http: `HTTP/1.1 401 Unauthorized
WWW-Authenticate: Bearer error="invalid_token", error_description="The access token expired"
Content-Type: application/json

{
  "error": "invalid_token",
  "error_description": "The access token expired"
}`,
        next: 'error-expired',
      },
      {
        id: 'insufficient_scope',
        label: 'Insufficient scope',
        kind: 'error',
        title: '7. Resource server validates the token — failed',
        detail: 'Token is valid, but its scope does not cover this endpoint.',
        http: `HTTP/1.1 403 Forbidden
WWW-Authenticate: Bearer error="insufficient_scope", scope="read:resource"
Content-Type: application/json

{
  "error": "insufficient_scope",
  "error_description": "Token does not carry the required scope"
}`,
        next: 'error-scope',
      },
    ],
  },

  'resource-returned': {
    id: 'resource-returned',
    from: 'resource',
    to: 'client',
    outcomes: [
      {
        id: 'happy',
        label: 'Data returned',
        kind: 'happy',
        title: '8. Resource returned',
        detail: 'Client has the data it needs to render the page.',
        http: `HTTP/1.1 200 OK
Content-Type: application/json

{ "data": "..." }`,
        next: 'render-page',
      },
    ],
  },

  'render-page': {
    id: 'render-page',
    from: 'client',
    to: 'client',
    outcomes: [
      {
        id: 'happy',
        label: 'Page rendered',
        kind: 'happy',
        title: '9. Client renders the page',
        detail:
          'SPA updates its own UI in-browser with the fetched data. Tokens stay in memory (never localStorage/cookies) to limit the blast radius of an XSS bug.',
        http: `# client-side
render view with resource data`,
        next: null,
      },
    ],
  },

  // --- terminal error steps ---
  'error-access-denied': terminalError('error-access-denied', 'auth', 'client', 'User declined consent — flow ends here.'),
  'error-login-failed': terminalError('error-login-failed', 'auth', 'auth', 'Login failed — flow ends here.'),
  'error-invalid-grant': terminalError('error-invalid-grant', 'auth', 'client', 'Code exchange rejected — flow ends here.'),
  'error-pkce-mismatch': terminalError('error-pkce-mismatch', 'auth', 'client', 'PKCE verification failed — flow ends here.'),
  'error-expired': terminalError('error-expired', 'resource', 'client', 'Token rejected as expired — flow ends here.'),
  'error-scope': terminalError('error-scope', 'resource', 'client', 'Token rejected for insufficient scope — flow ends here.'),
}
