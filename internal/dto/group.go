package dto

import (
	"time"

	"github.com/manhrev/gorest/pkg/dto/request"
)

// GroupDTO is the canonical group representation returned by the API.
type GroupDTO struct {
	ID          string    `json:"id" format:"uuid" doc:"Group ID (UUID)"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// CreateGroupInput represents the create-group operation request.
type CreateGroupInput struct {
	Body struct {
		Name        string `json:"name" example:"Admins" doc:"Unique group name"`
		Description string `json:"description,omitempty" doc:"Group description"`
	}
}

// GetGroupInput represents the get-group-by-id operation request.
type GetGroupInput struct {
	ID string `path:"id" format:"uuid" doc:"Group ID (UUID)"`
}

// UpdateGroupInput represents the update-group-by-id operation request.
// Every body field is optional; only fields present in the JSON body are
// changed, the rest are left as-is (nil pointer = "don't touch").
type UpdateGroupInput struct {
	ID   string `path:"id" format:"uuid" doc:"Group ID (UUID)"`
	Body struct {
		Name        *string `json:"name,omitempty" doc:"New name, omit to leave unchanged"`
		Description *string `json:"description,omitempty" doc:"New description, omit to leave unchanged"`
	}
}

// DeleteGroupInput represents the delete-group-by-id operation request.
type DeleteGroupInput struct {
	ID string `path:"id" format:"uuid" doc:"Group ID (UUID)"`
}

// ListGroupsInput represents the list-groups operation request.
type ListGroupsInput struct {
	Search        string            `query:"search" doc:"Filter by substring match on name"`
	CreatedAtFrom time.Time         `query:"createdAtFrom" doc:"Only groups created on/after this time (RFC 3339), omit to leave unbounded"`
	CreatedAtTo   time.Time         `query:"createdAtTo" doc:"Only groups created on/before this time (RFC 3339), omit to leave unbounded"`
	IDs           []string          `query:"ids,explode" format:"uuid" nullable:"false" doc:"Filter to only these group IDs (repeat as ?ids=a&ids=b)"`
	SortBy        string            `query:"sortBy" enum:"name,createdAt" default:"createdAt" doc:"Field to sort by"`
	SortOrder     request.SortOrder `query:"sortOrder" enum:"asc,desc" default:"asc" doc:"Sort direction"`
	request.Pagination
}
