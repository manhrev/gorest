package server

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/balancechange/internal/dto"
	"github.com/manhrev/mkit/dto/response"
)

func (s *Server) registerTransactionRoutes(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-transaction",
		Method:        http.MethodPost,
		Path:          basePath,
		Summary:       "Parse a raw bank notification and store it",
		Tags:          []string{"Transactions"},
		DefaultStatus: http.StatusCreated,
		Security:      []map[string][]string{{apiKeySchemeName: {}}},
		Errors:        []int{http.StatusUnauthorized, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusServiceUnavailable},
	}, s.CreateTransaction)
}

func (s *Server) CreateTransaction(ctx context.Context, input *dto.CreateTransactionInput) (*response.Output[dto.Transaction], error) {
	t, err := s.txSvc.Create(ctx, input.Body.RawText, input.Body.CreatedAt, input.Body.Owner)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, t), nil
}
