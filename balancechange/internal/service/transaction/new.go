// Package transaction is the transaction service: raw notification text ->
// Gemini -> validated model.TransactionSetter -> repository. Takes inline
// scalar params rather than dto structs, so it stays transport-agnostic.
package transaction

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/google/uuid"

	"github.com/manhrev/gorest/balancechange/internal/code"
	"github.com/manhrev/gorest/balancechange/internal/converter"
	"github.com/manhrev/gorest/balancechange/internal/db/model"
	"github.com/manhrev/gorest/balancechange/internal/dto"
	"github.com/manhrev/gorest/balancechange/internal/enum"
	"github.com/manhrev/gorest/balancechange/internal/gemini"
	txrepo "github.com/manhrev/gorest/balancechange/internal/repository/transaction"
	"github.com/manhrev/mkit/error/serviceerr"
)

type Service struct {
	repo   *txrepo.Repository
	gemini *gemini.Client
}

func New(repo *txrepo.Repository, gemini *gemini.Client) *Service {
	return &Service{repo: repo, gemini: gemini}
}

// Create parses rawText with Gemini and stores the result. owner, when
// non-empty, overrides Gemini's guess.
func (s *Service) Create(ctx context.Context, rawText string, createdAt time.Time, owner string) (dto.Transaction, error) {
	parsed, err := s.gemini.Parse(ctx, rawText)
	if err != nil {
		return dto.Transaction{}, serviceerr.NewUnavailable(err).SetMessage("Transaction parser unavailable.")
	}

	setter, err := buildSetter(parsed, rawText, createdAt, owner)
	if err != nil {
		return dto.Transaction{}, err
	}

	m, err := s.repo.Create(ctx, setter)
	if err != nil {
		if errors.Is(err, txrepo.ErrDuplicated) {
			return dto.Transaction{}, serviceerr.NewConflict(err).
				SetMessage("Transaction already stored.").
				AddDetail("raw_text", code.TransactionDuplicated, "Same raw_text at same created_at already stored.")
		}
		return dto.Transaction{}, serviceerr.NewInternal(err)
	}

	return converter.TransactionToDto(m), nil
}

// buildSetter re-validates Gemini's output: the response schema constrains
// it, but it's still model output and the DB is the trust boundary.
func buildSetter(p gemini.Parsed, rawText string, createdAt time.Time, owner string) (*model.TransactionSetter, error) {
	if !p.IsTransaction {
		return nil, serviceerr.NewInvalidArgument(errors.New("not a balance change")).
			SetHTTPStatus(http.StatusUnprocessableEntity).
			SetMessage("Text is not a balance change.").
			AddDetail("raw_text", code.TransactionNotParseable, "Text is not a balance change.")
	}

	if owner != "" {
		p.Owner = owner
	}
	if p.Amount <= 0 || !slices.Contains(enum.Types, p.Type) ||
		!slices.Contains(enum.Categories, p.Category) || !slices.Contains(enum.Owners, p.Owner) {
		return nil, serviceerr.NewUnavailable(fmt.Errorf("invalid gemini output: %+v", p)).
			SetMessage("Transaction parser returned invalid output.")
	}

	createdAt = createdAt.UTC()
	return &model.TransactionSetter{
		ID:        new(uuid.NewString()),
		Note:      &p.Note,
		Amount:    &p.Amount,
		Type:      &p.Type,
		Category:  &p.Category,
		Owner:     &p.Owner,
		RawText:   &rawText,
		CreatedAt: &createdAt,
	}, nil
}
