package server

import "testing"

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header  string
		want    string
		wantErr bool
	}{
		{"Bearer abc.def.ghi", "abc.def.ghi", false},
		{"bearer abc.def.ghi", "abc.def.ghi", false}, // scheme name case-insensitive, RFC 7235
		{"BEARER abc.def.ghi", "abc.def.ghi", false},
		{"", "", true},
		{"abc.def.ghi", "", true},   // missing scheme
		{"Bearer ", "", true},       // empty token
		{"Bearer", "", true},        // no space, no token
		{"Basic dXNlcjpwYXNz", "", true},
	}

	for _, c := range cases {
		got, err := bearerToken(c.header)
		if c.wantErr {
			if err == nil {
				t.Errorf("bearerToken(%q): expected error, got nil", c.header)
			}
			continue
		}

		if err != nil {
			t.Errorf("bearerToken(%q): unexpected error: %v", c.header, err)
		}

		if got != c.want {
			t.Errorf("bearerToken(%q) = %q, want %q", c.header, got, c.want)
		}
	}
}
