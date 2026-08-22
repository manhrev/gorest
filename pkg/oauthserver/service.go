// Package oauthserver implements the OAuth2 Authorization Code grant
// (RFC 6749 §4.1) for internal, confidential clients: no PKCE (clients hold
// a client_secret), no consent screen (first-party trusted clients), no
// dynamic client registration (ClientStore is expected to be a small static
// list). Token issuance itself is delegated to authservice.Service.
package oauthserver

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/manhrev/gorest/pkg/authservice"
	"github.com/manhrev/gorest/pkg/error/serviceerr"
)

// codeTTL is how long an authorization code is valid before exchange —
// short-lived by design, it's meant to be exchanged within seconds of the
// redirect, not stored or reused.
const codeTTL = 2 * time.Minute

// Client is a registered OAuth2 client (another confidential backend
// service, not a browser/SPA — it holds Secret and calls Exchange itself).
type Client struct {
	ID           string
	Secret       string
	RedirectURIs []string
	Scopes       []string // scopes this client is allowed to request
}

type ClientStore interface {
	Get(ctx context.Context, clientID string) (Client, error)
}

// AuthorizationCode is what AuthorizationCodeStore persists between
// Authorize (issues it) and Exchange (consumes it).
type AuthorizationCode struct {
	ClientID    string
	UserID      string
	RedirectURI string
	Scope       string
	ExpiresAt   time.Time
}

// AuthorizationCodeStore holds short-lived, single-use authorization codes.
type AuthorizationCodeStore interface {
	Save(ctx context.Context, code string, ac AuthorizationCode) error
	// Consume atomically gets and deletes code — single use, so a replay
	// (or a second /token call for the same code) is rejected.
	Consume(ctx context.Context, code string) (AuthorizationCode, error)
}

type Service struct {
	auth    *authservice.Service
	clients ClientStore
	codes   AuthorizationCodeStore
}

func New(auth *authservice.Service, clients ClientStore, codes AuthorizationCodeStore) *Service {
	return &Service{auth: auth, clients: clients, codes: codes}
}

// Authorize validates client_id/redirect_uri/scope for an already-
// authenticated user (userID from a prior ValidateAccessToken call) and
// returns the redirect URL carrying a fresh authorization code + state.
// scope is capped against client.Scopes here, at grant time — the code
// (and the token it's later exchanged for) carries only the requested
// scope, never the user's own roles.
//
// Note: this package doesn't itself establish who the user is — no cookie
// session, no browser login page. The caller (the HTTP controller) is
// expected to have already verified the user via a bearer access token.
// That means this only works when whatever calls Authorize already holds
// that token (a script/fetch call), not a real browser top-level redirect
// (which can't carry a custom header) — see the controller for the full
// caveat.
func (s *Service) Authorize(ctx context.Context, clientID, redirectURI, scope, state, userID string) (redirectURL string, err error) {
	client, err := s.clients.Get(ctx, clientID)
	if err != nil {
		return "", serviceerr.NewInvalidArgument(err).SetMessage("Unknown client_id.")
	}

	if !slices.Contains(client.RedirectURIs, redirectURI) {
		// Deliberately NOT redirecting on this error — redirecting to an
		// unregistered URI is exactly the open-redirect this check exists
		// to prevent.
		return "", serviceerr.NewInvalidArgument(fmt.Errorf("redirect_uri not registered for client")).
			SetMessage("redirect_uri does not match a registered URI for this client.")
	}

	for sc := range strings.FieldsSeq(scope) {
		if !slices.Contains(client.Scopes, sc) {
			return "", serviceerr.NewInvalidArgument(fmt.Errorf("scope %q not allowed for client", sc)).
				SetMessage("Requested scope exceeds what this client is allowed.")
		}
	}

	code, err := newCode()
	if err != nil {
		return "", serviceerr.NewInternal(err)
	}

	if err := s.codes.Save(ctx, code, AuthorizationCode{
		ClientID:    clientID,
		UserID:      userID,
		RedirectURI: redirectURI,
		Scope:       scope,
		ExpiresAt:   time.Now().Add(codeTTL),
	}); err != nil {
		return "", serviceerr.NewInternal(err)
	}

	return redirectURI + "?code=" + code + "&state=" + state, nil
}

// Exchange validates client credentials + the code, consumes it (single
// use), and issues an access+refresh pair for the code's user.
func (s *Service) Exchange(ctx context.Context, clientID, clientSecret, code, redirectURI string) (access, refresh string, err error) {
	client, err := s.clients.Get(ctx, clientID)
	if err != nil || client.Secret != clientSecret {
		return "", "", serviceerr.NewUnauthenticated(fmt.Errorf("invalid client credentials")).
			SetMessage("Invalid client_id or client_secret.")
	}

	ac, err := s.codes.Consume(ctx, code)
	if err != nil {
		return "", "", serviceerr.NewUnauthenticated(fmt.Errorf("invalid or already-used code: %w", err)).
			SetMessage("Invalid, expired, or already-used authorization code.")
	}

	if ac.ClientID != clientID || ac.RedirectURI != redirectURI {
		return "", "", serviceerr.NewUnauthenticated(fmt.Errorf("code was not issued to this client/redirect_uri")).
			SetMessage("Authorization code does not match this client/redirect_uri.")
	}

	if time.Now().After(ac.ExpiresAt) {
		return "", "", serviceerr.NewUnauthenticated(fmt.Errorf("authorization code expired")).
			SetMessage("Authorization code expired.")
	}

	return s.auth.IssueForClient(ctx, ac.UserID, ac.ClientID, ac.Scope)
}

// newCode generates an opaque, single-use authorization code — a random
// secret, not an identifier, so crypto/rand rather than uuid.
func newCode() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate authorization code: %w", err)
	}

	return hex.EncodeToString(b), nil
}

