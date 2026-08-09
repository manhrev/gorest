// Package user is the user service: business rules + translation between
// the DB shape (model.User, from the repository) and the API shape
// (dto.UserDTO, to the handler). Takes inline scalar params rather than
// dto structs, so it stays usable from any transport (HTTP via huma, or a
// future gRPC handler), not just the current huma handlers.
package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/stephenafamo/bob"

	"github.com/manhrev/gorest/internal/code"
	"github.com/manhrev/gorest/internal/converter"
	"github.com/manhrev/gorest/internal/dto"
	grouprepo "github.com/manhrev/gorest/internal/repository/group"
	userrepo "github.com/manhrev/gorest/internal/repository/user"
	"github.com/manhrev/gorest/pkg/db/model"
	"github.com/manhrev/gorest/pkg/dto/request"
	"github.com/manhrev/gorest/pkg/error/repoerr"
	"github.com/manhrev/gorest/pkg/error/serviceerr"
	"github.com/manhrev/gorest/pkg/txrunner"
)

// groupRepo/txRunner are only used by CreateUsersAndAddToGroups, the one
// operation that spans both resources — everything else here stays
// single-repo.
type Service struct {
	repo      *userrepo.Repository
	groupRepo *grouprepo.Repository
	txRunner  *txrunner.Runner
}

func New(repo *userrepo.Repository, groupRepo *grouprepo.Repository, txRunner *txrunner.Runner) *Service {
	return &Service{repo: repo, groupRepo: groupRepo, txRunner: txRunner}
}

func (s *Service) Create(ctx context.Context, username, email, firstName, lastName string, meta *dto.UserMetaBox) (dto.UserDTO, error) {
	m, err := s.repo.Create(ctx, newUserSetter(username, email, firstName, lastName, meta))
	if err != nil {
		return dto.UserDTO{}, mapCreateErr(err)
	}

	return converter.UserToDto(m), nil
}

func newUserSetter(username, email, firstName, lastName string, meta *dto.UserMetaBox) *model.UserSetter {
	id := uuid.New()
	now := time.Now().UTC()
	return &model.UserSetter{
		ID:        new(id.String()),
		Username:  &username,
		Email:     &email,
		FirstName: &firstName,
		LastName:  &lastName,
		CreatedAt: &now,
		UpdatedAt: &now,
		Meta:      converter.UserMetaToSetterField(meta),
	}
}

// mapCreateErr translates a userrepo.Create error into a serviceerr, shared
// by Create and CreateUsersAndAddToGroups so both surface the same
// conflict details for a duplicate username/email.
func mapCreateErr(err error) error {
	switch {
	case errors.Is(err, userrepo.ErrUsernameExisted):
		return serviceerr.NewConflict(err).
			SetMessage("Username already exists.").
			AddDetail("username", code.UserUsernameExisted, "Username already exists.")
	case errors.Is(err, userrepo.ErrEmailExisted):
		return serviceerr.NewConflict(err).
			SetMessage("Email already exists.").
			AddDetail("email", code.UserEmailExisted, "Email already exists.")
	default:
		return serviceerr.NewInternal(err)
	}
}

func (s *Service) GetByID(ctx context.Context, id string, loadGroups bool) (dto.UserDTO, error) {
	m, err := s.repo.FirstById(ctx, id, loadGroups)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return dto.UserDTO{}, serviceerr.NewNotFound(err)
		}
		return dto.UserDTO{}, serviceerr.NewInternal(err)
	}

	return converter.UserToDto(m), nil
}

// UpdateByID applies a partial update: nil fields are left unchanged (same
// contract as the repository's model.UserSetter). meta, when non-nil,
// replaces any existing meta wholesale (no partial-merge of its contents).
func (s *Service) UpdateByID(ctx context.Context, id string, firstName, lastName, email *string, meta *dto.UserMetaBox) (dto.UserDTO, error) {
	m, err := s.repo.UpdateById(ctx, id, &model.UserSetter{
		FirstName: firstName,
		LastName:  lastName,
		Email:     email,
		Meta:      converter.UserMetaToSetterField(meta),
	})
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return dto.UserDTO{}, serviceerr.NewNotFound(err)
		}
		return dto.UserDTO{}, serviceerr.NewInternal(err)
	}

	return converter.UserToDto(m), nil
}

// FindByFilters returns the page-th page (1-indexed, limit rows per page) of users
// matching the given filters, plus the total row count across all pages.
// search matches (case-insensitive, substring) against username or email;
// createdAtFrom/createdAtTo bound the created_at range; ids restricts to a
// specific set of user ids. Zero values (including zero time.Time, empty
// ids) mean "no filter" for that dimension. Every id in ids must be a
// well-formed UUID — this is the validation boundary, so a malformed one is
// rejected outright rather than silently excluded.
func (s *Service) FindByFilters(
	ctx context.Context, search string, createdAtFrom, createdAtTo time.Time, ids []string, page, limit int, loadGroups bool,
	sortBy string, sortOrder request.SortOrder,
) ([]dto.UserDTO, int64, error) {
	rows, total, err := s.repo.FindByFilters(ctx, search, createdAtFrom, createdAtTo, ids, page, limit, loadGroups, sortBy, sortOrder)
	if err != nil {
		return nil, 0, serviceerr.NewInternal(err)
	}

	return converter.UsersToDtos(rows), total, nil
}

func (s *Service) DeleteByID(ctx context.Context, id string) error {
	if err := s.repo.DeleteById(ctx, id); err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return serviceerr.NewNotFound(err)
		}
		return serviceerr.NewInternal(err)
	}

	return nil
}

// CreateUsersAndAddToGroups creates every user in users and adds each to
// its own GroupIds, all in one transaction: any failure — a bad group id,
// a duplicate username/email — rolls back every user in the request, not
// just the one that failed.
func (s *Service) CreateUsersAndAddToGroups(ctx context.Context, users []dto.CreateUserWithGroups) ([]dto.UserDTO, error) {
	if len(users) == 0 {
		return nil, serviceerr.NewInvalidArgument(fmt.Errorf("users must be non-empty"))
	}

	out := make([]dto.UserDTO, 0, len(users))
	err := s.txRunner.Run(ctx, func(ctx context.Context, exec bob.Executor) error {
		userRepoTx := s.repo.WithExecutor(exec)
		groupRepoTx := s.groupRepo.WithExecutor(exec)

		userGroups := make(map[string][]string)
		for _, u := range users {
			m, err := userRepoTx.Create(ctx, newUserSetter(u.Username, u.Email, u.FirstName, u.LastName, u.Meta))
			if err != nil {
				return mapCreateErr(err)
			}
			out = append(out, converter.UserToDto(m))
			if len(u.GroupIds) > 0 {
				userGroups[m.ID] = u.GroupIds
			}
		}

		if len(userGroups) > 0 {
			if err := groupRepoTx.AddUsersToGroups(ctx, userGroups); err != nil {
				if errors.Is(err, repoerr.ErrNotFound) {
					return serviceerr.NewNotFound(err)
				}
				return serviceerr.NewInternal(err)
			}
		}

		return nil
	})
	if err != nil {
		if svcErr, ok := errors.AsType[*serviceerr.Error](err); ok {
			return nil, svcErr // already mapped inside the tx func
		}
		return nil, serviceerr.NewInternal(err) // tx begin/commit itself failed
	}

	return out, nil
}
