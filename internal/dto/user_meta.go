package dto

import "github.com/manhrev/gorest/pkg/dto/wrapper"

// UserMeta is implemented by every concrete user-meta variant.
//
// ponytail: only two placeholder variants (AdminMeta, CustomerMeta) exist to
// prove the pattern — add a real variant by: defining the struct, giving it
// a kind(), and registering it below.
type UserMeta interface{ wrapper.Kind }

type AdminMeta struct {
	Level string `json:"level" doc:"Admin access level"`
}

func (AdminMeta) Kind() string { return "admin" }

type CustomerMeta struct {
	Company string `json:"company" doc:"Customer's company name"`
}

func (CustomerMeta) Kind() string { return "customer" }

var (
	_ = wrapper.RegisterVariant[UserMeta, AdminMeta]("admin")
	_ = wrapper.RegisterVariant[UserMeta, CustomerMeta]("customer")
	_ = wrapper.RegisterVariantFallback[UserMeta, wrapper.UnknownMeta]()
)

// UserMetaBox is wrapper.Box[UserMeta] under its own name so existing call
// sites (UserDTO.Meta *UserMetaBox, ...) don't need to change, and so it
// gets its own name in generated OpenAPI component names instead of a
// generic instantiation name. Must be a type alias (=), not a new
// definition — Box[T]'s MarshalJSON/UnmarshalJSON/Schema methods need to
// carry over unchanged.
type UserMetaBox = wrapper.Box[UserMeta]
