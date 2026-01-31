package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHexDigit(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected bool
	}{
		{"digit 0", '0', true},
		{"digit 5", '5', true},
		{"digit 9", '9', true},
		{"lowercase a", 'a', true},
		{"lowercase f", 'f', true},
		{"uppercase A", 'A', true},
		{"uppercase F", 'F', true},
		{"lowercase g", 'g', false},
		{"uppercase G", 'G', false},
		{"space", ' ', false},
		{"lowercase z", 'z', false},
		{"special char", '@', false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsHexDigit(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsDigit(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		expected bool
	}{
		{"digit 0", '0', true},
		{"digit 5", '5', true},
		{"digit 9", '9', true},
		{"lowercase a", 'a', false},
		{"uppercase A", 'A', false},
		{"space", ' ', false},
		{"special char", '-', false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IsDigit(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		needle   int
		haystack []int
		expected bool
	}{
		{"found at start", 1, []int{1, 2, 3}, true},
		{"found in middle", 2, []int{1, 2, 3}, true},
		{"found at end", 3, []int{1, 2, 3}, true},
		{"not found", 4, []int{1, 2, 3}, false},
		{"empty slice", 1, []int{}, false},
		{"single element found", 1, []int{1}, true},
		{"single element not found", 2, []int{1}, false},
		{"negative number found", -1, []int{-1, 0, 1}, true},
		{"zero found", 0, []int{-1, 0, 1}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Contains(tc.needle, tc.haystack)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"simple string", `"hello"`, "hello", false},
		{"empty string", `""`, "", false},
		{"with spaces", `"hello world"`, "hello world", false},
		{"with escape", `"hello\nworld"`, "hello\nworld", false},
		{"with tab", `"hello\tworld"`, "hello\tworld", false},
		{"with unicode", `"hello \u4e16\u754c"`, "hello 世界", false},
		{"with quotes inside", `"hello \"world\""`, `hello "world"`, false},
		{"with backslash", `"hello\\world"`, `hello\world`, false},
		{"invalid json", `hello`, "", true},
		{"number", `123`, "", true},
		{"unclosed quote", `"hello`, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := Unquote(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}
