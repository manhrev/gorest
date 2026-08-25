// OAuth 2.0 Authorization Code Grant (RFC 6749 §4.1) with optional PKCE (RFC 7636),
// for a CONFIDENTIAL client (a backend app that can hold a client_secret — not a SPA/mobile app).
import type { ActorId, FlowGraph, FlowStep, Outcome } from '../types'

export const START_STEP = 'start'

// static info shown when an actor is clicked
export const actorInfo: Record<'auth' | 'resource', { label: string; host: string; endpoints: string[]; holds: string[] }> = {
  auth: {
    label: 'Auth Server',
    host: 'auth.example.com',
    endpoints: [
      'GET /authorize — user login + consent, issues a code',
      'POST /oauth/token — exchange code (or refresh_token) for tokens',
    ],
    holds: [
      'Registered clients: client_id → hashed client_secret, redirect_uri allow-list',
      'Issued authorization codes (single-use, short-lived) → code_challenge if PKCE used',
      "Issued refresh_token / access_token records",
    ],
  },
  resource: {
    label: 'Resource Server',
    host: 'api.example.com',
    endpoints: ['GET /resource — requires Bearer token, scope read:resource'],
    holds: ["Auth server's public key / JWKS (to verify token signature)"],
  },
}

export function clientHolding(opts: { pkceGenerated: boolean; codeReceived: boolean; tokenIssued: boolean; resourceFetched: boolean }, usePkce: boolean): string[] {
  const items = ['client_id', 'client_secret (confidential — never exposed to the browser)']
  if (opts.pkceGenerated && usePkce) items.push('code_verifier (kept server-side, never sent until token exchange)')
  if (opts.codeReceived) items.push('authorization_code (single-use)')
  if (opts.tokenIssued) items.push('access_token (Bearer)', 'refresh_token')
  if (opts.resourceFetched) items.push('resource data (fetched)')
  return items
}

export function userHolding(opts: { stateIssued: boolean }): string[] {
  const items = ['browser session cookie with the client app (none yet, pre-login)']
  if (opts.stateIssued) items.push('state param round-tripped via redirects (CSRF check)')
  return items
}

function terminalError(id: string, from: ActorId, to: ActorId, detail: string): FlowStep {
  return {
    id,
    from,
    to,
    outcomes: [
      {
        id: 'terminal',
        label: 'End (error)',
        kind: 'error',
        title: 'Flow terminated',
        detail,
        http: `# no further requests — client should surface the error and stop`,
        next: null,
      },
    ],
  }
}

/** Graph content differs slightly (PKCE params on the wire) depending on whether PKCE is enabled. */
export function buildGraph(usePkce: boolean): FlowGraph {
  const pkceChallengeLines = usePkce
    ? `  code_challenge=E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM&
  code_challenge_method=S256`
    : undefined

  const validateCodeOutcomes: Outcome[] = [
    {
      id: 'valid',
      label: 'Code + PKCE valid',
      kind: 'happy',
      title: '8. Auth server validates the code',
      detail: usePkce
        ? "Checks the code hasn't been used, matches redirect_uri and client credentials, and verifies SHA256(code_verifier) equals the stored code_challenge."
        : "Checks the code hasn't been used and matches redirect_uri and client credentials.",
      http: `# server-side
lookup authorization_code, check unused + not expired
verify client_id/client_secret (confidential client, Basic auth)
verify redirect_uri matches the one used at /authorize${
        usePkce ? '\nverify SHA256(code_verifier) base64url == stored code_challenge' : ''
      }
=> OK, continue to token issuance`,
      next: 'issue-tokens',
    },
    {
      id: 'invalid_grant',
      label: 'Code expired / already used',
      kind: 'error',
      title: '8. Auth server validates the code — failed',
      detail: 'Authorization codes are single-use and short-lived (~60s). A replay or timeout lands here.',
      http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_grant",
  "error_description": "Authorization code is invalid, expired, or already used"
}`,
      next: 'error-invalid-grant',
    },
  ]
  if (usePkce) {
    validateCodeOutcomes.push({
      id: 'pkce_mismatch',
      label: 'PKCE verifier mismatch',
      kind: 'error',
      title: '8. Auth server validates the code — failed',
      detail: 'code_verifier does not hash to the code_challenge stored for this code — someone other than the original requester is trying to redeem it.',
      http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_grant",
  "error_description": "PKCE verification failed"
}`,
      next: 'error-pkce-mismatch',
    })
  }

  const graph: FlowGraph = {
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
          detail: `Confidential client (a backend app — holds a client_secret). ${
            usePkce
              ? 'PKCE is enabled: the client will also generate a code_verifier/code_challenge pair, defense-in-depth against code interception even though it already has a secret.'
              : 'PKCE is off — this client relies on its client_secret alone to redeem the code, which is acceptable for a confidential client.'
          } Nothing sent yet — press Next.`,
          http: `# no request sent yet`,
          next: 'request-page',
        },
      ],
    },

    'request-page': {
      id: 'request-page',
      from: 'user',
      to: 'client',
      outcomes: [
        {
          id: 'happy',
          label: 'Page requested',
          kind: 'happy',
          title: '1. User requests a protected page',
          detail: "Browser hits the client app; no session yet, so the client kicks off the OAuth dance.",
          http: `GET /dashboard HTTP/1.1
Host: app.example.com
Cookie: (none — no session yet)`,
          next: 'redirect-to-auth',
        },
      ],
    },

    'redirect-to-auth': {
      id: 'redirect-to-auth',
      from: 'client',
      to: 'user',
      outcomes: [
        {
          id: 'happy',
          label: 'Redirected',
          kind: 'happy',
          title: '2. Client redirects browser to the authorization endpoint',
          detail: usePkce
            ? 'Client generates a random state (CSRF protection) and a PKCE code_verifier, derives code_challenge = BASE64URL(SHA256(code_verifier)), and stores the verifier server-side against this session.'
            : 'Client generates a random state (CSRF protection) and sends the browser to the auth server.',
          http: `HTTP/1.1 302 Found
Location: https://auth.example.com/authorize?
  response_type=code&
  client_id=app-backend&
  redirect_uri=https://app.example.com/callback&
  scope=read:resource&
  state=xyzABC123${pkceChallengeLines ? '&\n' + pkceChallengeLines : ''}`,
          next: 'follow-redirect-auth',
        },
      ],
    },

    'follow-redirect-auth': {
      id: 'follow-redirect-auth',
      from: 'user',
      to: 'auth',
      outcomes: [
        {
          id: 'happy',
          label: 'Arrived at auth server',
          kind: 'happy',
          title: '3. Browser follows the redirect to the auth server',
          detail: 'This request goes straight from the browser to the auth server — the client is not in the loop here.',
          http: `GET /authorize?response_type=code&client_id=app-backend&redirect_uri=...&scope=read:resource&state=xyzABC123${
            usePkce ? '&code_challenge=...&code_challenge_method=S256' : ''
          } HTTP/1.1
Host: auth.example.com`,
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
          title: '4. Auth server authenticates the user and requests consent',
          detail: 'User logs in (if not already) and approves the requested scope.',
          http: `# server-side
show login form (if no session) + consent screen for scope "read:resource"
user approves
=> generate authorization_code, bind it to client_id + redirect_uri${usePkce ? ' + code_challenge' : ''}`,
          next: 'issue-code',
        },
        {
          id: 'denied',
          label: 'User denies consent',
          kind: 'error',
          title: '4. Auth server authenticates the user and requests consent — denied',
          detail: 'User declines the consent screen. Auth server redirects back with an error instead of a code.',
          http: `HTTP/1.1 302 Found
Location: https://app.example.com/callback?error=access_denied&state=xyzABC123`,
          next: 'error-access-denied',
        },
        {
          id: 'login_failed',
          label: 'Login failed',
          kind: 'error',
          title: '4. Auth server authenticates the user and requests consent — failed',
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
      to: 'user',
      outcomes: [
        {
          id: 'happy',
          label: 'Code issued',
          kind: 'happy',
          title: '5. Auth server redirects the browser back with a code',
          detail: 'The code is short-lived and single-use — it only stands in for "this user approved this client for this scope, once".',
          http: `HTTP/1.1 302 Found
Location: https://app.example.com/callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyzABC123`,
          next: 'follow-redirect-client',
        },
      ],
    },

    'follow-redirect-client': {
      id: 'follow-redirect-client',
      from: 'user',
      to: 'client',
      outcomes: [
        {
          id: 'happy',
          label: 'Back at client',
          kind: 'happy',
          title: '6. Browser follows the redirect back to the client, with the code',
          detail: 'Client checks the returned state matches what it issued in step 2, then moves to exchange the code.',
          http: `GET /callback?code=SplxlOBeZQQYbYS6WxSbIA&state=xyzABC123 HTTP/1.1
Host: app.example.com`,
          next: 'send-code',
        },
      ],
    },

    'send-code': {
      id: 'send-code',
      from: 'client',
      to: 'auth',
      outcomes: [
        {
          id: 'happy',
          label: 'Exchange sent',
          kind: 'happy',
          title: '7. Client exchanges the code for tokens',
          detail: 'This call happens server-to-server (never through the browser), authenticated with the client_secret.',
          http: `POST /oauth/token HTTP/1.1
Host: auth.example.com
Authorization: Basic base64(client_id:client_secret)
Content-Type: application/x-www-form-urlencoded

grant_type=authorization_code&
code=SplxlOBeZQQYbYS6WxSbIA&
redirect_uri=https://app.example.com/callback${usePkce ? '&\ncode_verifier=dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk' : ''}`,
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
          title: '9. Auth server issues tokens',
          detail: 'Unlike client_credentials, a refresh_token is issued here — the client can renew access without the user round-tripping through login again.',
          http: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "refresh_token": "8xLOxBtZp8",
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
          title: '10. Client calls the resource server',
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
          title: '11. Resource server validates the token',
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
          title: '11. Resource server validates the token — failed',
          detail: 'Signature is valid but exp has passed. Client should use the refresh_token to get a new access_token — no need to send the user through login again.',
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
          title: '11. Resource server validates the token — failed',
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
          title: '12. Resource returned',
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
      to: 'user',
      outcomes: [
        {
          id: 'happy',
          label: 'Page rendered',
          kind: 'happy',
          title: '13. Client renders the page',
          detail: 'Client establishes its own session with the browser (cookie) and keeps the refresh_token server-side for later silent renewal.',
          http: `HTTP/1.1 200 OK
Content-Type: text/html
Set-Cookie: session=...; HttpOnly; Secure

<!-- rendered page using resource data -->`,
          next: null,
        },
      ],
    },

    // --- terminal error steps ---
    'error-access-denied': terminalError('error-access-denied', 'auth', 'user', 'User declined consent — flow ends here.'),
    'error-login-failed': terminalError('error-login-failed', 'auth', 'auth', 'Login failed — flow ends here.'),
    'error-invalid-grant': terminalError('error-invalid-grant', 'auth', 'client', 'Code exchange rejected — flow ends here.'),
    'error-expired': terminalError('error-expired', 'resource', 'client', 'Token rejected as expired — flow ends here.'),
    'error-scope': terminalError('error-scope', 'resource', 'client', 'Token rejected for insufficient scope — flow ends here.'),
  }

  if (usePkce) {
    graph['error-pkce-mismatch'] = terminalError(
      'error-pkce-mismatch',
      'auth',
      'client',
      'PKCE verification failed — flow ends here.',
    )
  }

  return graph
}
