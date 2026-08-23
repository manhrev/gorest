// Package server wires the app's dependencies (config, tracing, logger) and
// runs the HTTP server. See serve.go for the Run/setup logic.
package server

import (
	"github.com/manhrev/gorest/common/authservice"
	"github.com/manhrev/gorest/common/jwtmanager"
	"github.com/manhrev/gorest/common/oauthserver"
)

// Server holds the app's dependencies and exposes them as huma operation
// handlers (see auth_controller.go, oauth_controller.go, jwks_controller.go),
// registered directly by method value rather than wrapped in inline closures.
type Server struct {
	authSvc  *authservice.Service
	oauthSvc *oauthserver.Service
	jwtSvc   *jwtmanager.Service
	// audience is this service's own identity, checked against a bearer
	// token's aud claim at the OAuth endpoints that act as a resource
	// server (see oauth_controller.go) — same value issued tokens carry
	// as Issuer.
	audience string
}

func NewServer(authSvc *authservice.Service, oauthSvc *oauthserver.Service, jwtSvc *jwtmanager.Service, audience string) *Server {
	return &Server{authSvc: authSvc, oauthSvc: oauthSvc, jwtSvc: jwtSvc, audience: audience}
}
