package oauthserver

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/manhrev/gorest/pkg/authservice"
	"github.com/manhrev/gorest/pkg/jwtmanager"
)

// --- fakes (authservice deps, same shape as authservice's own tests) ---

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, username, password string) (string, error) {
	if username == "alice" && password == "hunter2" {
		return "user-alice", nil
	}

	return "", errors.New("bad credentials")
}

type fakeUserLookup struct{}

func (fakeUserLookup) RolesByUserID(_ context.Context, userID string) ([]string, error) {
	return []string{"admin"}, nil
}

type fakeRefreshStore struct {
	mu      sync.Mutex
	records map[string]string
}

func newFakeRefreshStore() *fakeRefreshStore { return &fakeRefreshStore{records: map[string]string{}} }

func (f *fakeRefreshStore) Save(_ context.Context, jti, userID string, _ time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records[jti] = userID

	return nil
}

func (f *fakeRefreshStore) Get(_ context.Context, jti string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	userID, ok := f.records[jti]
	if !ok {
		return "", errors.New("not found")
	}

	return userID, nil
}

func (f *fakeRefreshStore) Delete(_ context.Context, jti string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.records, jti)

	return nil
}

type fakeBlocklist struct{}

func (fakeBlocklist) Block(_ context.Context, jti string, _ time.Time) error { return nil }
func (fakeBlocklist) IsBlocked(_ context.Context, jti string) (bool, error)  { return false, nil }

// --- fakes (oauthserver deps) ---

type fakeClientStore struct {
	clients map[string]Client
}

func (f *fakeClientStore) Get(_ context.Context, clientID string) (Client, error) {
	c, ok := f.clients[clientID]
	if !ok {
		return Client{}, errors.New("unknown client")
	}

	return c, nil
}

type fakeCodeStore struct {
	mu    sync.Mutex
	codes map[string]AuthorizationCode
}

func newFakeCodeStore() *fakeCodeStore { return &fakeCodeStore{codes: map[string]AuthorizationCode{}} }

func (f *fakeCodeStore) Save(_ context.Context, code string, ac AuthorizationCode) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.codes[code] = ac

	return nil
}

func (f *fakeCodeStore) Consume(_ context.Context, code string) (AuthorizationCode, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	ac, ok := f.codes[code]
	if !ok {
		return AuthorizationCode{}, errors.New("unknown code")
	}
	delete(f.codes, code)

	return ac, nil
}

// --- setup ---

const testRedirectURI = "http://localhost:9090/callback"

func testService(t *testing.T) *Service {
	t.Helper()

	priv, err := jwtmanager.LoadPrivateKey("../jwtmanager/testdata/priv.pem")
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	pub, err := jwtmanager.LoadPublicKey("../jwtmanager/testdata/pub.pem")
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}

	jwtSvc, err := jwtmanager.New(priv, pub, jwtmanager.Config{
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "oauthserver-test",
	})
	if err != nil {
		t.Fatalf("jwtmanager.New: %v", err)
	}

	authSvc := authservice.New(jwtSvc, fakeVerifier{}, fakeUserLookup{}, newFakeRefreshStore(), fakeBlocklist{})

	clients := &fakeClientStore{clients: map[string]Client{
		"internal-service": {
			ID:           "internal-service",
			Secret:       "dev-secret",
			RedirectURIs: []string{testRedirectURI},
			Scopes:       []string{"read:resource"},
		},
	}}

	return New(authSvc, clients, newFakeCodeStore())
}

// extractCode pulls the "code" query param out of a "url?code=...&state=..."
// redirect target built by Authorize.
func extractCode(t *testing.T, redirectURL string) string {
	t.Helper()

	u, err := url.Parse(redirectURL)
	if err != nil {
		t.Fatalf("parse redirect URL %q: %v", redirectURL, err)
	}

	code := u.Query().Get("code")
	if code == "" {
		t.Fatalf("redirect URL %q has no code param", redirectURL)
	}

	return code
}

// --- tests ---

func TestAuthorizeExchangeRoundtrip(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	redirectURL, err := s.Authorize(ctx, "internal-service", testRedirectURI, "read:resource", "xyz", "user-alice")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	code := extractCode(t, redirectURL)

	access, refresh, err := s.Exchange(ctx, "internal-service", "dev-secret", code, testRedirectURI)
	if err != nil {
		t.Fatalf("Exchange: %v", err)
	}

	if access == "" || refresh == "" {
		t.Fatal("Exchange returned empty tokens")
	}

	// the issued token must carry the granted scope, not the user's own
	// (broader) roles — user-alice's fakeUserLookup roles are ["admin"],
	// but only "read:resource" was requested/granted.
	claims, err := s.auth.ValidateAccessToken(ctx, access)
	if err != nil {
		t.Fatalf("ValidateAccessToken: %v", err)
	}

	if !claims.IsDelegated() {
		t.Error("IsDelegated() = false, want true for an OAuth-issued token")
	}

	if claims.ClientID != "internal-service" {
		t.Errorf("ClientID = %q, want internal-service", claims.ClientID)
	}

	if got := claims.Permissions(); len(got) != 1 || got[0] != "read:resource" {
		t.Errorf("Permissions() = %v, want [read:resource]", got)
	}

	if len(claims.Roles) != 0 {
		t.Errorf("Roles = %v, want empty — delegated token must not carry the user's own roles", claims.Roles)
	}
}

func TestExchangeRejectsReusedCode(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	redirectURL, err := s.Authorize(ctx, "internal-service", testRedirectURI, "read:resource", "xyz", "user-alice")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	code := extractCode(t, redirectURL)

	if _, _, err := s.Exchange(ctx, "internal-service", "dev-secret", code, testRedirectURI); err != nil {
		t.Fatalf("first Exchange: %v", err)
	}

	if _, _, err := s.Exchange(ctx, "internal-service", "dev-secret", code, testRedirectURI); err == nil {
		t.Fatal("second Exchange with reused code: expected error, got nil")
	}
}

func TestExchangeRejectsWrongSecret(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	redirectURL, err := s.Authorize(ctx, "internal-service", testRedirectURI, "read:resource", "xyz", "user-alice")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	code := extractCode(t, redirectURL)

	if _, _, err := s.Exchange(ctx, "internal-service", "wrong-secret", code, testRedirectURI); err == nil {
		t.Fatal("Exchange with wrong secret: expected error, got nil")
	}
}

func TestExchangeRejectsRedirectURIMismatch(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	redirectURL, err := s.Authorize(ctx, "internal-service", testRedirectURI, "read:resource", "xyz", "user-alice")
	if err != nil {
		t.Fatalf("Authorize: %v", err)
	}

	code := extractCode(t, redirectURL)

	if _, _, err := s.Exchange(ctx, "internal-service", "dev-secret", code, "http://evil.example/callback"); err == nil {
		t.Fatal("Exchange with mismatched redirect_uri: expected error, got nil")
	}
}

func TestExchangeRejectsExpiredCode(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	// insert an already-expired code directly, bypassing Authorize's TTL
	codes := s.codes.(*fakeCodeStore)
	if err := codes.Save(ctx, "expired-code", AuthorizationCode{
		ClientID:    "internal-service",
		UserID:      "user-alice",
		RedirectURI: testRedirectURI,
		Scope:       "read:resource",
		ExpiresAt:   time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if _, _, err := s.Exchange(ctx, "internal-service", "dev-secret", "expired-code", testRedirectURI); err == nil {
		t.Fatal("Exchange with expired code: expected error, got nil")
	}
}

func TestAuthorizeRejectsUnknownClient(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	if _, err := s.Authorize(ctx, "no-such-client", testRedirectURI, "read:resource", "xyz", "user-alice"); err == nil {
		t.Fatal("Authorize with unknown client_id: expected error, got nil")
	}
}

func TestAuthorizeRejectsUnregisteredRedirectURI(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	if _, err := s.Authorize(ctx, "internal-service", "http://evil.example/callback", "read:resource", "xyz", "user-alice"); err == nil {
		t.Fatal("Authorize with unregistered redirect_uri: expected error, got nil")
	}
}

func TestAuthorizeRejectsDisallowedScope(t *testing.T) {
	s := testService(t)
	ctx := context.Background()

	if _, err := s.Authorize(ctx, "internal-service", testRedirectURI, "write:resource", "xyz", "user-alice"); err == nil {
		t.Fatal("Authorize with disallowed scope: expected error, got nil")
	}
}
