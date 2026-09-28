package yaml

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// compact removes the whitespace goccy puts inside each JSON document.
func compact(t *testing.T, b []byte) string {
	t.Helper()
	var lines []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		var buf bytes.Buffer
		require.NoError(t, json.Compact(&buf, []byte(line)))
		lines = append(lines, buf.String())
	}
	return strings.Join(lines, "\n")
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"object keeps order", "zebra: 1\nalpha: 2\n", `{"zebra":1,"alpha":2}`},
		{"nested", "a:\n  b: [1, 2]\n  c: x\n", `{"a":{"b":[1,2],"c":"x"}}`},
		{"stream", "a: 1\n---\nb: 2\n", "{\"a\":1}\n{\"b\":2}"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON([]byte(tt.in))
			require.NoError(t, err)
			if tt.want == "" {
				require.Empty(t, got)
				return
			}
			require.Equal(t, tt.want, compact(t, got))
		})
	}
}

func TestToJSON_Invalid(t *testing.T) {
	_, err := ToJSON([]byte("a: [1"))
	require.Error(t, err)
}

func TestToJSON_Specials(t *testing.T) {
	got, err := ToJSON([]byte("a: .inf\nb: [-.Inf, .NaN, 1.5]\nc: {d: .nan}\n"))
	require.NoError(t, err)
	require.Equal(t, "{\"a\": Infinity, \"b\": [-Infinity, NaN, 1.5], \"c\": {\"d\": NaN}}\n", string(got))
}
