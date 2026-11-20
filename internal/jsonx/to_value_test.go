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

// TestToValue_Proto checks that a "__proto__" key is an own property, not
// the object's prototype.
func TestToValue_Proto(t *testing.T) {
	input := `{"__proto__": {"id": 1}, "ok": 2, "n": {"__proto__": null}, "d": {"__proto__": 1, "__proto__": 2}}`
	n, err := Parse([]byte(input))
	require.NoError(t, err)

	vm := goja.New()
	value, err := n.ToValue(vm)
	require.NoError(t, err)
	require.NoError(t, vm.Set("x", value))

	result, err := vm.RunString(`JSON.stringify([
		Object.keys(x),
		Object.getPrototypeOf(x) === Object.prototype,
		Object.getPrototypeOf(x.n) === Object.prototype,
		x.id,
		x.d,
		Object.getOwnPropertyDescriptor(x, "__proto__"),
	])`)
	require.NoError(t, err)
	require.Equal(t,
		`[["__proto__","ok","n","d"],true,true,null,{"__proto__":2},{"value":{"id":1},"writable":true,"enumerable":true,"configurable":true}]`,
		result.String())
}
