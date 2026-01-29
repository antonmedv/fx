package engine_test

import (
	"strings"
	"testing"

	"github.com/dop251/goja"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
)

// setupVM creates a new goja runtime with stdlib loaded.
func setupVM(t *testing.T) *goja.Runtime {
	var output []string
	vm := engine.NewVM(func(s string) {
		output = append(output, s)
	})
	_, err := vm.RunString(engine.Stdlib)
	require.NoError(t, err, "Failed to load stdlib")
	return vm
}

// setupVMWithOutput creates a new goja runtime with stdlib loaded and returns output slice.
func setupVMWithOutput(t *testing.T) (*goja.Runtime, *[]string) {
	output := &[]string{}
	vm := engine.NewVM(func(s string) {
		*output = append(*output, s)
	})
	_, err := vm.RunString(engine.Stdlib)
	require.NoError(t, err, "Failed to load stdlib")
	return vm, output
}

// TestApply tests the apply function.
func TestApply(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"apply function", "apply(x => x * 2, 5)", int64(10)},
		{"apply non-function", "apply(42)", int64(42)},
		{"apply with multiple args", "apply((a, b) => a + b, 2, 3)", int64(5)},
		{"apply string", "apply('hello')", "hello"},
		{"apply null", "apply(null)", nil},
		{"apply undefined", "apply(undefined)", goja.Undefined()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			if tc.expected == goja.Undefined() {
				assert.True(t, goja.IsUndefined(result))
			} else {
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestLen tests the len function.
func TestLen(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"array length", "len([1, 2, 3])", int64(3), false},
		{"empty array", "len([])", int64(0), false},
		{"string length", "len('hello')", int64(5), false},
		{"empty string", "len('')", int64(0), false},
		{"object keys count", "len({a: 1, b: 2})", int64(2), false},
		{"empty object", "len({})", int64(0), false},
		{"number error", "len(42)", nil, true},
		{"null error", "len(null)", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestUniq tests the uniq function.
func TestUniq(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"unique numbers", "uniq([1, 2, 2, 3, 3, 3])", []interface{}{int64(1), int64(2), int64(3)}, false},
		{"unique strings", "uniq(['a', 'b', 'a', 'c'])", []interface{}{"a", "b", "c"}, false},
		{"already unique", "uniq([1, 2, 3])", []interface{}{int64(1), int64(2), int64(3)}, false},
		{"empty array", "uniq([])", []interface{}{}, false},
		{"non-array error", "uniq('hello')", nil, true},
		{"number error", "uniq(42)", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestSort tests the sort function.
func TestSort(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"sort numbers", "sort([3, 1, 2])", []interface{}{int64(1), int64(2), int64(3)}, false},
		{"sort strings", "sort(['c', 'a', 'b'])", []interface{}{"a", "b", "c"}, false},
		{"empty array", "sort([])", []interface{}{}, false},
		{"already sorted", "sort([1, 2, 3])", []interface{}{int64(1), int64(2), int64(3)}, false},
		{"non-array error", "sort('hello')", nil, true},
		{"number error", "sort(42)", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestFilter tests the filter function.
func TestFilter(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"filter even", "filter(x => x % 2 === 0)([1, 2, 3, 4])", []interface{}{int64(2), int64(4)}},
		{"filter with index", "filter((x, i) => i > 0)(['a', 'b', 'c'])", []interface{}{"b", "c"}},
		{"filter all true", "filter(x => true)([1, 2, 3])", []interface{}{int64(1), int64(2), int64(3)}},
		{"filter all false", "filter(x => false)([1, 2, 3])", []interface{}{}},
		{"filter empty", "filter(x => true)([])", []interface{}{}},
		{"filter non-array truthy", "filter(x => x > 0)(5)", int64(5)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}

	// Test skip symbol for non-array falsy
	t.Run("filter non-array falsy returns skip", func(t *testing.T) {
		result, err := vm.RunString("filter(x => x > 10)(5) === skip")
		require.NoError(t, err)
		assert.Equal(t, true, result.Export())
	})

	// Test null/undefined/false filtering
	t.Run("filter removes null", func(t *testing.T) {
		result, err := vm.RunString("filter(x => null)([1, 2, 3])")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{}, result.Export())
	})

	t.Run("filter removes undefined", func(t *testing.T) {
		result, err := vm.RunString("filter(x => undefined)([1, 2, 3])")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{}, result.Export())
	})
}

// TestMap tests the map function.
func TestMap(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"map double", "map(x => x * 2)([1, 2, 3])", []interface{}{int64(2), int64(4), int64(6)}},
		{"map with index", "map((x, i) => i)(['a', 'b', 'c'])", []interface{}{int64(0), int64(1), int64(2)}},
		{"map empty", "map(x => x * 2)([])", []interface{}{}},
		{"map non-array", "map(x => x * 2)(5)", int64(10)},
		{"map to string", "map(x => String(x))([1, 2, 3])", []interface{}{"1", "2", "3"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestWalk tests the walk function.
func TestWalk(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"walk multiply primitives", "walk(x => typeof x === 'number' ? x * 2 : x)({a: 1, b: 2})",
			map[string]interface{}{"a": int64(2), "b": int64(4)}},
		{"walk nested array", "walk(x => typeof x === 'number' ? x + 1 : x)([[1, 2], [3, 4]])",
			[]interface{}{[]interface{}{int64(2), int64(3)}, []interface{}{int64(4), int64(5)}}},
		{"walk primitive", "walk(x => x * 2)(5)", int64(10)},
		{"walk with key", "walk((v, k) => k === 'double' ? v * 2 : v)({double: 5, keep: 10})",
			map[string]interface{}{"double": int64(10), "keep": int64(10)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestSortBy tests the sortBy function.
func TestSortBy(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"sortBy property", "sortBy(x => x.age)([{name: 'b', age: 30}, {name: 'a', age: 20}])",
			[]interface{}{
				map[string]interface{}{"name": "a", "age": int64(20)},
				map[string]interface{}{"name": "b", "age": int64(30)},
			}, false},
		{"sortBy computed", "sortBy(x => -x)([1, 3, 2])",
			[]interface{}{int64(3), int64(2), int64(1)}, false},
		{"sortBy string length", "sortBy(x => x.length)(['aaa', 'a', 'aa'])",
			[]interface{}{"a", "aa", "aaa"}, false},
		{"sortBy non-array error", "sortBy(x => x)('hello')", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestSortKeys tests the sortKeys function.
func TestSortKeys(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name  string
		code  string
		check func(t *testing.T, result goja.Value)
	}{
		{
			name: "sort object keys",
			code: "JSON.stringify(sortKeys({c: 1, a: 2, b: 3}))",
			check: func(t *testing.T, result goja.Value) {
				assert.Equal(t, `{"a":2,"b":3,"c":1}`, result.Export())
			},
		},
		{
			name: "sort nested object keys",
			code: "JSON.stringify(sortKeys({z: {c: 1, a: 2}, y: 3}))",
			check: func(t *testing.T, result goja.Value) {
				assert.Equal(t, `{"y":3,"z":{"a":2,"c":1}}`, result.Export())
			},
		},
		{
			name: "sort array of objects",
			code: "JSON.stringify(sortKeys([{b: 1, a: 2}, {d: 3, c: 4}]))",
			check: func(t *testing.T, result goja.Value) {
				assert.Equal(t, `[{"a":2,"b":1},{"c":4,"d":3}]`, result.Export())
			},
		},
		{
			name: "primitives unchanged",
			code: "sortKeys(42)",
			check: func(t *testing.T, result goja.Value) {
				assert.Equal(t, int64(42), result.Export())
			},
		},
		{
			name: "null unchanged",
			code: "sortKeys(null)",
			check: func(t *testing.T, result goja.Value) {
				assert.Nil(t, result.Export())
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			tc.check(t, result)
		})
	}
}

// TestGroupBy tests the groupBy function.
func TestGroupBy(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"groupBy function", "groupBy(x => x % 2 === 0 ? 'even' : 'odd')([1, 2, 3, 4])",
			map[string]interface{}{
				"odd":  []interface{}{int64(1), int64(3)},
				"even": []interface{}{int64(2), int64(4)},
			}},
		{"groupBy property name", "groupBy('type')([{type: 'a', v: 1}, {type: 'b', v: 2}, {type: 'a', v: 3}])",
			map[string]interface{}{
				"a": []interface{}{
					map[string]interface{}{"type": "a", "v": int64(1)},
					map[string]interface{}{"type": "a", "v": int64(3)},
				},
				"b": []interface{}{
					map[string]interface{}{"type": "b", "v": int64(2)},
				},
			}},
		{"groupBy empty", "groupBy(x => x)([])",
			map[string]interface{}{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestChunk tests the chunk function.
func TestChunk(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"chunk by 2", "chunk(2)([1, 2, 3, 4, 5])",
			[]interface{}{
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(3), int64(4)},
				[]interface{}{int64(5)},
			}},
		{"chunk by 3", "chunk(3)([1, 2, 3, 4, 5, 6])",
			[]interface{}{
				[]interface{}{int64(1), int64(2), int64(3)},
				[]interface{}{int64(4), int64(5), int64(6)},
			}},
		{"chunk empty", "chunk(2)([])",
			[]interface{}{}},
		{"chunk by 1", "chunk(1)([1, 2, 3])",
			[]interface{}{
				[]interface{}{int64(1)},
				[]interface{}{int64(2)},
				[]interface{}{int64(3)},
			}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestZip tests the zip function.
func TestZip(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"zip two arrays", "zip([1, 2, 3], ['a', 'b', 'c'])",
			[]interface{}{
				[]interface{}{int64(1), "a"},
				[]interface{}{int64(2), "b"},
				[]interface{}{int64(3), "c"},
			}},
		{"zip three arrays", "zip([1, 2], ['a', 'b'], [true, false])",
			[]interface{}{
				[]interface{}{int64(1), "a", true},
				[]interface{}{int64(2), "b", false},
			}},
		{"zip different lengths", "zip([1, 2, 3], ['a', 'b'])",
			[]interface{}{
				[]interface{}{int64(1), "a"},
				[]interface{}{int64(2), "b"},
			}},
		{"zip empty", "zip([], [])",
			[]interface{}{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestFlatten tests the flatten function.
func TestFlatten(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"flatten nested", "flatten([[1, 2], [3, 4]])",
			[]interface{}{int64(1), int64(2), int64(3), int64(4)}, false},
		{"flatten mixed", "flatten([[1], 2, [3, 4]])",
			[]interface{}{int64(1), int64(2), int64(3), int64(4)}, false},
		{"flatten one level", "flatten([[[1, 2]], [[3, 4]]])",
			[]interface{}{
				[]interface{}{int64(1), int64(2)},
				[]interface{}{int64(3), int64(4)},
			}, false},
		{"flatten empty", "flatten([])", []interface{}{}, false},
		{"flatten non-array error", "flatten('hello')", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestReverse tests the reverse function.
func TestReverse(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"reverse numbers", "reverse([1, 2, 3])",
			[]interface{}{int64(3), int64(2), int64(1)}, false},
		{"reverse strings", "reverse(['a', 'b', 'c'])",
			[]interface{}{"c", "b", "a"}, false},
		{"reverse empty", "reverse([])", []interface{}{}, false},
		{"reverse single", "reverse([1])", []interface{}{int64(1)}, false},
		{"reverse non-array error", "reverse('hello')", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestKeys tests the keys function.
func TestKeys(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		hasError bool
	}{
		{"object keys", "keys({a: 1, b: 2}).sort()", false},
		{"empty object", "keys({})", false},
		{"array keys", "keys([10, 20, 30])", false},
		{"null error", "keys(null)", true},
		{"number error", "keys(42)", true},
		{"string error", "keys('hello')", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result.Export())
			}
		})
	}

	// Specific value checks
	t.Run("object keys values", func(t *testing.T) {
		result, err := vm.RunString("keys({a: 1, b: 2}).sort()")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"a", "b"}, result.Export())
	})

	t.Run("array keys values", func(t *testing.T) {
		result, err := vm.RunString("keys([10, 20])")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{"0", "1"}, result.Export())
	})
}

// TestValues tests the values function.
func TestValues(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		hasError bool
	}{
		{"object values", "values({a: 1, b: 2}).sort()", false},
		{"empty object", "values({})", false},
		{"array values", "values([10, 20, 30])", false},
		{"null error", "values(null)", true},
		{"number error", "values(42)", true},
		{"string error", "values('hello')", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result.Export())
			}
		})
	}

	// Specific value checks
	t.Run("object values sorted", func(t *testing.T) {
		result, err := vm.RunString("values({a: 1, b: 2}).sort()")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{int64(1), int64(2)}, result.Export())
	})

	t.Run("array values", func(t *testing.T) {
		result, err := vm.RunString("values([10, 20, 30])")
		require.NoError(t, err)
		assert.Equal(t, []interface{}{int64(10), int64(20), int64(30)}, result.Export())
	})
}

// TestList tests the list function.
func TestList(t *testing.T) {
	vm, output := setupVMWithOutput(t)

	t.Run("list prints each item", func(t *testing.T) {
		*output = []string{} // Reset output
		result, err := vm.RunString("list([1, 2, 3]) === skip")
		require.NoError(t, err)
		assert.True(t, result.Export().(bool))
		assert.Equal(t, []string{"1", "2", "3"}, *output)
	})

	t.Run("list with objects", func(t *testing.T) {
		*output = []string{}
		result, err := vm.RunString(`list([{a: 1}]) === skip`)
		require.NoError(t, err)
		assert.True(t, result.Export().(bool))
		assert.Len(t, *output, 1)
	})

	t.Run("list non-array error", func(t *testing.T) {
		_, err := vm.RunString("list('hello')")
		assert.Error(t, err)
	})
}

// TestDel tests the del function.
func TestDel(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
		hasError bool
	}{
		{"del from object", "del('a')({a: 1, b: 2})",
			map[string]interface{}{"b": int64(2)}, false},
		{"del non-existent key", "del('c')({a: 1, b: 2})",
			map[string]interface{}{"a": int64(1), "b": int64(2)}, false},
		{"del from array", "del(1)([1, 2, 3])",
			[]interface{}{int64(1), int64(3)}, false},
		{"del first from array", "del(0)([1, 2, 3])",
			[]interface{}{int64(2), int64(3)}, false},
		{"del last from array", "del(2)([1, 2, 3])",
			[]interface{}{int64(1), int64(2)}, false},
		{"del from empty array", "del(0)([])", []interface{}{}, false},
		{"del from primitive error", "del('a')(42)", nil, true},
		{"del from null error", "del('a')(null)", nil, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}
}

// TestSkipSymbol tests the skip symbol.
func TestSkipSymbol(t *testing.T) {
	vm := setupVM(t)

	t.Run("skip is a symbol", func(t *testing.T) {
		result, err := vm.RunString("typeof skip")
		require.NoError(t, err)
		assert.Equal(t, "symbol", result.Export())
	})

	t.Run("skip is unique", func(t *testing.T) {
		result, err := vm.RunString("skip === skip")
		require.NoError(t, err)
		assert.True(t, result.Export().(bool))
	})
}

// TestConsoleLog tests console.log.
func TestConsoleLog(t *testing.T) {
	vm, output := setupVMWithOutput(t)

	tests := []struct {
		name     string
		code     string
		expected []string
	}{
		{"log string", "console.log('hello')", []string{"hello"}},
		{"log number", "console.log(42)", []string{"42"}},
		{"log object", "console.log({a: 1})", []string{"{\n  \"a\": 1\n}"}},
		{"log undefined", "console.log(undefined)", []string{"undefined"}},
		{"log multiple args", "console.log('a', 'b')", []string{"a b"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			*output = []string{}
			_, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, *output)
		})
	}
}

// TestToBase64 tests the toBase64 function.
func TestToBase64(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"encode hello", "toBase64('hello')", "aGVsbG8="},
		{"encode empty", "toBase64('')", ""},
		{"encode unicode", "toBase64('こんにちは')", "44GT44KT44Gr44Gh44Gv"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.input)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestFromBase64 tests the fromBase64 function.
func TestFromBase64(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		input    string
		expected string
		hasError bool
	}{
		{"decode hello", "fromBase64('aGVsbG8=')", "hello", false},
		{"decode empty", "fromBase64('')", "", false},
		{"decode unicode", "fromBase64('44GT44KT44Gr44Gh44Gv')", "こんにちは", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.input)
			if tc.hasError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expected, result.Export())
			}
		})
	}

	t.Run("invalid base64 error", func(t *testing.T) {
		_, err := vm.RunString("fromBase64('not-valid-base64!!!')")
		assert.Error(t, err)
	})
}

// TestYAML tests YAML.parse and YAML.stringify.
func TestYAML(t *testing.T) {
	vm := setupVM(t)

	t.Run("YAML.parse simple", func(t *testing.T) {
		result, err := vm.RunString(`YAML.parse('name: John\nage: 30')`)
		require.NoError(t, err)
		expected := map[string]interface{}{"name": "John", "age": int64(30)}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("YAML.parse array", func(t *testing.T) {
		result, err := vm.RunString(`YAML.parse('- 1\n- 2\n- 3')`)
		require.NoError(t, err)
		expected := []interface{}{int64(1), int64(2), int64(3)}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("YAML.stringify simple", func(t *testing.T) {
		result, err := vm.RunString(`YAML.stringify({name: 'John', age: 30})`)
		require.NoError(t, err)
		yamlStr := result.Export().(string)
		assert.Contains(t, yamlStr, "name: John")
		assert.Contains(t, yamlStr, "age: 30")
	})

	t.Run("YAML.stringify array", func(t *testing.T) {
		result, err := vm.RunString(`YAML.stringify([1, 2, 3])`)
		require.NoError(t, err)
		yamlStr := result.Export().(string)
		assert.Contains(t, yamlStr, "- 1")
		assert.Contains(t, yamlStr, "- 2")
		assert.Contains(t, yamlStr, "- 3")
	})

	t.Run("YAML.parse invalid", func(t *testing.T) {
		_, err := vm.RunString(`YAML.parse('invalid: [unclosed')`)
		assert.Error(t, err)
	})
}

// TestIsFalsely tests the internal isFalsely function behavior.
func TestIsFalsely(t *testing.T) {
	vm := setupVM(t)

	// isFalsely is used internally by filter
	// Testing through filter behavior

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{"false is falsely", "filter(x => false)([1])", []interface{}{}},
		{"null is falsely", "filter(x => null)([1])", []interface{}{}},
		{"undefined is falsely", "filter(x => undefined)([1])", []interface{}{}},
		{"0 is not falsely", "filter(x => 0)([1])", []interface{}{int64(1)}},
		{"empty string is not falsely", "filter(x => '')([1])", []interface{}{int64(1)}},
		{"true is not falsely", "filter(x => true)([1])", []interface{}{int64(1)}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestChainedOperations tests chaining multiple stdlib functions.
func TestChainedOperations(t *testing.T) {
	vm := setupVM(t)

	tests := []struct {
		name     string
		code     string
		expected interface{}
	}{
		{
			"filter then map",
			"map(x => x * 2)(filter(x => x > 1)([1, 2, 3]))",
			[]interface{}{int64(4), int64(6)},
		},
		{
			"map then filter",
			"filter(x => x > 3)(map(x => x * 2)([1, 2, 3]))",
			[]interface{}{int64(4), int64(6)},
		},
		{
			"sort then reverse",
			"reverse(sort([3, 1, 2]))",
			[]interface{}{int64(3), int64(2), int64(1)},
		},
		{
			"flatten then uniq",
			"uniq(flatten([[1, 2], [2, 3]]))",
			[]interface{}{int64(1), int64(2), int64(3)},
		},
		{
			"groupBy then keys",
			"keys(groupBy(x => x % 2)([1, 2, 3, 4])).sort()",
			[]interface{}{"0", "1"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := vm.RunString(tc.code)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, result.Export())
		})
	}
}

// TestEdgeCases tests various edge cases.
func TestEdgeCases(t *testing.T) {
	vm := setupVM(t)

	t.Run("deeply nested walk", func(t *testing.T) {
		result, err := vm.RunString(`
			walk(x => typeof x === 'number' ? x * 2 : x)({
				a: {
					b: {
						c: 1
					}
				}
			})
		`)
		require.NoError(t, err)
		expected := map[string]interface{}{
			"a": map[string]interface{}{
				"b": map[string]interface{}{
					"c": int64(2),
				},
			},
		}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("empty input handling", func(t *testing.T) {
		tests := []string{
			"len([])",
			"len({})",
			"len('')",
			"uniq([])",
			"sort([])",
			"flatten([])",
			"reverse([])",
			"filter(x => true)([])",
			"map(x => x)([])",
			"chunk(2)([])",
			"zip([], [])",
			"keys({})",
			"values({})",
		}
		for _, code := range tests {
			_, err := vm.RunString(code)
			assert.NoError(t, err, "Failed for: %s", code)
		}
	})

	t.Run("large array handling", func(t *testing.T) {
		result, err := vm.RunString(`
			const arr = [];
			for (let i = 0; i < 1000; i++) arr.push(i);
			len(arr)
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1000), result.Export())
	})

	t.Run("unicode strings", func(t *testing.T) {
		result, err := vm.RunString(`len('你好世界')`)
		require.NoError(t, err)
		assert.Equal(t, int64(4), result.Export())
	})
}

// TestBase64RoundTrip tests that base64 encoding/decoding is reversible.
func TestBase64RoundTrip(t *testing.T) {
	vm := setupVM(t)

	tests := []string{
		"hello",
		"",
		"Hello, 世界!",
		"Special chars: !@#$%^&*()",
		strings.Repeat("a", 1000),
	}

	for _, input := range tests {
		t.Run(input[:min(len(input), 20)], func(t *testing.T) {
			code := "fromBase64(toBase64('" + escapeJS(input) + "'))"
			result, err := vm.RunString(code)
			require.NoError(t, err)
			assert.Equal(t, input, result.Export())
		})
	}
}

func escapeJS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "'", "\\'")
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestExit tests the exit function.
func TestExit(t *testing.T) {
	vm := setupVM(t)

	t.Run("exit panics with ExitError", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r, "Expected panic from exit()")
			exitErr, ok := r.(engine.ExitError)
			require.True(t, ok, "Expected ExitError, got %T", r)
			assert.Equal(t, 42, exitErr.Code)
		}()
		_, _ = vm.RunString("exit(42)")
	})

	t.Run("exit with 0", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r)
			exitErr, ok := r.(engine.ExitError)
			require.True(t, ok)
			assert.Equal(t, 0, exitErr.Code)
		}()
		_, _ = vm.RunString("exit(0)")
	})
}

// TestSave tests the save function.
func TestSave(t *testing.T) {
	vm := setupVM(t)

	t.Run("save undefined throws error", func(t *testing.T) {
		_, err := vm.RunString("save(undefined)")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot save undefined")
	})

	t.Run("save without file path throws error", func(t *testing.T) {
		// FilePath is empty by default in tests
		_, err := vm.RunString("save({a: 1})")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "specify a file")
	})
}

// TestSortMutation tests that sort mutates the original array (JavaScript behavior).
func TestSortMutation(t *testing.T) {
	vm := setupVM(t)

	t.Run("sort mutates original array", func(t *testing.T) {
		result, err := vm.RunString(`
			const arr = [3, 1, 2];
			sort(arr);
			arr[0]
		`)
		require.NoError(t, err)
		// JavaScript sort mutates the array
		assert.Equal(t, int64(1), result.Export())
	})
}

// TestReverseMutation tests that reverse mutates the original array (JavaScript behavior).
func TestReverseMutation(t *testing.T) {
	vm := setupVM(t)

	t.Run("reverse mutates original array", func(t *testing.T) {
		result, err := vm.RunString(`
			const arr = [1, 2, 3];
			reverse(arr);
			arr[0]
		`)
		require.NoError(t, err)
		// JavaScript reverse mutates the array
		assert.Equal(t, int64(3), result.Export())
	})
}

// TestDelImmutability tests that del does not mutate the original.
func TestDelImmutability(t *testing.T) {
	vm := setupVM(t)

	t.Run("del does not mutate original object", func(t *testing.T) {
		result, err := vm.RunString(`
			const obj = {a: 1, b: 2};
			del('a')(obj);
			obj.a
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Export())
	})

	t.Run("del does not mutate original array", func(t *testing.T) {
		result, err := vm.RunString(`
			const arr = [1, 2, 3];
			del(0)(arr);
			arr[0]
		`)
		require.NoError(t, err)
		assert.Equal(t, int64(1), result.Export())
	})
}

// TestWalkWithNull tests walk behavior with null values.
func TestWalkWithNull(t *testing.T) {
	vm := setupVM(t)

	t.Run("walk handles null values", func(t *testing.T) {
		result, err := vm.RunString(`
			walk(x => x === null ? 'was null' : x)({a: null, b: 1})
		`)
		require.NoError(t, err)
		expected := map[string]interface{}{"a": "was null", "b": int64(1)}
		assert.Equal(t, expected, result.Export())
	})
}

// TestGroupByWithPrototypeKeys tests groupBy doesn't have prototype pollution issues.
func TestGroupByWithPrototypeKeys(t *testing.T) {
	vm := setupVM(t)

	t.Run("groupBy with hasOwnProperty key", func(t *testing.T) {
		result, err := vm.RunString(`
			groupBy(x => x)(['hasOwnProperty', 'toString', 'normal'])
		`)
		require.NoError(t, err)
		exported := result.Export().(map[string]interface{})
		assert.Len(t, exported["hasOwnProperty"], 1)
		assert.Len(t, exported["toString"], 1)
		assert.Len(t, exported["normal"], 1)
	})
}

// TestChunkEdgeCases tests chunk with edge case sizes.
func TestChunkEdgeCases(t *testing.T) {
	vm := setupVM(t)

	t.Run("chunk size larger than array", func(t *testing.T) {
		result, err := vm.RunString("chunk(10)([1, 2, 3])")
		require.NoError(t, err)
		expected := []interface{}{[]interface{}{int64(1), int64(2), int64(3)}}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("chunk size equals array length", func(t *testing.T) {
		result, err := vm.RunString("chunk(3)([1, 2, 3])")
		require.NoError(t, err)
		expected := []interface{}{[]interface{}{int64(1), int64(2), int64(3)}}
		assert.Equal(t, expected, result.Export())
	})
}

// TestZipEdgeCases tests zip with edge cases.
func TestZipEdgeCases(t *testing.T) {
	vm := setupVM(t)

	t.Run("zip single array", func(t *testing.T) {
		result, err := vm.RunString("zip([1, 2, 3])")
		require.NoError(t, err)
		expected := []interface{}{
			[]interface{}{int64(1)},
			[]interface{}{int64(2)},
			[]interface{}{int64(3)},
		}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("zip with one empty array", func(t *testing.T) {
		result, err := vm.RunString("zip([1, 2, 3], [])")
		require.NoError(t, err)
		expected := []interface{}{}
		assert.Equal(t, expected, result.Export())
	})
}

// TestFilterWithObjects tests filter with object predicates.
func TestFilterWithObjects(t *testing.T) {
	vm := setupVM(t)

	t.Run("filter objects by property", func(t *testing.T) {
		result, err := vm.RunString(`
			filter(x => x.active)([
				{name: 'a', active: true},
				{name: 'b', active: false},
				{name: 'c', active: true}
			])
		`)
		require.NoError(t, err)
		exported := result.Export().([]interface{})
		assert.Len(t, exported, 2)
	})
}

// TestMapWithObjects tests map transforming objects.
func TestMapWithObjects(t *testing.T) {
	vm := setupVM(t)

	t.Run("map extract property", func(t *testing.T) {
		result, err := vm.RunString(`
			map(x => x.name)([
				{name: 'a', value: 1},
				{name: 'b', value: 2}
			])
		`)
		require.NoError(t, err)
		expected := []interface{}{"a", "b"}
		assert.Equal(t, expected, result.Export())
	})

	t.Run("map transform object", func(t *testing.T) {
		result, err := vm.RunString(`
			map(x => ({...x, doubled: x.value * 2}))([
				{value: 1},
				{value: 2}
			])
		`)
		require.NoError(t, err)
		exported := result.Export().([]interface{})
		assert.Equal(t, int64(2), exported[0].(map[string]interface{})["doubled"])
		assert.Equal(t, int64(4), exported[1].(map[string]interface{})["doubled"])
	})
}

// TestSortByStability tests sortBy with equal keys.
func TestSortByStability(t *testing.T) {
	vm := setupVM(t)

	t.Run("sortBy preserves order for equal keys", func(t *testing.T) {
		// Note: JavaScript sort is not guaranteed to be stable, but this tests the behavior
		result, err := vm.RunString(`
			sortBy(x => x.group)([
				{name: 'a', group: 1},
				{name: 'b', group: 2},
				{name: 'c', group: 1}
			]).map(x => x.name)
		`)
		require.NoError(t, err)
		exported := result.Export().([]interface{})
		// Group 1 items should come before group 2
		assert.Equal(t, "b", exported[2])
	})
}

// TestNestedOperations tests deeply nested function compositions.
func TestNestedOperations(t *testing.T) {
	vm := setupVM(t)

	t.Run("complex nested operations", func(t *testing.T) {
		result, err := vm.RunString(`
			map(x => x * 2)(
				filter(x => x > 0)(
					flatten([
						[-1, 0, 1],
						[2, 3]
					])
				)
			)
		`)
		require.NoError(t, err)
		expected := []interface{}{int64(2), int64(4), int64(6)}
		assert.Equal(t, expected, result.Export())
	})
}
