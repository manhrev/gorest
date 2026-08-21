package jwtmanager

import (
	"crypto/ed25519"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Service struct {
	privateKey           ed25519.PrivateKey
	publicKey            ed25519.PublicKey
	accessTokenDuration  time.Duration
	refreshTokenDuration time.Duration
	issuer               string
}

func New(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, cfg Config) (*Service, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("jwtmanager: invalid private key length %d, want %d", len(privateKey), ed25519.PrivateKeySize)
	}

	if len(publicKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("jwtmanager: invalid public key length %d, want %d", len(publicKey), ed25519.PublicKeySize)
	}

	return &Service{
		privateKey:           privateKey,
		publicKey:            publicKey,
		accessTokenDuration:  cfg.AccessTokenDuration,
		refreshTokenDuration: cfg.RefreshTokenDuration,
		issuer:               cfg.Issuer,
	}, nil
}

// ClaimOption mutates an access token's claims before signing, e.g.
// WithDomain, WithDeviceID.
type ClaimOption func(*Claims)

func WithDomain(domain string) ClaimOption {
	return func(c *Claims) { c.Domain = domain }
}

func WithDeviceID(deviceID string) ClaimOption {
	return func(c *Claims) { c.DeviceID = deviceID }
}

func WithSessionID(sessionID string) ClaimOption {
	return func(c *Claims) { c.SessionID = sessionID }
}

func WithMetadata(metadata map[string]string) ClaimOption {
	return func(c *Claims) { c.Metadata = metadata }
}

func (s *Service) generate(subject string, roles []string, tokenType string, duration time.Duration, opts ...ClaimOption) (string, error) {
	now := time.Now()

	claims := &Claims{
		Identity:  Identity{Roles: roles},
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   subject,
			Issuer:    s.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			ID:        uuid.NewString(),
		},
	}

	for _, opt := range opts {
		opt(claims)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)

	signed, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}

	return signed, nil
}

func (s *Service) GenerateAccessToken(subject string, roles []string, opts ...ClaimOption) (string, error) {
	return s.generate(subject, roles, TokenTypeAccess, s.accessTokenDuration, opts...)
}

// GenerateRefreshToken carries only subject + type, least-privilege: smaller
// blast radius if leaked.
func (s *Service) GenerateRefreshToken(subject string) (string, error) {
	return s.generate(subject, nil, TokenTypeRefresh, s.refreshTokenDuration)
}

func (s *Service) Verify(tokenString string, wantType string) (*Claims, error) {
	claims := &Claims{}

	_, err := jwt.ParseWithClaims(tokenString, claims, func(*jwt.Token) (any, error) {
		return s.publicKey, nil
	}, jwt.WithValidMethods([]string{"EdDSA"}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenSignatureInvalid) {
			return nil, fmt.Errorf("invalid token signature: %w", err)
		}

		return nil, fmt.Errorf("parse token: %w", err)
	}

	if claims.TokenType != wantType {
		return nil, fmt.Errorf("unexpected token type: want %s got %s", wantType, claims.TokenType)
	}

	return claims, nil
}
