// Package server wires the app's dependencies (config, tracing, logger,
// postgres) and runs the HTTP server. See serve.go for the Run/setup logic.
package server

import (
	groupservice "github.com/manhrev/gorest/internal/service/group"
	userservice "github.com/manhrev/gorest/internal/service/user"
	"github.com/manhrev/gorest/pkg/authservice"
	"github.com/manhrev/gorest/pkg/oauthserver"
)

// Server holds the app's dependencies and exposes them as huma operation
// handlers (see user_controller.go, group_controller.go), registered
// directly by method value rather than wrapped in inline closures.
type Server struct {
	userSvc  *userservice.Service
	groupSvc *groupservice.Service
	authSvc  *authservice.Service
	oauthSvc *oauthserver.Service
}

func NewServer(userSvc *userservice.Service, groupSvc *groupservice.Service, authSvc *authservice.Service, oauthSvc *oauthserver.Service) *Server {
	return &Server{userSvc: userSvc, groupSvc: groupSvc, authSvc: authSvc, oauthSvc: oauthSvc}
}
