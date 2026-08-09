// Package user is the user repository, backed by bob (see pkg/db/model for
// the generated table bindings) against a pgxpool-backed connection.
//
// This package speaks the DB's native model.User/model.UserSetter types
// only — no dto.UserDTO here. Translating between the DB shape and the API
// shape, plus any business rules, belongs to the service layer above this,
// not the repository.
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	bobpgx "github.com/stephenafamo/bob/drivers/pgx"

	"github.com/manhrev/gorest/pkg/db/dberror"
	"github.com/manhrev/gorest/pkg/db/model"
	"github.com/manhrev/gorest/pkg/error/repoerr"
)

// Unique-constraint-violation sentinels for Create, mapped from Postgres'
// unique_violation by constraint name (see migrations/*_init.up.sql for
// users_username_key/users_email_key). repoerr stays generic on purpose
// (see its doc comment) — a constraint this specific belongs here.
var (
	ErrUsernameExisted = fmt.Errorf("username existed")
	ErrEmailExisted    = fmt.Errorf("email existed")
)

// db is bob.Executor, not the concrete bobpgx.Pool, so WithExecutor can
// rebind it to a transaction (bobpgx.Tx also satisfies bob.Executor).
type Repository struct {
	db bob.Executor
}

// New wraps pool as a bob.Executor. bobpgx.Pool wraps *pgxpool.Pool
// directly (implements bob.Executor's QueryContext/ExecContext itself) —
// no database/sql bridge needed.
func New(pool *pgxpool.Pool) *Repository {
	return &Repository{db: bobpgx.NewPool(pool)}
}

// WithExecutor returns a copy of the repo bound to exec (e.g. a tx from
// pkg/db/tx) instead of the pool — every call on the copy participates in
// that transaction.
func (r *Repository) WithExecutor(exec bob.Executor) *Repository {
	return &Repository{db: exec}
}

func (r *Repository) Create(ctx context.Context, setter *model.UserSetter) (*model.User, error) {
	m, err := model.Users.Insert(setter).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repoerr.ErrNotFound
		}

		switch {
		case dberror.UserErrors.ErrUniqueUsersUsernameKey.Is(err):
			return nil, ErrUsernameExisted
		case dberror.UserErrors.ErrUniqueUsersEmailKey.Is(err):
			return nil, ErrEmailExisted
		}

		return nil, err
	}

	return m, nil
}

// FirstById fetches the user by id. loadGroups additionally preloads the
// user's groups (m.R.UserGroups, each with .R.Group set) — off by default
// since most callers don't need it. model.FindUser takes no extra mods, so
// this goes through Users.Query directly (same sm.Where(ID.EQ(...)) it
// uses internally) to be able to attach the loader mod.
func (r *Repository) FirstById(ctx context.Context, id string, loadGroups bool) (*model.User, error) {
	mods := []bob.Mod[*dialect.SelectQuery]{sm.Where(model.Users.Columns.ID.EQ(psql.Arg(id)))}
	if loadGroups {
		mods = append(
			mods,
			model.SelectThenLoad.User.UserGroups(model.Preload.UserGroup.Group()),
		)
	}

	m, err := model.Users.Query(mods...).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repoerr.ErrNotFound
		}
		return nil, err
	}

	return m, nil
}

// UpdateById applies a partial update: only the fields set on setter are
// changed, everything else is left as-is (setter's pointer fields make this
// a true partial update, not a fetch-then-overwrite).
func (r *Repository) UpdateById(ctx context.Context, id string, setter *model.UserSetter) (*model.User, error) {
	updated, err := model.Users.Update(
		setter.UpdateMod(),
		um.Where(model.Users.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repoerr.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

// FindByFilters lists users. search matches (case-insensitive, substring)
// against username or email; createdAtFrom/createdAtTo bound the created_at
// range; ids restricts to a specific set of user ids. Zero values (including
// zero time.Time, empty ids) mean "no filter" for that dimension.
// page/limit are 1-indexed/page-size; FindByFilters returns the page's rows
// plus the total row count across all pages (ignoring limit/offset) so the
// caller can compute total pages. loadGroups preloads each row's groups
// (see FirstById's loadGroups doc).
func (r *Repository) FindByFilters(
	ctx context.Context, search string, createdAtFrom, createdAtTo time.Time, ids []string, page, limit int, loadGroups bool,
) (model.UserSlice, int64, error) {
	var mods []bob.Mod[*dialect.SelectQuery]

	if search != "" {
		pattern := "%" + search + "%"
		mods = append(mods, sm.Where(psql.Or(
			model.Users.Columns.Username.ILike(psql.Arg(pattern)),
			model.Users.Columns.Email.ILike(psql.Arg(pattern)),
		)))
	}
	if !createdAtFrom.IsZero() {
		mods = append(mods, sm.Where(model.Users.Columns.CreatedAt.GTE(psql.Arg(createdAtFrom))))
	}
	if !createdAtTo.IsZero() {
		mods = append(mods, sm.Where(model.Users.Columns.CreatedAt.LTE(psql.Arg(createdAtTo))))
	}
	if len(ids) > 0 {
		idExprs := make([]bob.Expression, len(ids))
		for i, id := range ids {
			idExprs[i] = psql.Arg(id)
		}
		mods = append(mods, sm.Where(model.Users.Columns.ID.In(idExprs...)))
	}

	total, err := model.Users.Query(mods...).Count(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}

	mods = append(mods, sm.Limit(limit), sm.Offset((page-1)*limit))

	// Batches the user_groups + groups fetch into one extra query each
	// (not one per row) — see model.SelectThenLoad's doc. Select then load for additional query, Preload for single join query
	if loadGroups {
		mods = append(
			mods,
			model.SelectThenLoad.User.UserGroups(
				model.Preload.UserGroup.Group(),
			),
		)
	}

	rows, err := model.Users.Query(mods...).All(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, repoerr.ErrNotFound
		}
		return nil, 0, err
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].CreatedAt.Before(rows[j].CreatedAt) })

	return rows, total, nil
}

func (r *Repository) DeleteById(ctx context.Context, id string) error {
	rowsAffected, err := model.Users.Delete(
		dm.Where(model.Users.Columns.ID.EQ(psql.Arg(id))),
	).Exec(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repoerr.ErrNotFound
		}
		return err
	}
	if rowsAffected == 0 {
		return repoerr.ErrNotFound
	}

	return nil
}
