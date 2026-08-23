package stub

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

type Verifier struct {
	users map[string]string // username -> password
}

func NewVerifier() *Verifier {
	return &Verifier{users: map[string]string{"alice": "hunter2"}}
}

func (v *Verifier) Verify(_ context.Context, username, password string) (string, error) {
	want, ok := v.users[username]
	if !ok || want != password {
		return "", errors.New("bad credentials")
	}

	return "user-" + username, nil
}

type UserLookup struct {
	roles map[string][]string // userID -> roles
}

func NewUserLookup() *UserLookup {
	return &UserLookup{roles: map[string][]string{"user-alice": {"admin"}}}
}

func (u *UserLookup) RolesByUserID(_ context.Context, userID string) ([]string, error) {
	return u.roles[userID], nil
}

type RefreshStore struct {
	mu      sync.Mutex
	records map[string]string // jti -> userID
}

func NewRefreshStore() *RefreshStore {
	return &RefreshStore{records: map[string]string{}}
}

func (s *RefreshStore) Save(_ context.Context, jti, userID string, _ time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[jti] = userID

	return nil
}

func (s *RefreshStore) Get(_ context.Context, jti string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	userID, ok := s.records[jti]
	if !ok {
		return "", errors.New("not found")
	}

	return userID, nil
}

func (s *RefreshStore) Delete(_ context.Context, jti string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.records, jti)

	return nil
}

type Blocklist struct {
	mu      sync.Mutex
	blocked map[string]bool
}

func NewBlocklist() *Blocklist {
	return &Blocklist{blocked: map[string]bool{}}
}

func (b *Blocklist) Block(_ context.Context, jti string, _ time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.blocked[jti] = true

	return nil
}

func (b *Blocklist) IsBlocked(_ context.Context, jti string) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.blocked[jti], nil
}
