package converter

import (
	"database/sql"

	"github.com/stephenafamo/bob/types"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/db/model"
)

func GroupToDto(m *model.Group) dto.GroupDTO {
	d := dto.GroupDTO{
		ID:          m.ID,
		Name:        m.Name,
		Description: m.Description.V,
		CreatedAt:   m.CreatedAt,
		UpdatedAt:   m.UpdatedAt,
	}

	// bob already decoded this into GroupInfoBox on scan (types.JSON[T]
	// uses T's json.Unmarshaler) — no manual decode needed here.
	if m.GroupInfo.Valid {
		d.GroupInfo = &m.GroupInfo.V.Val
	}

	return d
}

// GroupInfoToSetterField converts an API-facing *dto.GroupInfoBox to the
// model.GroupSetter.GroupInfo shape. nil -> nil (column left untouched on
// update, or omitted -> SQL NULL on insert since it has no default).
func GroupInfoToSetterField(info *dto.GroupInfoBox) *sql.Null[types.JSON[dto.GroupInfoBox]] {
	if info == nil {
		return nil
	}
	return &sql.Null[types.JSON[dto.GroupInfoBox]]{Valid: true, V: types.NewJSON(*info)}
}

func GroupsToDtos(rows model.GroupSlice) []dto.GroupDTO {
	out := make([]dto.GroupDTO, len(rows))
	for i, m := range rows {
		out[i] = GroupToDto(m)
	}

	return out
}
