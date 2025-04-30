package jsonpath_test

import (
	"reflect"
	"testing"

	"github.com/antonmedv/fx/internal/jsonpath"
)

func TestParseSchemaRef(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   []any
		wantOk bool
	}{
		{
			name:   "empty fragment",
			input:  "#",
			want:   []any{},
			wantOk: true,
		},
		{
			name:   "simple defs",
			input:  "#/$defs/OrganizationConfig/Options",
			want:   []any{"$defs", "OrganizationConfig", "Options"},
			wantOk: true,
		},
		{
			name:   "with slash escape",
			input:  "#/path/with~1slash",
			want:   []any{"path", "with/slash"},
			wantOk: true,
		},
		{
			name:   "with tilde escape",
			input:  "#/path/with~0tilde",
			want:   []any{"path", "with~tilde"},
			wantOk: true,
		},
		{
			name:   "with percent",
			input:  "#/a%20b/c%2Fd",
			want:   []any{"a b", "c/d"},
			wantOk: true,
		},
		{
			name:   "invalid no prefix",
			input:  "foo/bar",
			want:   nil,
			wantOk: false,
		},
		{
			name:   "invalid bad escape",
			input:  "#/bad/%GG",
			want:   nil,
			wantOk: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := jsonpath.ParseSchemaRef(tt.input)
			if ok != tt.wantOk {
				t.Errorf("ParseSchemaRef(%q) ok = %v, want %v", tt.input, ok, tt.wantOk)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseSchemaRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
