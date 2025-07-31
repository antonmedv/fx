package pretty_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/pretty"
)

func TestIsInlineable(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		// Array tests
		{
			name:     "simple array with numbers",
			json:     `{"key": [1, 2, 3]}`,
			expected: true,
		},
		{
			name:     "array with non-number elements",
			json:     `{"key": [1, "string", true]}`,
			expected: false,
		},
		{
			name:     "empty array",
			json:     `{"key": []}`,
			expected: true, // Empty arrays are inlined
		},
		{
			name:     "array without key",
			json:     `[1, 2, 3]`,
			expected: false,
		},

		// Object tests
		{
			name:     "simple object with number values",
			json:     `{"key": {"a": 1, "b": 2, "c": 3}}`,
			expected: true,
		},
		{
			name:     "simple object with boolean values",
			json:     `{"key": {"a": true, "b": false, "c": true}}`,
			expected: true,
		},
		{
			name:     "simple object with short string values",
			json:     `{"key": {"a": "short", "b": "string"}}`,
			expected: true,
		},
		{
			name:     "object with long key",
			json:     `{"key": {"thisIsAVeryLongKey": 1, "b": 2}}`,
			expected: false,
		},
		{
			name:     "object with mixed value types",
			json:     `{"key": {"a": 1, "b": "string"}}`,
			expected: false,
		},
		{
			name:     "object with long string value",
			json:     `{"key": {"a": "this is a very long string that exceeds twenty characters"}}`,
			expected: false,
		},
		{
			name:     "object with too many string values",
			json:     `{"key": {"a": "string1", "b": "string2", "c": "string3"}}`,
			expected: false,
		},
		{
			name:     "object with too many number values",
			json:     `{"key": {"a": 1, "b": 2, "c": 3, "d": 4}}`,
			expected: false,
		},
		{
			name:     "object without key",
			json:     `{"a": 1, "b": 2}`,
			expected: false,
		},
		{
			name:     "empty object",
			json:     `{"key": {}}`,
			expected: true, // Empty objects are inlined
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			var testNode *jsonx.Node
			if strings.Contains(tt.json, `"key":`) {
				testNode = node.FindChildByKey("key")
				require.NotNil(t, testNode, "Could not find node with key 'key'")
			} else {
				testNode = node
			}

			output := pretty.Print(testNode, true)

			lineCount := strings.Count(output, "\n")
			isInlined := lineCount == 1
			assert.Equal(t, tt.expected, isInlined,
				"Expected isInlineable to be %v for %s, but got %v\nOutput:\n%s",
				tt.expected, tt.json, isInlined, output)
		})
	}
}
