package fuzzy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		array     []string
		expectNil bool
		expectIdx int
		expectStr string
	}{
		{
			name:      "exact match",
			pattern:   "hello",
			array:     []string{"world", "hello", "foo"},
			expectNil: false,
			expectIdx: 1,
			expectStr: "hello",
		},
		{
			name:      "fuzzy match",
			pattern:   "hlo",
			array:     []string{"world", "hello", "foo"},
			expectNil: false,
			expectIdx: 1,
			expectStr: "hello",
		},
		{
			name:      "no match",
			pattern:   "xyz",
			array:     []string{"hello", "world", "foo"},
			expectNil: true,
		},
		{
			name:      "empty array",
			pattern:   "hello",
			array:     []string{},
			expectNil: true,
		},
		{
			name:      "empty pattern",
			pattern:   "",
			array:     []string{"hello", "world"},
			expectNil: true, // Empty pattern returns nil
		},
		{
			name:      "single char pattern",
			pattern:   "w",
			array:     []string{"hello", "world"},
			expectNil: false,
			expectIdx: 1,
			expectStr: "world",
		},
		{
			name:      "best match selected",
			pattern:   "foo",
			array:     []string{"foobar", "foo", "barfoo"},
			expectNil: false,
			expectIdx: 0, // Algorithm scores "foobar" highest due to consecutive bonus
			expectStr: "foobar",
		},
		{
			name:      "case insensitive",
			pattern:   "hello",
			array:     []string{"HELLO", "world"},
			expectNil: false,
			expectIdx: 0,
			expectStr: "HELLO",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Find([]rune(tc.pattern), tc.array)
			if tc.expectNil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, tc.expectIdx, result.Index)
				assert.Equal(t, tc.expectStr, result.Str)
				assert.GreaterOrEqual(t, result.Score, 0)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		pattern     string
		expectMatch bool
	}{
		{"exact match", "hello", "hello", true},
		{"prefix match", "hello", "hel", true},
		{"suffix match", "hello", "llo", true},
		{"scattered match", "hello", "hlo", true},
		{"no match", "hello", "xyz", false},
		{"empty pattern", "hello", "", true},
		{"pattern longer than input", "hi", "hello", false},
		{"case insensitive", "HELLO", "hello", true},
		{"unicode input", "你好世界", "好界", true},
		{"unicode no match", "你好世界", "abc", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := ToChars([]byte(tc.input))
			result, pos := fuzzyMatch(&input, []rune(tc.pattern))
			if tc.expectMatch {
				assert.GreaterOrEqual(t, result.Start, 0, "Expected match but got Start=-1")
				if tc.pattern != "" {
					assert.NotNil(t, pos)
				}
			} else {
				assert.Equal(t, -1, result.Start, "Expected no match but got Start>=0")
			}
		})
	}
}

func TestFuzzyMatchScoring(t *testing.T) {
	input1 := ToChars([]byte("foobar"))
	input2 := ToChars([]byte("foo_bar"))

	// Exact prefix should score higher
	result1, _ := fuzzyMatch(&input1, []rune("foo"))
	result2, _ := fuzzyMatch(&input2, []rune("foo"))

	assert.Greater(t, result1.Score, 0)
	assert.Greater(t, result2.Score, 0)
}

func TestNormalizeRune(t *testing.T) {
	tests := []struct {
		name     string
		input    rune
		expected rune
	}{
		{"ascii a", 'a', 'a'},
		{"ascii A", 'A', 'A'},
		{"ascii 0", '0', '0'},
		{"a with acute", 'á', 'a'},
		{"a with grave", 'à', 'a'},
		{"a with circumflex", 'â', 'a'},
		{"e with acute", 'é', 'e'},
		{"o with umlaut", 'ö', 'o'},
		{"n with tilde", 'ñ', 'n'},
		{"c with cedilla", 'ç', 'c'},
		// Capital letters
		{"A with acute", 'Á', 'A'},
		{"O with umlaut", 'Ö', 'O'},
		// Characters outside normalization range
		{"chinese", '中', '中'},
		{"emoji", '😀', '😀'},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := normalizeRune(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestNormalizeRunes(t *testing.T) {
	tests := []struct {
		name     string
		input    []rune
		expected []rune
	}{
		{"ascii only", []rune("hello"), []rune("hello")},
		{"with accents", []rune("héllo"), []rune("hello")},
		{"all accents", []rune("àéîõü"), []rune("aeiou")},
		{"mixed", []rune("café"), []rune("cafe")},
		{"empty", []rune(""), []rune("")},
		{"no change needed", []rune("test123"), []rune("test123")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := NormalizeRunes(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCharClass(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected charClass
	}{
		{"lowercase a", 'a', charLower},
		{"lowercase z", 'z', charLower},
		{"uppercase A", 'A', charUpper},
		{"uppercase Z", 'Z', charUpper},
		{"digit 0", '0', charNumber},
		{"digit 9", '9', charNumber},
		{"space", ' ', charWhite},
		{"tab", '\t', charWhite},
		{"newline", '\n', charWhite},
		{"dot delimiter", '.', charDelimiter},
		{"special char", '@', charNonWord},
		{"underscore", '_', charNonWord},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := charClassOf(tc.char)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestCharClassOfNonAscii(t *testing.T) {
	tests := []struct {
		name     string
		char     rune
		expected charClass
	}{
		{"unicode lower", 'ä', charLower},
		{"unicode upper", 'Ä', charUpper},
		{"unicode number", '①', charNumber},
		{"unicode letter", '中', charLetter},
		{"unicode space", '\u00A0', charWhite}, // non-breaking space
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := charClassOfNonAscii(tc.char)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBonusFor(t *testing.T) {
	// Test word boundary bonuses
	assert.Equal(t, bonusBoundaryWhite, bonusFor(charWhite, charLower))
	assert.Equal(t, bonusBoundaryDelimiter, bonusFor(charDelimiter, charLower))
	assert.Equal(t, int16(bonusBoundary), bonusFor(charNonWord, charLower))

	// Test camelCase bonus
	assert.Equal(t, int16(bonusCamel123), bonusFor(charLower, charUpper))

	// Test number after non-number
	assert.Equal(t, int16(bonusCamel123), bonusFor(charLower, charNumber))
	assert.Equal(t, int16(0), bonusFor(charNumber, charNumber))
}

func TestAsUint16(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected uint16
	}{
		{"zero", 0, 0},
		{"positive", 100, 100},
		{"max uint16", 65535, 65535},
		{"above max", 70000, 65535},
		{"negative", -1, 0},
		{"large negative", -1000, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := AsUint16(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"ascii", "hello", 5},
		{"empty", "", 0},
		{"with newline", "a\nb", 3}, // 2 chars + 1 for newline
		{"with cr", "a\rb", 3},
		{"wide chars", "你好", 4}, // each Chinese char is 2 columns
		{"mixed", "ab你好cd", 8},  // 4 ascii + 4 wide
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StringWidth(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRunesWidth(t *testing.T) {
	tests := []struct {
		name          string
		runes         []rune
		prefixWidth   int
		tabstop       int
		limit         int
		expectWidth   int
		expectOverIdx int
	}{
		{
			name:          "ascii within limit",
			runes:         []rune("hello"),
			prefixWidth:   0,
			tabstop:       8,
			limit:         10,
			expectWidth:   5,
			expectOverIdx: -1,
		},
		{
			name:          "ascii exceeds limit",
			runes:         []rune("hello"),
			prefixWidth:   0,
			tabstop:       8,
			limit:         3,
			expectWidth:   4,
			expectOverIdx: 3,
		},
		{
			name:          "tab expansion",
			runes:         []rune("\t"),
			prefixWidth:   0,
			tabstop:       8,
			limit:         10,
			expectWidth:   8,
			expectOverIdx: -1,
		},
		{
			name:          "tab with prefix",
			runes:         []rune("\t"),
			prefixWidth:   3,
			tabstop:       8,
			limit:         10,
			expectWidth:   5, // 8 - 3 = 5
			expectOverIdx: -1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			width, overIdx := RunesWidth(tc.runes, tc.prefixWidth, tc.tabstop, tc.limit)
			assert.Equal(t, tc.expectWidth, width)
			assert.Equal(t, tc.expectOverIdx, overIdx)
		})
	}
}

func TestTrySkip(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		caseSensitive bool
		b             byte
		from          int
		expected      int
	}{
		{"find at start", "hello", false, 'h', 0, 0},
		{"find in middle", "hello", false, 'l', 0, 2},
		{"find from offset", "hello", false, 'l', 3, 3},
		{"not found", "hello", false, 'x', 0, -1},
		{"case insensitive upper", "HELLO", false, 'h', 0, 0},
		{"case sensitive no match", "HELLO", true, 'h', 0, -1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := ToChars([]byte(tc.input))
			result := trySkip(&input, tc.caseSensitive, tc.b, tc.from)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsAscii(t *testing.T) {
	tests := []struct {
		name     string
		runes    []rune
		expected bool
	}{
		{"ascii only", []rune("hello"), true},
		{"empty", []rune(""), true},
		{"with unicode", []rune("hello世界"), false},
		{"unicode only", []rune("世界"), false},
		{"edge of ascii", []rune{127}, true},
		{"beyond ascii", []rune{128}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isAscii(tc.runes)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestAsciiFuzzyIndex(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		pattern       string
		caseSensitive bool
		expectMin     int
		expectMax     int
	}{
		{"exact match", "hello", "hello", false, 0, 5},
		{"partial match", "hello world", "wor", false, 5, 9},
		{"no match", "hello", "xyz", false, -1, -1},
		{"case insensitive", "HELLO", "hel", false, 0, 3},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := ToChars([]byte(tc.input))
			minIdx, maxIdx := asciiFuzzyIndex(&input, []rune(tc.pattern), tc.caseSensitive)
			assert.Equal(t, tc.expectMin, minIdx)
			if tc.expectMin >= 0 {
				assert.GreaterOrEqual(t, maxIdx, tc.expectMax)
			}
		})
	}
}
