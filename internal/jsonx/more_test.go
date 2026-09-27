package jsonx_test

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

func TestJsonParserMore(t *testing.T) {
	tests := []struct {
		input string
		more  bool
	}{
		{`{"a":1}`, false},
		{"{\"a\":1}\n", false},
		{"{\"a\":1}\n  // comment\n", false},
		{"1 2", true},
		{"{\"a\":1}\n{\"a\":2}\n", true},
		{"{\"a\":1}\nx", true}, // Invalid input is left for Parse.
	}
	for _, tt := range tests {
		p := jsonx.NewJsonParser(strings.NewReader(tt.input), false)
		_, err := p.Parse()
		require.NoError(t, err, tt.input)
		for range 2 {
			more, err := p.More()
			require.NoError(t, err, tt.input)
			require.Equal(t, tt.more, more, tt.input)
		}
	}
}

// An error found while looking ahead must not be lost: More and Parse keep
// returning it instead of reporting the end of input.
func TestJsonParserMoreKeepsError(t *testing.T) {
	for _, input := range []string{"{} /* unfinished", "{} /x"} {
		p := jsonx.NewJsonParser(strings.NewReader(input), false)
		_, err := p.Parse()
		require.NoError(t, err, input)

		_, first := p.More()
		require.Error(t, first, input)
		for range 2 {
			more, err := p.More()
			require.False(t, more, input)
			require.Equal(t, first, err, input)
		}
		_, err = p.Parse()
		require.Equal(t, first, err, input)
		require.NotEqual(t, io.EOF, err, input)
	}
}

func TestLineParserMore(t *testing.T) {
	for input, want := range map[string]bool{"a": false, "a\n": false, "a\nb": true} {
		p := jsonx.NewLineParser(strings.NewReader(input))
		_, err := p.Parse()
		require.NoError(t, err, input)
		more, err := p.More()
		require.NoError(t, err, input)
		require.Equal(t, want, more, input)
	}
}

// Input without any value, even if it has whitespace or comments, ends at once.
func TestJsonParserEmpty(t *testing.T) {
	for _, input := range []string{"", " \n\t\r\n", "// comment\n", "\xEF\xBB\xBF\n"} {
		p := jsonx.NewJsonParser(strings.NewReader(input), false)
		_, err := p.Parse()
		require.Equal(t, io.EOF, err, input)
	}
}
