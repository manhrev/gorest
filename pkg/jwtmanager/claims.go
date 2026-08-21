package jwtmanager

import "github.com/golang-jwt/jwt/v5"

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Identity holds the caller-visible claims. Subject (user id) lives in
// jwt.RegisteredClaims.Subject, that's what it's for.
type Identity struct {
	Roles     []string          `json:"roles"`
	Domain    string            `json:"dom,omitempty"`
	DeviceID  string            `json:"did,omitempty"`
	SessionID string            `json:"sid,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

type Claims struct {
	Identity
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}
