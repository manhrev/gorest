package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/dto/response"
	"github.com/manhrev/gorest/pkg/error/serviceerr"
)

// registerOAuthRoutes registers the OAuth2 authorization-code endpoints
// relative to basePath (e.g. "/oauth"). See pkg/oauthserver's package doc
// and dto.OAuthAuthorizeInput for the scope this deliberately covers (no
// PKCE, no consent screen, no real browser redirect).
func (s *Server) registerOAuthRoutes(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "oauth-authorize",
		Method:      http.MethodGet,
		Path:        basePath + "/authorize",
		Summary:     "Issue an authorization code for an already-logged-in user",
		Tags:        []string{"OAuth"},
		Security:    []map[string][]string{{"bearerAuth": {}}},
	}, s.OAuthAuthorize)

	huma.Register(api, huma.Operation{
		OperationID: "oauth-token",
		Method:      http.MethodPost,
		Path:        basePath + "/token",
		Summary:     "Exchange an authorization code for an access+refresh token pair",
		Tags:        []string{"OAuth"},
	}, s.OAuthToken)
}

func (s *Server) OAuthAuthorize(ctx context.Context, input *dto.OAuthAuthorizeInput) (*dto.OAuthAuthorizeOutput, error) {
	if input.ResponseType != "code" {
		return nil, response.NewError(ctx, serviceerr.NewInvalidArgument(fmt.Errorf("unsupported response_type %q", input.ResponseType)).
			SetMessage(`response_type must be "code".`))
	}

	token, err := bearerToken(input.Authorization)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	claims, err := s.authSvc.ValidateAccessToken(ctx, token)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	if claims.IsDelegated() {
		// a client's own delegated token can't be used to mint further
		// delegations on the user's behalf — only the user's own token can.
		return nil, response.NewError(ctx, serviceerr.NewPermissionDenied(fmt.Errorf("delegated token cannot authorize another client")).
			SetMessage("A delegated access token cannot be used to authorize another client."))
	}

	redirectURL, err := s.oauthSvc.Authorize(ctx, input.ClientID, input.RedirectURI, input.Scope, input.State, claims.Subject)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	return &dto.OAuthAuthorizeOutput{Status: http.StatusFound, Location: redirectURL}, nil
}

func (s *Server) OAuthToken(ctx context.Context, input *dto.OAuthTokenInput) (*dto.OAuthTokenOutput, error) {
	if input.Body.GrantType != "authorization_code" {
		return nil, response.NewError(ctx, serviceerr.NewInvalidArgument(fmt.Errorf("unsupported grant_type %q", input.Body.GrantType)).
			SetMessage(`grant_type must be "authorization_code".`))
	}

	access, refresh, err := s.oauthSvc.Exchange(ctx, input.Body.ClientID, input.Body.ClientSecret, input.Body.Code, input.Body.RedirectURI)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}

	// re-verify the token we just issued to read its exp back out for
	// expires_in — avoids threading access-token-duration config through
	// Server just for this.
	claims, err := s.authSvc.ValidateAccessToken(ctx, access)
	if err != nil {
		return nil, response.NewError(ctx, serviceerr.NewInternal(err))
	}

	out := &dto.OAuthTokenOutput{}
	out.Body.AccessToken = access
	out.Body.RefreshToken = refresh
	out.Body.TokenType = "Bearer"
	out.Body.ExpiresIn = int(time.Until(claims.ExpiresAt.Time).Seconds())

	return out, nil
}
