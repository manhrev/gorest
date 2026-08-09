// Package converter holds model <-> dto translation, kept out of the
// service layer so it's reusable and independently testable.
package converter

import (
	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/db/model"
)

func UserToDto(m *model.User) dto.UserDTO {
	d := dto.UserDTO{
		ID:        m.ID,
		Username:  m.Username,
		Email:     m.Email,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	// Only set when the caller preloaded it (m.R.UserGroups + each
	// UserGroup's .R.Group) — see repo.FirstById's loadGroups param.
	if m.R.Loaded.UserGroups {
		d.Groups = make([]dto.GroupDTO, 0, len(m.R.UserGroups))
		for _, ug := range m.R.UserGroups {
			if ug.R.Group != nil {
				d.Groups = append(d.Groups, GroupToDto(ug.R.Group))
			}
		}
	}

	return d
}

func UsersToDtos(rows model.UserSlice) []dto.UserDTO {
	out := make([]dto.UserDTO, len(rows))
	for i, m := range rows {
		out[i] = UserToDto(m)
	}

	return out
}
