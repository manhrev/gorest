package dto

import "github.com/manhrev/gorest/pkg/jwtmanager"

// JWKSInput is a bare GET, no params.
type JWKSInput struct{}

// JWKSOutput is deliberately flat (no response.Output[T] envelope), same as
// OAuthTokenOutput — RFC 7517 mandates the exact {"keys":[...]} shape so any
// standard JWKS client can parse it.
type JWKSOutput struct {
	Body jwtmanager.JWKS
}
