// Package dto holds request/response types for the API's operations.
// Response bodies use pkg/dto.Output[T] directly (a common generic type),
// so this package only needs to define the data shapes (User,
// ListUsersData) and the request Input types, no per-operation Output
// wrapper structs.
package dto

import (
	"time"

	"github.com/manhrev/gorest/pkg/dto/request"
)

// User is the canonical user representation returned by the API.
// Groups is only populated when the request opted in (e.g. GetUserInput's
// LoadGroups) — nil/omitted otherwise, not an empty-vs-absent distinction
// worth modeling further.
type User struct {
	ID        string       `json:"id" format:"uuid" doc:"User ID (UUID)"`
	Username  string       `json:"username"`
	Email     string       `json:"email"`
	FirstName string       `json:"firstName"`
	LastName  string       `json:"lastName"`
	CreatedAt time.Time    `json:"createdAt"`
	UpdatedAt time.Time    `json:"updatedAt"`
	Groups    []Group      `json:"groups,omitempty" doc:"User's groups, only present when requested"`
	Meta      *UserMetaBox `json:"meta,omitempty" doc:"User's typed metadata, absent when not set"`
}

// CreateUserInput represents the create-user operation request.
type CreateUserInput struct {
	Body struct {
		Username  string       `json:"username" example:"homer" doc:"Unique username"`
		Email     string       `json:"email" format:"email" example:"homer@example.com" doc:"Email address"`
		FirstName string       `json:"firstName" example:"Homer" doc:"First name"`
		LastName  string       `json:"lastName" example:"Simpson" doc:"Last name"`
		Meta      *UserMetaBox `json:"meta,omitempty" doc:"Typed metadata, omit for none"`
	}
}

// GetUserInput represents the get-user-by-id operation request.
type GetUserInput struct {
	ID           string `path:"id" format:"uuid" doc:"User ID (UUID)"`
	IsLoadGroups bool   `query:"isLoadGroups" doc:"Preload and include the user's groups in the response"`
}

// UpdateUserInput represents the update-user-by-id operation request.
// Every body field is optional; only fields present in the JSON body are
// changed, the rest are left as-is (nil pointer = "don't touch"). Meta
// follows the same rule: omit to leave unchanged, send a value to replace
// it wholesale — there's no way to explicitly clear an existing meta back
// to unset via this endpoint (not requested — add a distinct signal, e.g. a
// tri-state wrapper, if that's needed later).
type UpdateUserInput struct {
	ID   string `path:"id" format:"uuid" doc:"User ID (UUID)"`
	Body struct {
		FirstName *string      `json:"firstName,omitempty" doc:"New first name, omit to leave unchanged"`
		LastName  *string      `json:"lastName,omitempty" doc:"New last name, omit to leave unchanged"`
		Email     *string      `json:"email,omitempty" format:"email" doc:"New email, omit to leave unchanged"`
		Meta      *UserMetaBox `json:"meta,omitempty" doc:"New typed metadata (replaces any existing value), omit to leave unchanged"`
	}
}

// DeleteUserInput represents the delete-user-by-id operation request.
type DeleteUserInput struct {
	ID string `path:"id" format:"uuid" doc:"User ID (UUID)"`
}

// ListUsersInput represents the list-users operation request.
// CreatedAtFrom/CreatedAtTo use the zero time.Time value to mean "not set"
// (huma query params can't be pointers).
type ListUsersInput struct {
	Search        string            `query:"search" doc:"Filter by substring match on username or email"`
	CreatedAtFrom time.Time         `query:"createdAtFrom" doc:"Only users created on/after this time (RFC 3339), omit to leave unbounded"`
	CreatedAtTo   time.Time         `query:"createdAtTo" doc:"Only users created on/before this time (RFC 3339), omit to leave unbounded"`
	IDs           []string          `query:"ids,explode" format:"uuid" nullable:"false" doc:"Filter to only these user IDs (repeat as ?ids=a&ids=b)"`
	IsLoadGroups  bool              `query:"isLoadGroups" doc:"Preload and include each user's groups in the response"`
	SortBy        string            `query:"sortBy" enum:"username,email,createdAt" default:"createdAt" doc:"Field to sort by"`
	SortOrder     request.SortOrder `query:"sortOrder" enum:"asc,desc" default:"asc" doc:"Sort direction"`
	request.Pagination
}

// AddUsersToGroupsInput represents the add-users-to-groups operation
// request: bulk-add users to groups, each user with its own group set.
type AddUsersToGroupsInput struct {
	Body struct {
		// Keyed by userId; a duplicate userId is impossible by
		// construction (JSON object keys are unique).
		UserGroups map[string][]string `json:"userGroups" doc:"userId -> groupIds to add that user to"`
	}
}

// CreateUserWithGroups is one entry of CreateUsersWithGroupsInput: a user to
// create plus the groups to add it to (GroupIds may be empty).
type CreateUserWithGroups struct {
	Username  string       `json:"username" example:"homer" doc:"Unique username"`
	Email     string       `json:"email" format:"email" example:"homer@example.com" doc:"Email address"`
	FirstName string       `json:"firstName" example:"Homer" doc:"First name"`
	LastName  string       `json:"lastName" example:"Simpson" doc:"Last name"`
	Meta      *UserMetaBox `json:"meta,omitempty" doc:"Typed metadata, omit for none"`
	GroupIds  []string     `json:"groupIds,omitempty" format:"uuid" doc:"Groups to add this user to"`
}

// CreateUsersWithGroupsInput represents the create-users-with-groups
// operation request: create users and add each to its own groups,
// atomically — any failure (bad group id, duplicate username, ...) rolls
// back every user in the request, not just the one that failed.
type CreateUsersWithGroupsInput struct {
	Body struct {
		Users []CreateUserWithGroups `json:"users" doc:"Users to create, each with its own group set"`
	}
}
