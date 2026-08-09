package dto

import (
	"encoding/json"
	"testing"
)

func TestUserMetaBox_RoundTrip(t *testing.T) {
	in := UserMetaBox{Meta: AdminMeta{Level: "super"}}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"level":"super","type":"admin"}` {
		t.Fatalf("unexpected json: %s", raw)
	}

	var out UserMetaBox
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Meta != in.Meta {
		t.Fatalf("got %#v, want %#v", out.Meta, in.Meta)
	}
}

func TestUserMetaBox_UnknownType(t *testing.T) {
	src := `{"type":"bogus","foo":"bar"}`

	var out UserMetaBox
	if err := json.Unmarshal([]byte(src), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	unk, ok := out.Meta.(UnknownMeta)
	if !ok {
		t.Fatalf("got %#v, want UnknownMeta", out.Meta)
	}
	if unk.Type != "bogus" {
		t.Fatalf("got type %q, want %q", unk.Type, "bogus")
	}

	// Round-trips back to the same fields (key order isn't preserved —
	// UserMetaBox.MarshalJSON always goes through a map to inject "type").
	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal got: %v", err)
	}
	if err := json.Unmarshal([]byte(src), &want); err != nil {
		t.Fatalf("unmarshal want: %v", err)
	}
	if len(got) != len(want) || got["type"] != want["type"] || got["foo"] != want["foo"] {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestUserMetaBox_Null(t *testing.T) {
	var out UserMetaBox
	if err := json.Unmarshal([]byte("null"), &out); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if out.Meta != nil {
		t.Fatalf("expected nil Meta, got %#v", out.Meta)
	}
}
