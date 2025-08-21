package engine_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/antonmedv/fx/internal/engine"
)

func TestQuote_BasicASCII(t *testing.T) {
	assert.Equal(t, "\"hello\"", engine.Quote("hello"))
	assert.Equal(t, "\"\"", engine.Quote(""))
	assert.Equal(t, "\"Hello, world!\"", engine.Quote("Hello, world!"))
}

func TestQuote_EscapesSpecialCharacters(t *testing.T) {
	assert.Equal(t, `"\""`, engine.Quote("\""))
	assert.Equal(t, `"\\"`, engine.Quote("\\"))
	assert.Equal(t, `"\b"`, engine.Quote("\b"))
	assert.Equal(t, `"\f"`, engine.Quote("\f"))
	assert.Equal(t, `"\n"`, engine.Quote("\n"))
	assert.Equal(t, `"\r"`, engine.Quote("\r"))
	assert.Equal(t, `"\t"`, engine.Quote("\t"))
}

func TestQuote_ControlCharactersAndDEL(t *testing.T) {
	hex4Lower := func(n int) string {
		const hexdigits = "0123456789abcdef"
		b0 := hexdigits[(n>>12)&0xF]
		b1 := hexdigits[(n>>8)&0xF]
		b2 := hexdigits[(n>>4)&0xF]
		b3 := hexdigits[n&0xF]
		return string([]byte{b0, b1, b2, b3})
	}

	// 0x00 .. 0x1F should be \uXXXX
	for b := 0; b < 0x20; b++ {
		s := string([]byte{byte(b)})
		q := engine.Quote(s)
		expected := "\"\\u" + hex4Lower(b) + "\""
		// For those with dedicated escapes, engine.Quote uses short escapes; both are valid.
		// We'll accept either short escape or \uXXXX for those particular bytes.
		switch b {
		case '\b':
			assert.Equal(t, `"\b"`, q)
		case '\f':
			assert.Equal(t, `"\f"`, q)
		case '\n':
			assert.Equal(t, `"\n"`, q)
		case '\r':
			assert.Equal(t, `"\r"`, q)
		case '\t':
			assert.Equal(t, `"\t"`, q)
		default:
			assert.Equal(t, expected, q, "byte %d", b)
		}
	}
	// 0x7F DEL
	assert.Equal(t, `"\u007f"`, engine.Quote(string([]byte{0x7F})))
}

func TestQuote_BMP_Characters_AsIs(t *testing.T) {
	// Latin-1 supplement, Cyrillic, CJK BMP characters should appear as-is
	assert.Equal(t, "\"café\"", engine.Quote("café"))
	assert.Equal(t, "\"Привет\"", engine.Quote("Привет"))
	assert.Equal(t, "\"漢字\"", engine.Quote("漢字"))
}

func TestQuote_SurrogatePairs_AsIs(t *testing.T) {
	assert.Equal(t, `"🚀"`, engine.Quote("🚀"))
	assert.Equal(t, `"👍🏻"`, engine.Quote("👍🏻"))
	assert.Equal(t, `"𝄞"`, engine.Quote("𝄞"))
}

func TestQuote_InvalidUTF8BytesAreEscaped(t *testing.T) {
	// Construct a string with invalid UTF-8 byte 0xFF and 0xC0 (overlong lead)
	s := string([]byte{'A', 0xFF, 'B', 0xC0, 'C'})
	got := engine.Quote(s)
	// Expect bytes to be escaped as \u00xx in lowercase hex
	want := `"A\u00ffB\u00c0C"`
	assert.Equal(t, want, got)
}

func TestQuote_JSONRoundTrip_ValidUTF8(t *testing.T) {
	tests := []struct{ input string }{
		{""},
		{"simple"},
		{"line\nfeed"},
		{"tab\tchar"},
		{"quote \" here"},
		{"backslash \\"},
		{"café"},
		{"Привет"},
		{"漢字"},
		{"emoji 🚀"},
		{"mix: \b\f\n\r\t and \u007F:" + string([]byte{0x7F})},
		{"Line1\n\t\"Quote\" and backslash \\ and DEL:" + string([]byte{0x7F}) + " and emoji 🚀"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			q := engine.Quote(tt.input)
			var v string
			err := json.Unmarshal([]byte(q), &v)
			assert.NoError(t, err, "failed to unmarshal: %q", q)
			assert.Equal(t, tt.input, v)
		})
	}
}
