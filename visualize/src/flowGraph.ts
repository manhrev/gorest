// OAuth 2.0 Client Credentials Grant (RFC 6749 §4.4) — step graph.
// Ported from the original flow.js linear steps[]; branch points get >1 outcome.
import type { ActorId, FlowGraph } from './types'

export const actors: ActorId[] = ['client', 'auth', 'resource']

// static info shown when an actor is clicked
export const actorInfo: Record<Exclude<ActorId, 'client'>, { label: string; host: string; endpoints: string[]; holds: string[] }> = {
  auth: {
    label: 'Auth Server',
    host: 'auth.example.com',
    endpoints: [
      'POST /oauth/token — issue access token',
      'GET /.well-known/jwks.json — public signing keys',
    ],
    holds: ['Registered clients: client_id → hashed client_secret', 'Allowed scopes per client'],
  },
  resource: {
    label: 'Resource Server',
    host: 'api.example.com',
    endpoints: ['GET /resource — requires Bearer token, scope read:resource'],
    holds: ["Auth server's public key / JWKS (to verify token signature)"],
  },
}

/** What the client is holding, given how far along the currently-active path it is. */
export function clientHolding(opts: { tokenIssued: boolean; resourceFetched: boolean }): string[] {
  const items = ['client_id', 'client_secret']
  if (opts.tokenIssued) items.push('access_token (Bearer, expires_in 3600s)')
  if (opts.resourceFetched) items.push('resource data (fetched)')
  return items
}

export const START_STEP = 'start'

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
          'Client holds a client_id/client_secret issued by the auth server ahead of time. Nothing has been sent yet — press Next to fire the token request.',
        http: `# no request sent yet`,
        next: 'request-token',
      },
    ],
  },

  'request-token': {
    id: 'request-token',
    from: 'client',
    to: 'auth',
    outcomes: [
      {
        id: 'happy',
        label: 'Request sent',
        kind: 'happy',
        title: '1. Client requests token',
        detail:
          'Client authenticates itself directly (no user involved) and asks for an access token, optionally scoped.',
        http: `POST /oauth/token HTTP/1.1
Host: auth.example.com
Authorization: Basic base64(client_id:client_secret)
Content-Type: application/x-www-form-urlencoded

grant_type=client_credentials&scope=read:resource`,
        next: 'validate-client',
      },
    ],
  },

  'validate-client': {
    id: 'validate-client',
    from: 'auth',
    to: 'auth',
    outcomes: [
      {
        id: 'valid',
        label: 'Valid credentials',
        kind: 'happy',
        title: '2. Auth server validates client',
        detail:
          'Checks client_id/client_secret, confirms client is allowed the client_credentials grant and the requested scope.',
        http: `# server-side
lookup client by client_id
verify client_secret (hash compare)
check grant_type "client_credentials" allowed
check requested scope <= client's allowed scopes
=> OK, continue to token issuance`,
        next: 'issue-token',
      },
      {
        id: 'invalid_client',
        label: 'Invalid client_secret',
        kind: 'error',
        title: '2. Auth server validates client — failed',
        detail: 'client_secret does not match the hash on record for client_id. Request is rejected.',
        http: `HTTP/1.1 401 Unauthorized
Content-Type: application/json

{
  "error": "invalid_client",
  "error_description": "Client authentication failed"
}`,
        next: 'error-invalid-client',
      },
      {
        id: 'unauthorized_client',
        label: 'Grant not allowed for client',
        kind: 'error',
        title: '2. Auth server validates client — failed',
        detail: 'Credentials are valid, but this client is not registered for the client_credentials grant.',
        http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "unauthorized_client",
  "error_description": "This client is not authorized to use grant_type=client_credentials"
}`,
        next: 'error-unauthorized-client',
      },
    ],
  },

  'issue-token': {
    id: 'issue-token',
    from: 'auth',
    to: 'client',
    outcomes: [
      {
        id: 'success',
        label: 'Token issued',
        kind: 'happy',
        title: '3. Auth server issues access token',
        detail:
          'No refresh token here — RFC 6749 §4.4.3 says refresh tokens SHOULD NOT be issued for this grant, since the client can just re-authenticate.',
        http: `HTTP/1.1 200 OK
Content-Type: application/json

{
  "access_token": "eyJhbGciOi...",
  "token_type": "Bearer",
  "expires_in": 3600,
  "scope": "read:resource"
}`,
        next: 'call-resource',
      },
      {
        id: 'invalid_scope',
        label: 'Requested scope too broad',
        kind: 'error',
        title: '3. Auth server issues access token — failed',
        detail: 'Client is valid, but requested a scope it is not allowed to hold.',
        http: `HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid_scope",
  "error_description": "Requested scope exceeds client's allowed scopes"
}`,
        next: 'error-invalid-scope',
      },
      {
        id: 'server_error',
        label: 'Auth server error',
        kind: 'error',
        title: '3. Auth server issues access token — failed',
        detail: 'Everything checked out, but the auth server hit an internal error while signing the token.',
        http: `HTTP/1.1 500 Internal Server Error
Content-Type: application/json

{
  "error": "server_error",
  "error_description": "Unable to issue token, try again"
}`,
        next: 'error-server',
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
        title: '4. Client calls resource server',
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
        title: '5. Resource server validates token',
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
        title: '5. Resource server validates token — failed',
        detail: 'Signature is valid but exp has passed. Client needs to request a fresh token (no refresh token to use here).',
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
        title: '5. Resource server validates token — failed',
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
        title: '6. Resource returned',
        detail: 'Client gets the data. No user session, no redirect — pure machine-to-machine.',
        http: `HTTP/1.1 200 OK
Content-Type: application/json

{ "data": "..." }`,
        next: null,
      },
    ],
  },

  // --- terminal error steps ---
  'error-invalid-client': terminalError('error-invalid-client', 'auth', 'auth', 'Client authentication failed — flow ends here.'),
  'error-unauthorized-client': terminalError('error-unauthorized-client', 'auth', 'auth', 'Client not allowed this grant type — flow ends here.'),
  'error-invalid-scope': terminalError('error-invalid-scope', 'auth', 'client', 'Token issuance rejected — flow ends here.'),
  'error-server': terminalError('error-server', 'auth', 'client', 'Auth server failed — flow ends here.'),
  'error-expired': terminalError('error-expired', 'resource', 'client', 'Token rejected as expired — flow ends here.'),
  'error-scope': terminalError('error-scope', 'resource', 'client', 'Token rejected for insufficient scope — flow ends here.'),
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
