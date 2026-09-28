package maml

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"object keeps order", "{zebra: 1, alpha: 2}", `{"zebra":1,"alpha":2}`},
		{"newline separators", "{\n  a: 1\n  b: [\n    true\n    false\n  ]\n}", `{"a":1,"b":[true,false]}`},
		{"quoted keys", `{"a b": null, 42: "x"}`, `{"a b":null,"42":"x"}`},
		{"numbers keep text", "[1.0, -0, 1e400, 12345678901234567890.5]", `[1.0,-0,1e400,12345678901234567890.5]`},
		{"escapes", `"tab\there \u{1F600} \"q\""`, `"tab\there 😀 \"q\""`},
		{"raw string", "\"\"\"\nline \"one\"\nline two\"\"\"", `"line \"one\"\nline two"`},
		{"comments dropped", "# head\n{a: 1 # tail\n}", `{"a":1}`},
		{"scalar root", "42", `42`},
		{"bom", "\xEF\xBB\xBF{}", `{}`},
		{"trailing comma", "[1, 2,]", `[1,2]`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestToJSON_Invalid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		err  string
	}{
		{"unclosed", "{a: 1", "maml: Unexpected end of input on line 1."},
		{"duplicate key", "{a: 1, a: 2}", `maml: Duplicate key "a" on line 1.`},
		{"integer overflow", "[99999999999999999999]", "maml: Integer out of range"},
		{"two roots", "1 2", "maml: Unexpected character"},
		{"empty", "", "maml: Unexpected end of input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ToJSON([]byte(tt.in))
			require.ErrorContains(t, err, tt.err)
		})
	}
}
