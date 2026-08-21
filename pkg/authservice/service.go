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

type Service struct {
	jwt      *jwtmanager.Service
	verifier CredentialVerifier
	users    UserLookup
	store    RefreshTokenStore
}

func New(jwt *jwtmanager.Service, verifier CredentialVerifier, users UserLookup, store RefreshTokenStore) *Service {
	return &Service{jwt: jwt, verifier: verifier, users: users, store: store}
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

// ValidateAccessToken verifies an access token and returns its claims. Pure
// check, no I/O, so no ctx.
func (s *Service) ValidateAccessToken(token string) (*jwtmanager.Claims, error) {
	claims, err := s.jwt.Verify(token, jwtmanager.TokenTypeAccess)
	if err != nil {
		return nil, serviceerr.NewUnauthenticated(err).SetMessage("Invalid or expired access token.")
	}

	return claims, nil
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
