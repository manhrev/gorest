package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/dto/response"
	"github.com/manhrev/gorest/pkg/error/serviceerr"
)

// registerAuthRoutes registers the auth resource's operations relative to
// basePath (e.g. "/auth").
func (s *Server) registerAuthRoutes(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "login",
		Method:      http.MethodPost,
		Path:        basePath + "/login",
		Summary:     "Login with username/password, issue access+refresh tokens",
		Tags:        []string{"Auth"},
	}, s.Login)

	huma.Register(api, huma.Operation{
		OperationID:   "logout",
		Method:        http.MethodPost,
		Path:          basePath + "/logout",
		Summary:       "Revoke the current access token (and refresh token, if sent)",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusNoContent,
		Security:      []map[string][]string{{"bearerAuth": {}}},
	}, s.Logout)

	huma.Register(api, huma.Operation{
		OperationID: "refresh-token",
		Method:      http.MethodPost,
		Path:        basePath + "/refresh",
		Summary:     "Rotate a refresh token for a new access+refresh pair",
		Tags:        []string{"Auth"},
	}, s.Refresh)

	huma.Register(api, huma.Operation{
		OperationID: "check-auth",
		Method:      http.MethodGet,
		Path:        basePath + "/check",
		Summary:     "Validate an access token, return its identity",
		Tags:        []string{"Auth"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, s.CheckAuth)
}

// bearerToken strips the "Bearer " prefix off an Authorization header value.
// The scheme name is case-insensitive per RFC 7235 §2.1.
func bearerToken(header string) (string, error) {
	const prefix = "Bearer "

	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", serviceerr.NewUnauthenticated(fmt.Errorf("missing or malformed Authorization header")).
			SetMessage("Missing or malformed Authorization header.")
	}

	return header[len(prefix):], nil
}

func (s *Server) Login(ctx context.Context, input *dto.LoginInput) (*response.Output[dto.TokenPair], error) {
	access, refresh, err := s.authSvc.Login(ctx, input.Body.Username, input.Body.Password)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	return response.NewOutput(ctx, dto.TokenPair{AccessToken: access, RefreshToken: refresh}), nil
}

func (s *Server) Logout(ctx context.Context, input *dto.LogoutInput) (*response.Output[response.EmptyData], error) {
	token, err := bearerToken(input.Authorization)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	if err := s.authSvc.RevokeAccessToken(ctx, token); err != nil {
		return nil, response.NewError(ctx, err)
	}

	if input.Body.RefreshToken != "" {
		// best-effort: the access token is already revoked at this point,
		// don't fail the whole logout over a refresh token that's already
		// unknown/expired/malformed.
		_ = s.authSvc.RevokeRefreshToken(ctx, input.Body.RefreshToken)
	}

	return response.NewOutput(ctx, response.EmptyData{}), nil
}

func (s *Server) Refresh(ctx context.Context, input *dto.RefreshInput) (*response.Output[dto.TokenPair], error) {
	access, refresh, err := s.authSvc.Refresh(ctx, input.Body.RefreshToken)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	return response.NewOutput(ctx, dto.TokenPair{AccessToken: access, RefreshToken: refresh}), nil
}

func (s *Server) CheckAuth(ctx context.Context, input *dto.CheckAuthInput) (*response.Output[dto.AuthIdentity], error) {
	token, err := bearerToken(input.Authorization)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	claims, err := s.authSvc.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	return response.NewOutput(ctx, dto.AuthIdentity{
		UserID:   claims.Subject,
		Roles:    claims.Roles,
		ClientID: claims.ClientID,
		Scope:    claims.Scope,
	}), nil
}
