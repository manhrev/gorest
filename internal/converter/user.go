// Package converter holds model <-> dto translation, kept out of the
// service layer so it's reusable and independently testable.
package converter

import (
	"database/sql"

	"github.com/stephenafamo/bob/types"

	"github.com/manhrev/gorest/internal/dto"
	"github.com/manhrev/gorest/pkg/db/model"
)

func UserToDto(m *model.User) dto.User {
	d := dto.User{
		ID:        m.ID,
		Username:  m.Username,
		Email:     m.Email,
		FirstName: m.FirstName,
		LastName:  m.LastName,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}

	// bob already decoded this into UserMetaBox on scan (types.JSON[T] uses
	// T's json.Unmarshaler) — no manual decode needed here.
	if m.Meta.Valid {
		d.Meta = &m.Meta.V.Val
	}

	// Only set when the caller preloaded it (m.R.UserGroups + each
	// UserGroup's .R.Group) — see repo.FirstById's loadGroups param.
	if m.R.Loaded.UserGroups {
		d.Groups = make([]dto.Group, 0, len(m.R.UserGroups))
		for _, ug := range m.R.UserGroups {
			if ug.R.Group != nil {
				d.Groups = append(d.Groups, GroupToDto(ug.R.Group))
			}
		}
	}

	return d
}

// UserMetaToSetterField converts an API-facing *dto.UserMetaBox to the
// model.UserSetter.Meta shape. nil -> nil (column left untouched on update,
// or omitted -> SQL NULL on insert since it has no default).
func UserMetaToSetterField(meta *dto.UserMetaBox) *sql.Null[types.JSON[dto.UserMetaBox]] {
	if meta == nil {
		return nil
	}
	return &sql.Null[types.JSON[dto.UserMetaBox]]{Valid: true, V: types.NewJSON(*meta)}
}

func UsersToDtos(rows model.UserSlice) []dto.User {
	out := make([]dto.User, len(rows))
	for i, m := range rows {
		out[i] = UserToDto(m)
	}

	return out
}
