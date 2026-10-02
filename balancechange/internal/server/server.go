// Package server wires the app's dependencies (config, tracing, logger,
// postgres, gemini) and runs the HTTP server. See serve.go for Run.
package server

import (
	txservice "github.com/manhrev/gorest/balancechange/internal/service/transaction"
)

// Server holds the app's dependencies and exposes them as huma operation
// handlers (see transaction_controller.go).
type Server struct {
	txSvc *txservice.Service
}

func NewServer(txSvc *txservice.Service) *Server {
	return &Server{txSvc: txSvc}
}
