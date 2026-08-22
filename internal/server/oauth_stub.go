package server

// ponytail: placeholder oauthserver.ClientStore/AuthorizationCodeStore
// adapters, same spirit as auth_stub.go — one hardcoded client, in-process
// code store (lost on restart, not shared across instances). Real
// follow-up: a real client registry (DB table) once clients are more than
// "the one other service I run", and a shared store (redis) once this runs
// on more than one instance.

import (
	"context"
	"errors"
	"sync"

	"github.com/manhrev/gorest/pkg/oauthserver"
)

type stubClientStore struct {
	clients map[string]oauthserver.Client
}

func newStubClientStore() *stubClientStore {
	return &stubClientStore{clients: map[string]oauthserver.Client{
		"internal-service": {
			ID:           "internal-service",
			Secret:       "dev-secret",
			RedirectURIs: []string{"http://localhost:9090/callback"},
			Scopes:       []string{"read:user_password", "read:user_email"},
		},
	}}
}

func (s *stubClientStore) Get(_ context.Context, clientID string) (oauthserver.Client, error) {
	c, ok := s.clients[clientID]
	if !ok {
		return oauthserver.Client{}, errors.New("unknown client")
	}

	return c, nil
}

type memCodeStore struct {
	mu    sync.Mutex
	codes map[string]oauthserver.AuthorizationCode
}

func newMemCodeStore() *memCodeStore {
	return &memCodeStore{codes: map[string]oauthserver.AuthorizationCode{}}
}

func (s *memCodeStore) Save(_ context.Context, code string, ac oauthserver.AuthorizationCode) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = ac

	return nil
}

func (s *memCodeStore) Consume(_ context.Context, code string) (oauthserver.AuthorizationCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ac, ok := s.codes[code]
	if !ok {
		return oauthserver.AuthorizationCode{}, errors.New("unknown or already-used code")
	}
	delete(s.codes, code)

	return ac, nil
}
