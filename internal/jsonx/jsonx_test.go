package jsonx_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/jsonx"
)

func TestJsonParser_Parse(t *testing.T) {
	tests := []struct {
		input    string
		wantKind jsonx.Kind
		wantErr  bool
	}{
		{`"hello"`, jsonx.String, false},
		{`42`, jsonx.Number, false},
		{`-123.45`, jsonx.Number, false},
		{`true`, jsonx.Bool, false},
		{`false`, jsonx.Bool, false},
		{`null`, jsonx.Null, false},
		{`{}`, jsonx.Object, false},
		{`[]`, jsonx.Array, false},
		{`{"key":"value"}`, jsonx.Object, false},
		{`[1, 2, 3]`, jsonx.Array, false},
		{`   "test"   `, jsonx.String, false},
		{`// comment
		"test"`, jsonx.String, false},
		{`/* comment */"test"`, jsonx.String, false},
		{`{"a":1,}`, jsonx.Object, false},
		{`[1,2,]`, jsonx.Array, false},
		{`"abc`, jsonx.Err, true},
		{`"ab\q"`, jsonx.Err, true},
		{`truth`, jsonx.Err, true},
		{`1e`, jsonx.Err, true},
		{`[1, 2`, jsonx.Err, true},
		{`/* test`, jsonx.Err, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.input))
			if tt.wantErr {
				assert.Error(t, err, "expected error for input: %s", tt.input)
			} else {
				assert.NoError(t, err, "unexpected error for input: %s", tt.input)
				assert.Equal(t, tt.wantKind, node.Kind, "unexpected kind for input: %s", tt.input)
			}
		})
	}
}

func TestJsonParser_Recovery(t *testing.T) {
	brokenJSON := `{ "a": 1 }here goes the text`
	t.Run("Recover", func(t *testing.T) {
		p := jsonx.NewJsonParser(strings.NewReader(brokenJSON))
		_, _ = p.Parse() // trigger error
		node := p.Recover()

		assert.Equal(t, jsonx.Err, node.Kind, "expected recovery node to be of Kind Err")
		assert.NotEmpty(t, node.Value, "expected recovery node to contain error snippet")
		assert.Equal(t, string(node.Value), "here goes the text", "expected recovery node to contain error snippet")
	})
}
