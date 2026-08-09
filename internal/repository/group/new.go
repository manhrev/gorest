// Package group is the group repository, backed by bob (see pkg/db/model
// for the generated table bindings) against a pgxpool-backed connection.
//
// This package speaks the DB's native model.Group/model.GroupSetter types
// only — no dto.GroupDTO here. Translating between the DB shape and the API
// shape, plus any business rules, belongs to the service layer above this,
// not the repository.
package group

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	bobpgx "github.com/stephenafamo/bob/drivers/pgx"

	"github.com/manhrev/gorest/pkg/db/dberror"
	"github.com/manhrev/gorest/pkg/db/model"
	"github.com/manhrev/gorest/pkg/dto/request"
	"github.com/manhrev/gorest/pkg/error/repoerr"
)

// groupSortColumns whitelists the columns FindByFilters can sort by, keyed
// by the same strings validated on the wire via ListGroupsInput's SortBy
// enum tag. Looking them up here too (not just trusting the enum
// validation) is defense in depth — a raw column name never flows into SQL
// from a request.
var groupSortColumns = map[string]any{
	"name":      model.Groups.Columns.Name,
	"createdAt": model.Groups.Columns.CreatedAt,
}

// ErrNameExisted is the unique-constraint-violation sentinel for Create,
// mapped from Postgres' unique_violation via dberror.GroupErrors (see
// migrations/*_groups.up.sql for groups_name_key). repoerr stays generic on
// purpose (see its doc comment) — a constraint this specific belongs here.
var ErrNameExisted = fmt.Errorf("name existed")

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

func (r *Repository) Create(ctx context.Context, setter *model.GroupSetter) (*model.Group, error) {
	m, err := model.Groups.Insert(setter).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repoerr.ErrNotFound
		}

		if dberror.GroupErrors.ErrUniqueGroupsNameKey.Is(err) {
			return nil, ErrNameExisted
		}

		return nil, err
	}

	return m, nil
}

func (r *Repository) FirstById(ctx context.Context, id string) (*model.Group, error) {
	m, err := model.FindGroup(ctx, r.db, id)
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
func (r *Repository) UpdateById(ctx context.Context, id string, setter *model.GroupSetter) (*model.Group, error) {
	updated, err := model.Groups.Update(
		setter.UpdateMod(),
		um.Where(model.Groups.Columns.ID.EQ(psql.Arg(id))),
	).One(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, repoerr.ErrNotFound
		}
		return nil, err
	}

	return updated, nil
}

// FindByFilters lists groups. search matches (case-insensitive, substring)
// against name; createdAtFrom/createdAtTo bound the created_at range; ids
// restricts to a specific set of group ids. Zero values (including zero
// time.Time, empty ids) mean "no filter" for that dimension. page/limit are
// 1-indexed/page-size; FindByFilters returns the page's rows plus the total
// row count across all pages (ignoring limit/offset) so the caller can
// compute total pages. sortBy/sortOrder control the ORDER BY, applied in SQL
// before limit/offset — an unrecognized sortBy (shouldn't happen,
// ListGroupsInput's enum tag already rejects it) falls back to created_at
// rather than erroring.
func (r *Repository) FindByFilters(
	ctx context.Context, search string, createdAtFrom, createdAtTo time.Time, ids []string, page, limit int,
	sortBy string, sortOrder request.SortOrder,
) (model.GroupSlice, int64, error) {
	var mods []bob.Mod[*dialect.SelectQuery]

	if search != "" {
		pattern := "%" + search + "%"
		mods = append(mods, sm.Where(model.Groups.Columns.Name.ILike(psql.Arg(pattern))))
	}
	if !createdAtFrom.IsZero() {
		mods = append(mods, sm.Where(model.Groups.Columns.CreatedAt.GTE(psql.Arg(createdAtFrom))))
	}
	if !createdAtTo.IsZero() {
		mods = append(mods, sm.Where(model.Groups.Columns.CreatedAt.LTE(psql.Arg(createdAtTo))))
	}
	if len(ids) > 0 {
		idExprs := make([]bob.Expression, len(ids))
		for i, id := range ids {
			idExprs[i] = psql.Arg(id)
		}
		mods = append(mods, sm.Where(model.Groups.Columns.ID.In(idExprs...)))
	}

	total, err := model.Groups.Query(mods...).Count(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}

	col, ok := groupSortColumns[sortBy]
	if !ok {
		col = model.Groups.Columns.CreatedAt
	}
	order := sm.OrderBy(col)
	if sortOrder == request.SortOrderDesc {
		mods = append(mods, order.Desc())
	} else {
		mods = append(mods, order.Asc())
	}

	mods = append(mods, sm.Limit(limit), sm.Offset((page-1)*limit))

	rows, err := model.Groups.Query(mods...).All(ctx, r.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, repoerr.ErrNotFound
		}
		return nil, 0, err
	}

	return rows, total, nil
}

func (r *Repository) DeleteById(ctx context.Context, id string) error {
	rowsAffected, err := model.Groups.Delete(
		dm.Where(model.Groups.Columns.ID.EQ(psql.Arg(id))),
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

// AddUsersToGroups bulk-inserts one user_groups row per (userId, groupId)
// pair — userGroups is keyed by userId, each mapping to that user's own set
// of groupIds. ON CONFLICT DO NOTHING makes re-adding an existing pair a
// no-op, so the call is safely repeatable. A userId or groupId that doesn't
// exist surfaces as a foreign-key violation, mapped to repoerr.ErrNotFound.
func (r *Repository) AddUsersToGroups(ctx context.Context, userGroups map[string][]string) error {
	now := time.Now().UTC()
	mods := make([]bob.Mod[*dialect.InsertQuery], 0, len(userGroups)+1)
	for userID, groupIDs := range userGroups {
		for _, groupID := range groupIDs {
			mods = append(mods, &model.UserGroupSetter{UserID: &userID, GroupID: &groupID, JoinedAt: &now})
		}
	}
	mods = append(mods, im.OnConflict(
		model.UserGroups.Columns.UserID.Name(), model.UserGroups.Columns.GroupID.Name(),
	).DoNothing())

	_, err := model.UserGroups.Insert(mods...).All(ctx, r.db)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == pgerrcode.ForeignKeyViolation {
			return repoerr.ErrNotFound
		}
		return err
	}

	return nil
}
