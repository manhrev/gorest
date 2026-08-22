package server

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/internal/dto"
)

// registerJWKSRoute registers the standard JWKS discovery endpoint — public,
// no Security field, since its whole point is letting a resource server
// verify tokens without calling back into this one.
func (s *Server) registerJWKSRoute(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "jwks",
		Method:      http.MethodGet,
		Path:        "/.well-known/jwks.json",
		Summary:     "Public keys for verifying this server's access/refresh tokens",
		Tags:        []string{"OAuth"},
	}, s.JWKS)
}

func (s *Server) JWKS(ctx context.Context, input *dto.JWKSInput) (*dto.JWKSOutput, error) {
	out := &dto.JWKSOutput{}
	out.Body = s.jwtSvc.JWKS()

	return out, nil
}
