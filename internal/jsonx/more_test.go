package jsonx_test

import (
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
		{"{\"a\":1}\n/x", true}, // Invalid input is left for Parse.
	}
	for _, tt := range tests {
		p := jsonx.NewJsonParser(strings.NewReader(tt.input), false)
		_, err := p.Parse()
		require.NoError(t, err, tt.input)
		require.Equal(t, tt.more, p.More(), tt.input)
		require.Equal(t, tt.more, p.More(), "idempotent: %q", tt.input)
	}
}

func TestLineParserMore(t *testing.T) {
	for input, more := range map[string]bool{"a": false, "a\n": false, "a\nb": true} {
		p := jsonx.NewLineParser(strings.NewReader(input))
		_, err := p.Parse()
		require.NoError(t, err, input)
		require.Equal(t, more, p.More(), input)
	}
}
