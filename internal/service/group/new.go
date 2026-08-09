// Package group is the group service: business rules + translation between
// the DB shape (model.Group, from the repository) and the API shape
// (dto.GroupDTO, to the handler). Takes inline scalar params rather than
// dto structs, so it stays usable from any transport (HTTP via huma, or a
// future gRPC handler), not just the current huma handlers.
package group

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/manhrev/gorest/internal/code"
	"github.com/manhrev/gorest/internal/converter"
	"github.com/manhrev/gorest/internal/dto"
	grouprepo "github.com/manhrev/gorest/internal/repository/group"
	"github.com/manhrev/gorest/pkg/db/model"
	"github.com/manhrev/gorest/pkg/error/repoerr"
	"github.com/manhrev/gorest/pkg/error/serviceerr"
)

type Service struct {
	repo *grouprepo.Repository
}

func New(repo *grouprepo.Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, name, description string) (dto.GroupDTO, error) {
	id := uuid.New()
	now := time.Now().UTC()

	m, err := s.repo.Create(ctx, &model.GroupSetter{
		ID:          new(id.String()),
		Name:        &name,
		Description: &sql.Null[string]{V: description, Valid: description != ""},
		CreatedAt:   &now,
		UpdatedAt:   &now,
	})
	if err != nil {
		switch {
		case errors.Is(err, grouprepo.ErrNameExisted):
			return dto.GroupDTO{}, serviceerr.NewConflict(err).
				SetMessage("Group name already exists.").
				AddDetail("name", code.GroupNameExisted, "Group name already exists.")
		default:
			return dto.GroupDTO{}, serviceerr.NewInternal(err)
		}
	}

	return converter.GroupToDto(m), nil
}

func (s *Service) GetByID(ctx context.Context, id string) (dto.GroupDTO, error) {
	m, err := s.repo.FirstById(ctx, id)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return dto.GroupDTO{}, serviceerr.NewNotFound(err)
		}
		return dto.GroupDTO{}, serviceerr.NewInternal(err)
	}

	return converter.GroupToDto(m), nil
}

// UpdateByID applies a partial update: nil fields are left unchanged (same
// contract as the repository's model.GroupSetter).
func (s *Service) UpdateByID(ctx context.Context, id string, name, description *string) (dto.GroupDTO, error) {
	setter := &model.GroupSetter{Name: name}
	if description != nil {
		setter.Description = &sql.Null[string]{V: *description, Valid: *description != ""}
	}

	m, err := s.repo.UpdateById(ctx, id, setter)
	if err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return dto.GroupDTO{}, serviceerr.NewNotFound(err)
		}
		return dto.GroupDTO{}, serviceerr.NewInternal(err)
	}

	return converter.GroupToDto(m), nil
}

// FindByFilters returns the page-th page (1-indexed, limit rows per page) of
// groups matching the given filters, plus the total row count across all
// pages. search matches (case-insensitive, substring) against name;
// createdAtFrom/createdAtTo bound the created_at range; ids restricts to a
// specific set of group ids. Zero values (including zero time.Time, empty
// ids) mean "no filter" for that dimension.
func (s *Service) FindByFilters(
	ctx context.Context, search string, createdAtFrom, createdAtTo time.Time, ids []string, page, limit int,
) ([]dto.GroupDTO, int64, error) {
	rows, total, err := s.repo.FindByFilters(ctx, search, createdAtFrom, createdAtTo, ids, page, limit)
	if err != nil {
		return nil, 0, serviceerr.NewInternal(err)
	}

	return converter.GroupsToDtos(rows), total, nil
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

// AddUsersToGroups bulk-adds users to groups: userGroups is keyed by
// userId, each mapping to that user's own set of groupIds (not a cartesian
// product across users). Re-adding an existing pair is a no-op.
//
// map keys can't carry a struct `format:"uuid"` tag (JSON Schema has no
// format-on-keys), so every key/value is validated as a UUID here instead.
func (s *Service) AddUsersToGroups(ctx context.Context, userGroups map[string][]string) error {
	if len(userGroups) == 0 {
		return serviceerr.NewInvalidArgument(fmt.Errorf("userGroups must be non-empty"))
	}
	for userID, groupIDs := range userGroups {
		if _, err := uuid.Parse(userID); err != nil {
			return serviceerr.NewInvalidArgument(fmt.Errorf("userId %q is not a valid UUID", userID))
		}
		if len(groupIDs) == 0 {
			return serviceerr.NewInvalidArgument(fmt.Errorf("userId %s has no groupIds", userID))
		}
		for _, groupID := range groupIDs {
			if _, err := uuid.Parse(groupID); err != nil {
				return serviceerr.NewInvalidArgument(fmt.Errorf("groupId %q is not a valid UUID", groupID))
			}
		}
	}

	if err := s.repo.AddUsersToGroups(ctx, userGroups); err != nil {
		if errors.Is(err, repoerr.ErrNotFound) {
			return serviceerr.NewNotFound(err)
		}
		return serviceerr.NewInternal(err)
	}

	return nil
}
