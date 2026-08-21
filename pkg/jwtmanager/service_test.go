package jwtmanager

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func testService(t *testing.T) *Service {
	t.Helper()

	priv, err := LoadPrivateKey("testdata/priv.pem")
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	pub, err := LoadPublicKey("testdata/pub.pem")
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}

	s, err := New(priv, pub, Config{
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "gorest-test",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return s
}

func TestAccessTokenRoundtrip(t *testing.T) {
	s := testService(t)

	tok, err := s.GenerateAccessToken("user-1", []string{"admin"})
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := s.Verify(tok, TokenTypeAccess)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	if claims.Subject != "user-1" {
		t.Errorf("Subject = %q, want user-1", claims.Subject)
	}

	if len(claims.Roles) != 1 || claims.Roles[0] != "admin" {
		t.Errorf("Roles = %v, want [admin]", claims.Roles)
	}

	if claims.TokenType != TokenTypeAccess {
		t.Errorf("TokenType = %q, want %q", claims.TokenType, TokenTypeAccess)
	}
}

func TestRefreshTokenRejectedAsAccess(t *testing.T) {
	s := testService(t)

	tok, err := s.GenerateRefreshToken("user-1")
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	if _, err := s.Verify(tok, TokenTypeAccess); err == nil {
		t.Fatal("Verify(want=access) on refresh token: expected error, got nil")
	}
}

func TestWrongKeyRejected(t *testing.T) {
	s := testService(t)

	otherPriv, err := LoadPrivateKey("testdata/other_priv.pem")
	if err != nil {
		t.Fatalf("LoadPrivateKey: %v", err)
	}

	pub, err := LoadPublicKey("testdata/pub.pem") // mismatched pair
	if err != nil {
		t.Fatalf("LoadPublicKey: %v", err)
	}

	other, err := New(otherPriv, pub, Config{
		AccessTokenDuration:  time.Hour,
		RefreshTokenDuration: 24 * time.Hour,
		Issuer:               "gorest-test",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	tok, err := other.GenerateAccessToken("user-1", nil)
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	if _, err := s.Verify(tok, TokenTypeAccess); err == nil {
		t.Fatal("Verify with mismatched key: expected error, got nil")
	}
}

func TestAlgNoneRejected(t *testing.T) {
	s := testService(t)

	claims := &Claims{
		Identity:  Identity{Roles: []string{"admin"}},
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}

	tok, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign alg=none token: %v", err)
	}

	if _, err := s.Verify(tok, TokenTypeAccess); err == nil {
		t.Fatal("Verify on alg=none token: expected error, got nil")
	}
}

func TestExpiredTokenRejected(t *testing.T) {
	s := testService(t)

	tok, err := s.generate("user-1", nil, TokenTypeAccess, -time.Minute)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if _, err := s.Verify(tok, TokenTypeAccess); err == nil {
		t.Fatal("Verify on expired token: expected error, got nil")
	}
}
