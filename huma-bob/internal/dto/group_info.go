package dto

import "github.com/manhrev/gorest/pkg/dto/wrapper"

// GroupInfo is implemented by every concrete group-info variant. Same
// pattern as UserMeta (see user_meta.go) — Box[T] discriminated union via
// wrapper.Kind, decoded by a "type" field.
type GroupInfo interface{ wrapper.Kind }

type DepartmentInfo struct {
	Code string `json:"code" doc:"Department code"`
}

func (DepartmentInfo) Kind() string { return "department" }

type TeamInfo struct {
	Lead string `json:"lead" doc:"Team lead"`
}

func (TeamInfo) Kind() string { return "team" }

type ProjectInfo struct {
	ProjectCode string `json:"projectCode" doc:"Project code"`
}

func (ProjectInfo) Kind() string { return "project" }

type LocationInfo struct {
	Office string `json:"office" doc:"Office location"`
}

func (LocationInfo) Kind() string { return "location" }

type CommitteeInfo struct {
	Chair string `json:"chair" doc:"Committee chair"`
}

func (CommitteeInfo) Kind() string { return "committee" }

var (
	_ = wrapper.RegisterVariant[GroupInfo, DepartmentInfo]("department")
	_ = wrapper.RegisterVariant[GroupInfo, TeamInfo]("team")
	_ = wrapper.RegisterVariant[GroupInfo, ProjectInfo]("project")
	_ = wrapper.RegisterVariant[GroupInfo, LocationInfo]("location")
	_ = wrapper.RegisterVariant[GroupInfo, CommitteeInfo]("committee")
	_ = wrapper.RegisterVariantFallback[GroupInfo, wrapper.UnknownMeta]()
)

// GroupInfoBox is wrapper.Box[GroupInfo] under its own name — see
// UserMetaBox's doc comment for why this must be a type alias.
type GroupInfoBox = wrapper.Box[GroupInfo]
