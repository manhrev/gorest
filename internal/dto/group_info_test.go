package dto

import (
	"encoding/json"
	"testing"

	"github.com/manhrev/gorest/pkg/dto/wrapper"
)

func TestGroupInfoBox_RoundTrip(t *testing.T) {
	in := GroupInfoBox{Value: TeamInfo{Lead: "Marge"}}

	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(raw) != `{"lead":"Marge","type":"team"}` {
		t.Fatalf("unexpected json: %s", raw)
	}

	var out GroupInfoBox
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.Value != in.Value {
		t.Fatalf("got %#v, want %#v", out.Value, in.Value)
	}
}

func TestGroupInfoBox_AllVariants(t *testing.T) {
	cases := []struct {
		name string
		in   GroupInfo
	}{
		{"department", DepartmentInfo{Code: "ENG"}},
		{"team", TeamInfo{Lead: "Marge"}},
		{"project", ProjectInfo{ProjectCode: "P-42"}},
		{"location", LocationInfo{Office: "Springfield"}},
		{"committee", CommitteeInfo{Chair: "Lisa"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			raw, err := json.Marshal(GroupInfoBox{Value: c.in})
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var out GroupInfoBox
			if err := json.Unmarshal(raw, &out); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if out.Value != c.in {
				t.Fatalf("got %#v, want %#v", out.Value, c.in)
			}
		})
	}
}

func TestGroupInfoBox_UnknownType(t *testing.T) {
	var out GroupInfoBox
	if err := json.Unmarshal([]byte(`{"type":"bogus","foo":"bar"}`), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := out.Value.(wrapper.UnknownMeta); !ok {
		t.Fatalf("got %#v, want wrapper.UnknownMeta", out.Value)
	}
}

func TestGroupInfoBox_Null(t *testing.T) {
	var out GroupInfoBox
	if err := json.Unmarshal([]byte("null"), &out); err != nil {
		t.Fatalf("unmarshal null: %v", err)
	}
	if out.Value != nil {
		t.Fatalf("expected nil Value, got %#v", out.Value)
	}
}
