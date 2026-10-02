package stub

// ponytail: placeholder oauthserver.ClientStore/AuthorizationCodeStore
// adapters, same spirit as auth.go — one hardcoded client, in-process
// code store (lost on restart, not shared across instances). Real
// follow-up: a real client registry (DB table) once clients are more than
// "the one other service I run", and a shared store (redis) once this runs
// on more than one instance.

import (
	"context"
	"errors"
	"sync"

	"github.com/manhrev/mkit/oauthserver"
)

type ClientStore struct {
	clients map[string]oauthserver.Client
}

func NewClientStore() *ClientStore {
	return &ClientStore{clients: map[string]oauthserver.Client{
		"internal-service": {
			ID:           "internal-service",
			Secret:       "dev-secret",
			RedirectURIs: []string{"http://localhost:9090/callback", "http://localhost:5173/callback"},
			Scopes:       []string{"read:user_password", "read:user_email"},
			// first-party trusted, auto-approved (RequireConsent false).
		},
		"partner-app": {
			ID:             "partner-app",
			Secret:         "dev-secret",
			RedirectURIs:   []string{"http://localhost:9091/callback", "http://localhost:5173/callback"},
			Scopes:         []string{"read:user_password", "read:user_email"},
			RequireConsent: true,
		},
	}}
}

func (s *ClientStore) Get(_ context.Context, clientID string) (oauthserver.Client, error) {
	c, ok := s.clients[clientID]
	if !ok {
		return oauthserver.Client{}, errors.New("unknown client")
	}

	return c, nil
}

type CodeStore struct {
	mu    sync.Mutex
	codes map[string]oauthserver.AuthorizationCode
}

func NewCodeStore() *CodeStore {
	return &CodeStore{codes: map[string]oauthserver.AuthorizationCode{}}
}

func (s *CodeStore) Save(_ context.Context, code string, ac oauthserver.AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = ac

	return nil
}

func (s *CodeStore) Consume(_ context.Context, code string) (oauthserver.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.codes[code]
	if !ok {
		return oauthserver.AuthorizationCode{}, errors.New("unknown or already-used code")
	}
	delete(s.codes, code)

	return ac, nil
}

type ConsentStore struct {
	mu      sync.Mutex
	tickets map[string]oauthserver.ConsentTicket
}

func NewConsentStore() *ConsentStore {
	return &ConsentStore{tickets: map[string]oauthserver.ConsentTicket{}}
}

func (s *ConsentStore) Save(_ context.Context, consentID string, t oauthserver.ConsentTicket) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tickets[consentID] = t

	return nil
}

func (s *ConsentStore) Consume(_ context.Context, consentID string) (oauthserver.ConsentTicket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.tickets[consentID]
	if !ok {
		return oauthserver.ConsentTicket{}, errors.New("unknown or already-decided consent")
	}
	delete(s.tickets, consentID)

	return t, nil
}
