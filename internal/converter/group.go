package converter

import (
	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/db/model"
)

func GroupToDto(m *model.Group) dto.GroupDTO {
	return dto.GroupDTO{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description.V,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}
}

func GroupsToDtos(rows model.GroupSlice) []dto.GroupDTO {
	out := make([]dto.GroupDTO, len(rows))
	for i, m := range rows {
		out[i] = GroupToDto(m)
	}

	return out
}
