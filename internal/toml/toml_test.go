package toml

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

// helper: compare JSON as exact bytes
func assertJSONBytesEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	require.Equal(t, want, string(got), "unexpected JSON")
}

// helper: compare JSON by unmarshalling into interface{} (order-insensitive for objects)
func assertJSONStructEqual(t *testing.T, got []byte, want string) {
	t.Helper()
	var g any
	var w any
	require.NoErrorf(t, json.Unmarshal(got, &g), "json.Unmarshal(got) failed; json: %s", string(got))
	require.NoErrorf(t, json.Unmarshal([]byte(want), &w), "json.Unmarshal(want) failed; json: %s", want)
	if !reflect.DeepEqual(g, w) {
		gb, _ := json.Marshal(g)
		wb, _ := json.Marshal(w)
		require.Equal(t, string(wb), string(gb), "JSON structures differ")
	}
}

func TestToJSON_SimpleScalars(t *testing.T) {
	in := []byte(`a = 1
b = "x"
c = true
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	assertJSONBytesEqual(t, got, `{"a":1,"b":"x","c":true}`)
}

func TestToJSON_NestedTable(t *testing.T) {
	in := []byte(`[a]
b = 1
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	assertJSONBytesEqual(t, got, `{"a":{"b":1}}`)
}

func TestToJSON_DottedKeys(t *testing.T) {
	in := []byte(`a.b.c = 2
a.b.d = 3
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	assertJSONBytesEqual(t, got, `{"a":{"b":{"c":2,"d":3}}}`)
}

func TestToJSON_ArraysAndInlineTables(t *testing.T) {
	in := []byte(`arr = [1, 2, "x"]
obj = { b = 1, c = 2 }
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	// Inline table field order is not guaranteed; compare structurally
	assertJSONStructEqual(t, got, `{"arr":[1,2,"x"],"obj":{"b":1,"c":2}}`)
}

func TestToJSON_ArraysOfTables(t *testing.T) {
	in := []byte(`[[fruit]]
name = "apple"
[fruit.variety]
name = "red delicious"

[[fruit]]
name = "banana"
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	assertJSONStructEqual(t, got, `{
		"fruit": [
			{"name":"apple","variety":{"name":"red delicious"}},
			{"name":"banana"}
		]
	}`)
}

func TestToJSON_NestedAOT(t *testing.T) {
	in := []byte(`[a]
[[a.b]]
x = 1
[[a.b]]
x = 2
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	assertJSONStructEqual(t, got, `{"a":{"b":[{"x":1},{"x":2}]}}`)
}

func TestToJSON_MixedTablesAndKeysOrder(t *testing.T) {
	in := []byte(`title = "TOML Example"

[owner]
name = "Tom"
age = 42

[database]
server = "192.168.1.1"
ports = [ 8001, 8001, 8002 ]
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	// Expect compact JSON and correct nesting; top-level key order preserved
	assertJSONBytesEqual(t, got, `{"title":"TOML Example","owner":{"name":"Tom","age":42},"database":{"server":"192.168.1.1","ports":[8001,8001,8002]}}`)
}

func TestToJSON_Datetime(t *testing.T) {
	in := []byte(`dt = 1979-05-27T07:32:00Z
`)
	got, err := ToJSON(in)
	require.NoError(t, err)
	// Marshal time values usually as RFC3339 strings
	var obj map[string]any
	require.NoError(t, json.Unmarshal(got, &obj))
	v, ok := obj["dt"].(string)
	require.Truef(t, ok && v != "", "expected dt to be a JSON string, got: %T %v", obj["dt"], obj["dt"])
}
