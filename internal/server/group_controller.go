package server

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/dto/response"
)

// registerGroupRoutes registers the group resource's operations relative to
// basePath (e.g. "/groups") — the caller (Run) decides the mount point,
// this only knows the resource-relative shape ("", "/{id}").
func (s *Server) registerGroupRoutes(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "create-group",
		Method:      http.MethodPost,
		Path:        basePath,
		Summary:     "Create a group",
		Tags:        []string{"Groups"},
	}, s.CreateGroup)

	huma.Register(api, huma.Operation{
		OperationID: "get-group",
		Method:      http.MethodGet,
		Path:        basePath + "/{id}",
		Summary:     "Get a group by ID",
		Tags:        []string{"Groups"},
	}, s.GetGroup)

	huma.Register(api, huma.Operation{
		OperationID: "update-group",
		Method:      http.MethodPatch,
		Path:        basePath + "/{id}",
		Summary:     "Update a group by ID",
		Tags:        []string{"Groups"},
	}, s.UpdateGroup)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-group",
		Method:        http.MethodDelete,
		Path:          basePath + "/{id}",
		Summary:       "Delete a group by ID",
		Tags:          []string{"Groups"},
		DefaultStatus: http.StatusNoContent,
	}, s.DeleteGroup)

	huma.Register(api, huma.Operation{
		OperationID: "list-groups",
		Method:      http.MethodGet,
		Path:        basePath,
		Summary:     "List groups",
		Tags:        []string{"Groups"},
	}, s.ListGroups)
}

func (s *Server) CreateGroup(ctx context.Context, input *dto.CreateGroupInput) (*response.Output[dto.GroupDTO], error) {
	g, err := s.groupSvc.Create(ctx, input.Body.Name, input.Body.Description)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, g), nil
}

func (s *Server) GetGroup(ctx context.Context, input *dto.GetGroupInput) (*response.Output[dto.GroupDTO], error) {
	g, err := s.groupSvc.GetByID(ctx, input.ID)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, g), nil
}

func (s *Server) UpdateGroup(ctx context.Context, input *dto.UpdateGroupInput) (*response.Output[dto.GroupDTO], error) {
	g, err := s.groupSvc.UpdateByID(ctx, input.ID, input.Body.Name, input.Body.Description)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, g), nil
}

func (s *Server) DeleteGroup(ctx context.Context, input *dto.DeleteGroupInput) (*response.Output[response.EmptyData], error) {
	if err := s.groupSvc.DeleteByID(ctx, input.ID); err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, response.EmptyData{}), nil
}

func (s *Server) ListGroups(ctx context.Context, input *dto.ListGroupsInput) (*response.Output[response.PaginatedData[dto.GroupDTO]], error) {
	list, total, err := s.groupSvc.FindByFilters(ctx, input.Search, input.CreatedAtFrom, input.CreatedAtTo, input.IDs, input.Page, input.Limit, input.SortBy, input.SortOrder)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, response.PaginatedData[dto.GroupDTO]{
		Items:      list,
		Total:      int(total),
		Page:       input.Page,
		Limit:      input.Limit,
		TotalPages: (int(total) + input.Limit - 1) / input.Limit,
	}), nil
}
