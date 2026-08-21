package authservice

import (
	"context"
	"fmt"
	"time"

	"github.com/manhrev/gorest/pkg/error/serviceerr"
	"github.com/manhrev/gorest/pkg/jwtmanager"
)

// CredentialVerifier checks username/password, returns the user id on success.
type CredentialVerifier interface {
	Verify(ctx context.Context, username, password string) (userID string, err error)
}

// UserLookup fetches current roles for a user id, called on login and on
// every refresh, so a role change takes effect without re-login.
type UserLookup interface {
	RolesByUserID(ctx context.Context, userID string) ([]string, error)
}

// RefreshTokenStore tracks issued refresh tokens by jti, so a refresh token
// can be rotated (old jti deleted, new jti saved) or revoked.
type RefreshTokenStore interface {
	Save(ctx context.Context, jti, userID string, expiresAt time.Time) error
	// Get returns the stored userID for jti, or an error if absent/expired/revoked.
	Get(ctx context.Context, jti string) (userID string, err error)
	Delete(ctx context.Context, jti string) error
}

// AccessTokenBlocklist records revoked-but-not-yet-expired access tokens by
// jti. Access tokens are otherwise stateless, this is the only way to kill
// one before its exp.
type AccessTokenBlocklist interface {
	// Block marks jti revoked until expiresAt (store can drop it after,
	// the token would be rejected on exp anyway).
	Block(ctx context.Context, jti string, expiresAt time.Time) error
	IsBlocked(ctx context.Context, jti string) (bool, error)
}

type Service struct {
	jwt       *jwtmanager.Service
	verifier  CredentialVerifier
	users     UserLookup
	store     RefreshTokenStore
	blocklist AccessTokenBlocklist
}

func New(jwt *jwtmanager.Service, verifier CredentialVerifier, users UserLookup, store RefreshTokenStore, blocklist AccessTokenBlocklist) *Service {
	return &Service{jwt: jwt, verifier: verifier, users: users, store: store, blocklist: blocklist}
}

func (s *Service) Login(ctx context.Context, username, password string) (access, refresh string, err error) {
	userID, err := s.verifier.Verify(ctx, username, password)
	if err != nil {
		return "", "", serviceerr.NewUnauthenticated(err).SetMessage("Invalid username or password.")
	}

	roles, err := s.users.RolesByUserID(ctx, userID)
	if err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	return s.issue(ctx, userID, roles)
}

// ValidateAccessToken verifies an access token, returns its claims. Also
// checks the blocklist, so a revoked token is rejected before its exp.
func (s *Service) ValidateAccessToken(ctx context.Context, token string) (*jwtmanager.Claims, error) {
	claims, err := s.jwt.Verify(token, jwtmanager.TokenTypeAccess)
	if err != nil {
		return nil, serviceerr.NewUnauthenticated(err).SetMessage("Invalid or expired access token.")
	}

	blocked, err := s.blocklist.IsBlocked(ctx, claims.ID)
	if err != nil {
		return nil, serviceerr.NewInternal(err)
	}

	if blocked {
		return nil, serviceerr.NewUnauthenticated(fmt.Errorf("access token revoked")).
			SetMessage("Access token has been revoked.")
	}

	return claims, nil
}

// RevokeAccessToken blocks an access token before its natural expiry (e.g.
// on logout). Idempotent-ish: revoking an already-expired token just fails
// Verify, nothing left to block.
func (s *Service) RevokeAccessToken(ctx context.Context, token string) error {
	claims, err := s.jwt.Verify(token, jwtmanager.TokenTypeAccess)
	if err != nil {
		return serviceerr.NewUnauthenticated(err).SetMessage("Invalid or expired access token.")
	}

	if err := s.blocklist.Block(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		return serviceerr.NewInternal(err)
	}

	return nil
}

// RevokeRefreshToken invalidates a refresh token before it's used (e.g.
// logout). No-op-ish on an already-used/unknown jti: store.Delete on a
// missing key isn't an error.
func (s *Service) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	claims, err := s.jwt.Verify(refreshToken, jwtmanager.TokenTypeRefresh)
	if err != nil {
		return serviceerr.NewUnauthenticated(err).SetMessage("Invalid or expired refresh token.")
	}

	if err := s.store.Delete(ctx, claims.ID); err != nil {
		return serviceerr.NewInternal(err)
	}

	return nil
}

// Refresh rotates a refresh token: the old jti is invalidated even if
// issuing the replacement fails partway, so a token can never be reused.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (access, refresh string, err error) {
	claims, err := s.jwt.Verify(refreshToken, jwtmanager.TokenTypeRefresh)
	if err != nil {
		return "", "", serviceerr.NewUnauthenticated(err).SetMessage("Invalid or expired refresh token.")
	}

	storedUserID, err := s.store.Get(ctx, claims.ID)
	if err != nil {
		return "", "", serviceerr.NewUnauthenticated(err).SetMessage("Refresh token not recognized.")
	}

	if storedUserID != claims.Subject {
		return "", "", serviceerr.NewUnauthenticated(fmt.Errorf("refresh token subject mismatch")).
			SetMessage("Refresh token not recognized.")
	}

	if err := s.store.Delete(ctx, claims.ID); err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	roles, err := s.users.RolesByUserID(ctx, claims.Subject)
	if err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	return s.issue(ctx, claims.Subject, roles)
}

// issue generates a fresh access+refresh pair and records the refresh
// token's jti in the store.
func (s *Service) issue(ctx context.Context, userID string, roles []string) (access, refresh string, err error) {
	access, err = s.jwt.GenerateAccessToken(userID, roles)
	if err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	refresh, err = s.jwt.GenerateRefreshToken(userID)
	if err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	claims, err := s.jwt.Verify(refresh, jwtmanager.TokenTypeRefresh)
	if err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	if err := s.store.Save(ctx, claims.ID, userID, claims.ExpiresAt.Time); err != nil {
		return "", "", serviceerr.NewInternal(err)
	}

	return access, refresh, nil
}
