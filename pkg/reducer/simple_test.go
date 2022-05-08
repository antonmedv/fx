package reducer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_splitPath(t *testing.T) {
	tests := []struct {
		args []string
		want []interface{}
	}{
		{
			args: []string{},
			want: []interface{}{},
		},
		{
			args: []string{"."},
			want: []interface{}{},
		},
		{
			args: []string{"x"},
			want: []interface{}{},
		},
		{
			args: []string{".foo"},
			want: []interface{}{"foo"},
		},
		{
			args: []string{"x.foo"},
			want: []interface{}{"foo"},
		},
		{
			args: []string{"x[42]"},
			want: []interface{}{42},
		},
		{
			args: []string{".[42]"},
			want: []interface{}{42},
		},
		{
			args: []string{".42"},
			want: []interface{}{"42"},
		},
		{
			args: []string{".физ"},
			want: []interface{}{"физ"},
		},
		{
			args: []string{".foo.bar"},
			want: []interface{}{"foo", "bar"},
		},
		{
			args: []string{".foo", ".bar"},
			want: []interface{}{"foo", "bar"},
		},
		{
			args: []string{".foo[42]"},
			want: []interface{}{"foo", 42},
		},
		{
			args: []string{".foo[42].bar"},
			want: []interface{}{"foo", 42, "bar"},
		},
		{
			args: []string{".foo[1][2]"},
			want: []interface{}{"foo", 1, 2},
		},
		{
			args: []string{".foo[\"bar\"]"},
			want: []interface{}{"foo", "bar"},
		},
		{
			args: []string{".foo[\"bar\\\"\"]"},
			want: []interface{}{"foo", "bar\""},
		},
		{
			args: []string{".foo['bar']['baz\\'']"},
			want: []interface{}{"foo", "bar", "baz\\'"},
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			path, ok := splitPath(tt.args)
			require.Equal(t, tt.want, path)
			require.True(t, ok)
		})
	}
}

func Test_splitPath_negative(t *testing.T) {
	tests := []struct {
		args []string
	}{
		{
			args: []string{"./"},
		},
		{
			args: []string{"x/"},
		},
		{
			args: []string{"1+1"},
		},
		{
			args: []string{"x[42"},
		},
		{
			args: []string{".i % 2"},
		},
		{
			args: []string{"x[for x]"},
		},
		{
			args: []string{"x['y'."},
		},
		{
			args: []string{"x[0?"},
		},
		{
			args: []string{"x[\"\\u"},
		},
		{
			args: []string{"x['\\n"},
		},
		{
			args: []string{"x[9999999999999999999999999999999999999]"},
		},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			path, ok := splitPath(tt.args)
			require.False(t, ok, path)
		})
	}
}
