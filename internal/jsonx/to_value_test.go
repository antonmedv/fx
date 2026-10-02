package jsonx

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/require"
)

func TestToValue_WrappedAndCollapsed(t *testing.T) {
	long := strings.Repeat("word ", 30)
	input := `{"s": "` + long + `", "o": {"t": "` + long + `"}, "a": ["` + long + `", 1]}`
	n, err := Parse([]byte(input))
	require.NoError(t, err)

	Wrap(n, 20)
	collapsed := false
	for it := n.Next; it != nil; it = it.Next {
		if it.Key == `"o"` {
			it.Collapse()
			collapsed = true
		}
	}
	require.True(t, collapsed)

	vm := goja.New()
	value, err := n.ToValue(vm)
	require.NoError(t, err)
	got, err := json.Marshal(value.Export())
	require.NoError(t, err)

	var want, have any
	require.NoError(t, json.Unmarshal([]byte(input), &want))
	require.NoError(t, json.Unmarshal(got, &have))
	require.Equal(t, want, have)
}
