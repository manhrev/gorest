package server

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/dto/response"
)

// registerUserRoutes registers the user resource's operations relative to
// basePath (e.g. "/users") — the caller (Run) decides the mount point,
// this only knows the resource-relative shape ("", "/{id}").
func (s *Server) registerUserRoutes(api huma.API, basePath string) {
	huma.Register(api, huma.Operation{
		OperationID: "create-user",
		Method:      http.MethodPost,
		Path:        basePath,
		Summary:     "Create a user",
		Tags:        []string{"Users"},
	}, s.CreateUser)

	huma.Register(api, huma.Operation{
		OperationID: "get-user",
		Method:      http.MethodGet,
		Path:        basePath + "/{id}",
		Summary:     "Get a user by ID",
		Tags:        []string{"Users"},
	}, s.GetUser)

	huma.Register(api, huma.Operation{
		OperationID: "update-user",
		Method:      http.MethodPatch,
		Path:        basePath + "/{id}",
		Summary:     "Update a user by ID",
		Tags:        []string{"Users"},
	}, s.UpdateUser)

	huma.Register(api, huma.Operation{
		OperationID:   "delete-user",
		Method:        http.MethodDelete,
		Path:          basePath + "/{id}",
		Summary:       "Delete a user by ID",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, s.DeleteUser)

	huma.Register(api, huma.Operation{
		OperationID: "list-users",
		Method:      http.MethodGet,
		Path:        basePath,
		Summary:     "List users",
		Tags:        []string{"Users"},
	}, s.ListUsers)

	huma.Register(api, huma.Operation{
		OperationID:   "add-users-to-groups",
		Method:        http.MethodPost,
		Path:          basePath + "/groups",
		Summary:       "Add users to groups",
		Tags:          []string{"Users"},
		DefaultStatus: http.StatusNoContent,
	}, s.AddUsersToGroups)

	huma.Register(api, huma.Operation{
		OperationID: "create-users-with-groups",
		Method:      http.MethodPost,
		Path:        basePath + "/with-groups",
		Summary:     "Create users and add each to its own groups (atomic)",
		Tags:        []string{"Users"},
	}, s.CreateUsersWithGroups)
}

func (s *Server) CreateUser(ctx context.Context, input *dto.CreateUserInput) (*response.Output[dto.UserDTO], error) {
	u, err := s.userSvc.Create(ctx, input.Body.Username, input.Body.Email, input.Body.FirstName, input.Body.LastName)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, u), nil
}

func (s *Server) GetUser(ctx context.Context, input *dto.GetUserInput) (*response.Output[dto.UserDTO], error) {
	u, err := s.userSvc.GetByID(ctx, input.ID, input.IsLoadGroups)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, u), nil
}

func (s *Server) UpdateUser(ctx context.Context, input *dto.UpdateUserInput) (*response.Output[dto.UserDTO], error) {
	u, err := s.userSvc.UpdateByID(ctx, input.ID, input.Body.FirstName, input.Body.LastName, input.Body.Email)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, u), nil
}

func (s *Server) DeleteUser(ctx context.Context, input *dto.DeleteUserInput) (*response.Output[response.EmptyData], error) {
	if err := s.userSvc.DeleteByID(ctx, input.ID); err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, response.EmptyData{}), nil
}

func (s *Server) ListUsers(ctx context.Context, input *dto.ListUsersInput) (*response.Output[response.PaginatedData[dto.UserDTO]], error) {
	list, total, err := s.userSvc.FindByFilters(ctx, input.Search, input.CreatedAtFrom, input.CreatedAtTo, input.IDs, input.Page, input.Limit, input.IsLoadGroups, input.SortBy, input.SortOrder)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, response.PaginatedData[dto.UserDTO]{
		Items:      list,
		Total:      int(total),
		Page:       input.Page,
		Limit:      input.Limit,
		TotalPages: (int(total) + input.Limit - 1) / input.Limit,
	}), nil
}

func (s *Server) AddUsersToGroups(ctx context.Context, input *dto.AddUsersToGroupsInput) (*response.Output[response.EmptyData], error) {
	if err := s.groupSvc.AddUsersToGroups(ctx, input.Body.UserGroups); err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, response.EmptyData{}), nil
}

func (s *Server) CreateUsersWithGroups(ctx context.Context, input *dto.CreateUsersWithGroupsInput) (*response.Output[[]dto.UserDTO], error) {
	users, err := s.userSvc.CreateUsersAndAddToGroups(ctx, input.Body.Users)
	if err != nil {
		return nil, response.NewError(ctx, err)
	}
	return response.NewOutput(ctx, users), nil
}
