package engine

import (
	"testing"
)

func TestTranspile(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{".", "x"},
		{".foo", "x.foo"},
		{".[0]", "x[0]"},
		{"foo", "foo"},
		{"@.baz", "map((x, i) => apply(x.baz, x, i))"},
		{"?.foo > 42", "filter((x, i) => apply(x.foo > 42, x, i))"},
		{".foo[].bar[]", "(x => x.foo.flatMap(x => x.bar.flatMap(x => x)))(x)"},
	}
	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := transpile(tt.code)
			if got != tt.want {
				t.Errorf("transpile(%q) = %q; want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestFoldSimple(t *testing.T) {
	tests := []struct {
		parts []string
		want  string
	}{
		{[]string{".foo"}, "x => x.foo"},
		{[]string{".foo", ".bar"}, "x => x.foo.flatMap(x => x.bar)"},
	}
	for _, tt := range tests {
		got := fold(tt.parts)
		if got != tt.want {
			t.Errorf("fold(%v) = %q; want %q", tt.parts, got, tt.want)
		}
	}
}
