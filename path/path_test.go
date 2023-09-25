package path_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/path"
)

func Test_SplitPath(t *testing.T) {
	tests := []struct {
		input string
		want  []any
	}{
		{
			input: "",
			want:  []any{},
		},
		{
			input: ".",
			want:  []any{},
		},
		{
			input: "x",
			want:  []any{},
		},
		{
			input: ".foo",
			want:  []any{"foo"},
		},
		{
			input: "x.foo",
			want:  []any{"foo"},
		},
		{
			input: "x[42]",
			want:  []any{42},
		},
		{
			input: ".[42]",
			want:  []any{42},
		},
		{
			input: ".42",
			want:  []any{"42"},
		},
		{
			input: ".физ",
			want:  []any{"физ"},
		},
		{
			input: ".foo.bar",
			want:  []any{"foo", "bar"},
		},
		{
			input: ".foo[42]",
			want:  []any{"foo", 42},
		},
		{
			input: ".foo[42].bar",
			want:  []any{"foo", 42, "bar"},
		},
		{
			input: ".foo[1][2]",
			want:  []any{"foo", 1, 2},
		},
		{
			input: ".foo[\"bar\"]",
			want:  []any{"foo", "bar"},
		},
		{
			input: ".foo[\"bar\\\"\"]",
			want:  []any{"foo", "bar\""},
		},
		{
			input: ".foo['bar']['baz\\'']",
			want:  []any{"foo", "bar", "baz\\'"},
		},
		{
			input: "[42]",
			want:  []any{42},
		},
		{
			input: "[42].foo",
			want:  []any{42, "foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, ok := path.Split(tt.input)
			require.Equal(t, tt.want, p)
			require.True(t, ok)
		})
	}
}

func Test_SplitPath_negative(t *testing.T) {
	tests := []struct {
		input string
	}{
		{
			input: "./",
		},
		{
			input: "x/",
		},
		{
			input: "1+1",
		},
		{
			input: "x[42",
		},
		{
			input: ".i % 2",
		},
		{
			input: "x[for x]",
		},
		{
			input: "x['y'.",
		},
		{
			input: "x[0?",
		},
		{
			input: "x[\"\\u",
		},
		{
			input: "x['\\n",
		},
		{
			input: "x[9999999999999999999999999999999999999]",
		},
		{
			input: "x[]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			p, ok := path.Split(tt.input)
			require.False(t, ok, p)
		})
	}
}

func TestJoin(t *testing.T) {
	tests := []struct {
		input []any
		want  string
	}{
		{
			input: []any{},
			want:  "",
		},
		{
			input: []any{"foo"},
			want:  ".foo",
		},
		{
			input: []any{"foo", "bar"},
			want:  ".foo.bar",
		},
		{
			input: []any{"foo", 42},
			want:  ".foo[42]",
		},
		{
			input: []any{"foo", "bar", 42},
			want:  ".foo.bar[42]",
		},
		{
			input: []any{"foo", "bar", 42, "baz"},
			want:  ".foo.bar[42].baz",
		},
		{
			input: []any{"foo", "bar", 42, "baz", 1},
			want:  ".foo.bar[42].baz[1]",
		},
		{
			input: []any{"foo", "bar", 42, "baz", 1, "qux"},
			want:  ".foo.bar[42].baz[1].qux",
		},
		{
			input: []any{"foo bar"},
			want:  "[\"foo bar\"]",
		},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			require.Equal(t, tt.want, path.Join(tt.input))
		})
	}
}
