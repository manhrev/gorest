package jwtmanager

import (
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Identity holds the caller-visible claims. Subject (user id) lives in
// jwt.RegisteredClaims.Subject, that's what it's for.
//
// A token is either a direct user token (ClientID empty, Roles is the
// user's own RBAC roles) or a delegated/OAuth token issued to a client
// acting on the user's behalf (ClientID + Scope, per RFC 9068 "JWT Profile
// for OAuth 2.0 Access Tokens" §2.2's client_id and scope claims — Scope
// is what that client was granted, capped by and usually narrower than the
// user's own Roles). Roles and Scope are never both populated on the same
// token; use Claims.Permissions to read whichever applies.
type Identity struct {
	Roles     []string          `json:"roles,omitempty"`
	Domain    string            `json:"dom,omitempty"`
	DeviceID  string            `json:"did,omitempty"`
	SessionID string            `json:"sid,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	ClientID  string            `json:"client_id,omitempty" doc:"OAuth client this token was issued to (RFC 9068), empty for a direct user login token"`
	Scope     string            `json:"scope,omitempty" doc:"Space-separated OAuth scope (RFC 9068), only set on a delegated (ClientID != \"\") token"`
}

type Claims struct {
	Identity
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

// IsDelegated reports whether this token was issued to an OAuth client
// acting on the user's behalf, rather than directly to the user.
func (c *Claims) IsDelegated() bool {
	return c.ClientID != ""
}

// Permissions returns the permission list to authorize against: Scope
// (split on whitespace) for a delegated token, Roles for a direct user
// token. Resource servers should call this instead of reading Roles/Scope
// directly, so a check written once works for both token kinds.
func (c *Claims) Permissions() []string {
	if c.IsDelegated() {
		return strings.Fields(c.Scope)
	}

	return c.Roles
}
