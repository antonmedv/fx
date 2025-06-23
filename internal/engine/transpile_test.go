package engine

import (
	"testing"
)

func TestTranspileBasic(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{".", "x"},
		{".foo", "x.foo"},
		{".[0]", "x[0]"},
		{"foo", "foo"},
	}
	for _, tt := range tests {
		got := transpile(tt.code)
		if got != tt.want {
			t.Errorf("transpile(%q) = %q; want %q", tt.code, got, tt.want)
		}
	}
}

func TestTranspileBracketAndNested(t *testing.T) {
	code := ".foo[].bar[]"
	want := "(x => x.foo.flatMap(x => x.bar.flatMap(x => x)))(x)"
	got := transpile(code)
	if got != want {
		t.Errorf("transpile(%q) = %q; want %q", code, got, want)
	}
}

func TestTranspileMapAndAt(t *testing.T) {
	tests := []struct {
		code string
		want string
	}{
		{"@.baz", "x.map((x, i) => apply(x.baz, x, i))"},
	}
	for _, tt := range tests {
		got := transpile(tt.code)
		if got != tt.want {
			t.Errorf("transpile(%q) = %q; want %q", tt.code, got, tt.want)
		}
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
