package dto

import (
	"encoding/json"
	"reflect"

	"github.com/danielgtaylor/huma/v2"
)

// UserMeta is implemented by every concrete meta variant. metaKind() is
// written to/read from the "type" discriminator field in the JSON.
//
// ponytail: only two placeholder variants (AdminMeta, CustomerMeta) exist to
// prove the pattern — add a real variant by: defining the struct, giving it
// a metaKind(), and adding one line to userMetaVariants below.
type UserMeta interface {
	metaKind() string
}

type AdminMeta struct {
	Level string `json:"level" doc:"Admin access level"`
}

func (AdminMeta) metaKind() string { return "admin" }

type CustomerMeta struct {
	Company string `json:"company" doc:"Customer's company name"`
}

func (CustomerMeta) metaKind() string { return "customer" }

// UnknownMeta preserves any meta payload whose "type" isn't one of the
// known variants (a legacy row, or one written by a newer app version)
// instead of failing to decode it — a bad/legacy row must not break every
// query that scans the users table.
type UnknownMeta struct {
	Type string
	Raw  json.RawMessage
}

func (m UnknownMeta) metaKind() string { return m.Type }

// MarshalJSON returns the original payload verbatim — it already contains
// "type", so UserMetaBox.MarshalJSON's re-injection of it is a no-op.
func (m UnknownMeta) MarshalJSON() ([]byte, error) { return m.Raw, nil }

// userMetaVariants is the single source of truth for every known meta
// shape — driving both JSON (de)serialization and the OpenAPI schema.
var userMetaVariants = []struct {
	kind string
	typ  reflect.Type
}{
	{"admin", reflect.TypeFor[AdminMeta]()},
	{"customer", reflect.TypeFor[CustomerMeta]()},
}

// UserMetaBox wraps a UserMeta for (de)serialization — a bare interface
// field can't unmarshal on its own. Use as a pointer field in a DTO; nil
// means "not set".
type UserMetaBox struct {
	Meta UserMeta
}

func (b UserMetaBox) MarshalJSON() ([]byte, error) {
	if b.Meta == nil {
		return []byte("null"), nil
	}
	raw, err := json.Marshal(b.Meta)
	if err != nil {
		return nil, err
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	kind, err := json.Marshal(b.Meta.metaKind())
	if err != nil {
		return nil, err
	}
	m["type"] = kind
	return json.Marshal(m)
}

func (b *UserMetaBox) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		b.Meta = nil
		return nil
	}
	var disc struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	for _, v := range userMetaVariants {
		if v.kind != disc.Type {
			continue
		}
		m := reflect.New(v.typ)
		if err := json.Unmarshal(data, m.Interface()); err != nil {
			return err
		}
		b.Meta = m.Elem().Interface().(UserMeta)
		return nil
	}
	b.Meta = UnknownMeta{Type: disc.Type, Raw: append(json.RawMessage(nil), data...)}
	return nil
}

// Schema implements huma.SchemaProvider so the OpenAPI doc shows the real
// union (oneOf + discriminator) instead of huma's default "any" fallback
// for an interface-typed field. UnknownMeta is intentionally not part of
// the oneOf — it's a decode-time fallback for unrecognized "type" values,
// not a shape clients should send; unrecognized types still decode fine
// at runtime, they just don't validate against the published schema.
func (UserMetaBox) Schema(r huma.Registry) *huma.Schema {
	oneOf := make([]*huma.Schema, len(userMetaVariants))
	for i, v := range userMetaVariants {
		s := r.Schema(v.typ, false, v.typ.Name())
		s.Properties["type"] = &huma.Schema{Type: huma.TypeString, Enum: []any{v.kind}}
		s.Required = append(s.Required, "type")
		oneOf[i] = s
	}
	return &huma.Schema{
		OneOf:         oneOf,
		Discriminator: &huma.Discriminator{PropertyName: "type"},
	}
}
