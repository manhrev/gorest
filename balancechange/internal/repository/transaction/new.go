// Package transaction is the transaction repository, backed by bob (see
// internal/db/model for the generated table bindings) against a
// pgxpool-backed connection. Speaks model.* types only, never dto.
package transaction

import (
	"context"
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	bobpgx "github.com/stephenafamo/bob/drivers/pgx"

	"github.com/manhrev/gorest/balancechange/internal/db/model"
)

// ErrDuplicated: same raw_text at the same created_at already stored (see
// transactions_dedupe_key in migrations). It's an expression index, not a
// constraint, so bob's dberror doesn't generate a matcher for it.
var ErrDuplicated = errors.New("transaction duplicated")

const dedupeIndex = "transactions_dedupe_key"

type Repository struct {
	db bob.Executor
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: bobpgx.NewPool(pool)}
}

func (r *Repository) Create(ctx context.Context, setter *model.TransactionSetter) (*model.Transaction, error) {
	m, err := model.Transactions.Insert(setter).One(ctx, r.db)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok &&
			pgErr.Code == pgerrcode.UniqueViolation && pgErr.ConstraintName == dedupeIndex {
			return nil, ErrDuplicated
		}
		return nil, err
	}

	return m, nil
}
