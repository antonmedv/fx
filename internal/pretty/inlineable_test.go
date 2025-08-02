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
			expected: true,
		},
		{
			name:     "array without key",
			json:     `[1, 2, 3]`,
			expected: false,
		},
		{
			name:     "simple object with number values",
			json:     `{"key": {"a": 1, "b": 2, "c": 3}}`,
			expected: false,
		},
		{
			name:     "simple object with boolean values",
			json:     `{"key": {"a": true, "b": false, "c": true}}`,
			expected: false,
		},
		{
			name:     "simple object with short string values",
			json:     `{"key": {"a": "short", "b": "string"}}`,
			expected: false,
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
			expected: true,
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

func TestIsNestedArrays(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		// Valid tables
		{
			name:     "valid table - array of arrays with numbers of same size",
			json:     `[[1, 2, 3], [4, 5, 6], [7, 8, 9]]`,
			expected: true,
		},
		{
			name:     "valid table - array of arrays with single number",
			json:     `[[1], [2], [3]]`,
			expected: true,
		},
		{
			name:     "not a table - table with key",
			json:     `{"table": [[1, 2], [3, 4]]}`,
			expected: false,
		},
		{
			name:     "valid table with many rows",
			json:     `[[1, 2], [3, 4], [5, 6], [7, 8], [9, 10]]`,
			expected: true,
		},
		{
			name:     "valid table with only one inner array",
			json:     `[[1, 2, 3]]`,
			expected: true,
		},
		{
			name:     "valid table with multiple arrays of different sizes",
			json:     `[[1, 2, 3, 4], [5, 6], [7, 8, 9], [10]]`,
			expected: true,
		},

		// Invalid tables
		{
			name:     "not a table - array with non-array elements",
			json:     `[1, 2, 3]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with non-number elements",
			json:     `[[1, 2], ["a", "b"]]`,
			expected: false,
		},
		{
			name:     "valid table - array of arrays with different sizes",
			json:     `[[1, 2, 3], [4, 5]]`,
			expected: true,
		},
		{
			name:     "not a table - empty array",
			json:     `[]`,
			expected: false,
		},
		{
			name:     "not a table - array with mixed content",
			json:     `[[1, 2], 3, [4, 5]]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with boolean values",
			json:     `[[true, false], [false, true]]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with string values",
			json:     `[["a", "b"], ["c", "d"]]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with null values",
			json:     `[[null, null], [null, null]]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with object values",
			json:     `[[{"a": 1}, {"b": 2}], [{"c": 3}, {"d": 4}]]`,
			expected: false,
		},
		{
			name:     "not a table - array of arrays with array values",
			json:     `[[[1], [2]], [[3], [4]]]`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			var testNode *jsonx.Node
			if strings.Contains(tt.json, `"table":`) {
				testNode = node.FindChildByKey("table")
				require.NotNil(t, testNode, "Could not find node with key 'table'")
			} else {
				testNode = node
			}

			output := pretty.Print(testNode, true)

			// Check if the output has the characteristics of a table format
			// For a table, each inner array should be on its own line in a tabular format
			lines := strings.Split(output, "\n")

			// A table should have at least 3 lines (opening bracket, content, closing bracket)
			// and the content lines should be formatted in a specific way
			isTable := false
			if len(lines) >= 3 {
				// Check if the first line contains the opening bracket
				if strings.Contains(lines[0], "[") {
					// Check if the last non-empty line contains the closing bracket
					lastNonEmptyIndex := len(lines) - 1
					for lastNonEmptyIndex >= 0 && lines[lastNonEmptyIndex] == "" {
						lastNonEmptyIndex--
					}

					if lastNonEmptyIndex >= 0 && strings.Contains(lines[lastNonEmptyIndex], "]") {
						// Check if the middle lines have a consistent format
						// In a table, each line should start with the same indentation and contain numbers
						isTable = true
						for i := 1; i < lastNonEmptyIndex; i++ {
							if lines[i] == "" {
								continue
							}
							// Each line in a table should contain numbers and be properly indented
							if !strings.Contains(lines[i], "[") || !strings.Contains(lines[i], "]") {
								isTable = false
								break
							}
						}
					}
				}
			}

			assert.Equal(t, tt.expected, isTable,
				"Expected isNestedArrays to be %v for %s, but got %v\nOutput:\n%s",
				tt.expected, tt.json, isTable, output)
		})
	}
}

func TestIsArrayOfSimpleObject(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected bool
	}{
		// Valid arrays of simple objects
		{
			name:     "array of simple objects with number values",
			json:     `[{"a": 1, "b": 2}, {"a": 3, "b": 4}, {"a": 5, "b": 6}]`,
			expected: true,
		},
		{
			name:     "array of simple objects with boolean values",
			json:     `[{"a": true, "b": false}, {"a": false, "b": true}]`,
			expected: true,
		},
		{
			name:     "array of simple objects with short string values",
			json:     `[{"a": "short", "b": "text"}, {"a": "another", "b": "value"}]`,
			expected: true,
		},
		{
			name:     "array with single simple object",
			json:     `[{"a": 1, "b": 2, "c": 3}]`,
			expected: true,
		},
		{
			name:     "array of simple objects with different keys but same value types",
			json:     `[{"a": 1, "b": 2}, {"c": 3, "d": 4}]`,
			expected: true,
		},
		{
			name:     "object containing array of simple objects",
			json:     `{"data": [{"a": 1, "b": 2}, {"a": 3, "b": 4}]}`,
			expected: false,
		},
		{
			name:     "empty array",
			json:     `[]`,
			expected: false,
		},
		{
			name:     "array of non-objects",
			json:     `[1, 2, 3]`,
			expected: false,
		},
		{
			name:     "array of mixed types",
			json:     `[{"a": 1}, 2, {"b": 3}]`,
			expected: false,
		},
		{
			name:     "array of objects with long keys",
			json:     `[{"veryLongKey": 1, "b": 2}, {"veryLongKey": 3, "b": 4}]`,
			expected: false,
		},
		{
			name:     "array - array of objects with mixed value types",
			json:     `[{"a": 1, "b": "string"}, {"a": 2, "b": "text"}]`,
			expected: true,
		},
		{
			name:     "array of objects with long string values",
			json:     `[{"a": "this is a very long string that exceeds twenty characters"}, {"a": "short"}]`,
			expected: true,
		},
		{
			name:     "array of objects with too many string values",
			json:     `[{"a": "string1", "b": "string2", "c": "string3"}, {"a": "text1", "b": "text2", "c": "text3"}]`,
			expected: false,
		},
		{
			name:     "array of objects with too many number values",
			json:     `[{"a": 1, "b": 2, "c": 3, "d": 4}, {"a": 5, "b": 6, "c": 7, "d": 8}]`,
			expected: false,
		},
		{
			name:     "array of objects with nested objects",
			json:     `[{"a": {"nested": 1}}, {"a": {"nested": 2}}]`,
			expected: false,
		},
		{
			name:     "array of objects with arrays",
			json:     `[{"a": [1, 2]}, {"a": [3, 4]}]`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			node, err := jsonx.Parse([]byte(tt.json))
			require.NoError(t, err)

			var testNode *jsonx.Node
			if strings.Contains(tt.json, `"data":`) {
				testNode = node.FindChildByKey("data")
				require.NotNil(t, testNode, "Could not find node with key 'data'")
			} else {
				testNode = node
			}

			output := pretty.Print(testNode, true)

			// Check if the output has the characteristics of a table format
			// For a table, each object should be on its own line in a tabular format
			lines := strings.Split(output, "\n")

			// A table should have at least 3 lines (opening bracket, content, closing bracket)
			// and the content lines should be formatted in a specific way
			isTable := false
			if len(lines) >= 3 {
				// Check if the first line contains the opening bracket
				if strings.Contains(lines[0], "[") {
					// Check if the last non-empty line contains the closing bracket
					lastNonEmptyIndex := len(lines) - 1
					for lastNonEmptyIndex >= 0 && lines[lastNonEmptyIndex] == "" {
						lastNonEmptyIndex--
					}

					if lastNonEmptyIndex >= 0 && strings.Contains(lines[lastNonEmptyIndex], "]") {
						// Check if the middle lines have a consistent format
						// In a table, each line should start with the same indentation and contain objects
						isTable = true
						for i := 1; i < lastNonEmptyIndex; i++ {
							if lines[i] == "" {
								continue
							}
							// Each line in a table should contain objects and be properly indented
							if !strings.Contains(lines[i], "{") || !strings.Contains(lines[i], "}") {
								isTable = false
								break
							}
						}
					}
				}
			}

			assert.Equal(t, tt.expected, isTable,
				"Expected isArrayOfSimpleObject to be %v for %s, but got %v\nOutput:\n%s",
				tt.expected, tt.json, isTable, output)
		})
	}
}
