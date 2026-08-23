package server

// ponytail: placeholder authservice.CredentialVerifier/UserLookup/
// RefreshTokenStore/AccessTokenBlocklist adapters, wired into serve.go so
// the auth endpoints have something to run against. The User model has no
// password field or roles concept yet, and the stores are in-process maps
// (lost on restart, not shared across instances). Real follow-up: a
// password column + hashing on User, a real roles source, and redis-backed
// stores (pkg/cache/redis.SetStruct/GetStruct fit both interfaces
// directly).

import (
	"context"
	"errors"
	"sync"
	"time"
)

type stubVerifier struct {
	users map[string]string // username -> password
}

func newStubVerifier() *stubVerifier {
	return &stubVerifier{users: map[string]string{"alice": "hunter2"}}
}

func (v *stubVerifier) Verify(_ context.Context, username, password string) (string, error) {
	want, ok := v.users[username]
	if !ok || want != password {
		return "", errors.New("bad credentials")
	}

	return "user-" + username, nil
}

type stubUserLookup struct {
	roles map[string][]string // userID -> roles
}

func newStubUserLookup() *stubUserLookup {
	return &stubUserLookup{roles: map[string][]string{"user-alice": {"admin"}}}
}

func (u *stubUserLookup) RolesByUserID(_ context.Context, userID string) ([]string, error) {
	return u.roles[userID], nil
}

type memRefreshStore struct {
	mu      sync.Mutex
	records map[string]string // jti -> userID
}

func newMemRefreshStore() *memRefreshStore {
	return &memRefreshStore{records: map[string]string{}}
}

func (s *memRefreshStore) Save(_ context.Context, jti, userID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[jti] = userID

	return nil
}

func (s *memRefreshStore) Get(_ context.Context, jti string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, ok := s.records[jti]
	if !ok {
		return "", errors.New("not found")
	}

	return userID, nil
}

func (s *memRefreshStore) Delete(_ context.Context, jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, jti)

	return nil
}

type memBlocklist struct {
	mu      sync.Mutex
	blocked map[string]bool
}

func newMemBlocklist() *memBlocklist {
	return &memBlocklist{blocked: map[string]bool{}}
}

func (b *memBlocklist) Block(_ context.Context, jti string, _ time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked[jti] = true

	return nil
}

func (b *memBlocklist) IsBlocked(_ context.Context, jti string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.blocked[jti], nil
}
