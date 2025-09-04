package jsonx_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func TestDeleteNode_ObjectScenarios(t *testing.T) {
	root, err := Parse([]byte(`{"a":1,"b":2,"c":3}`))
	require.NoError(t, err)
	obj := root
	require.Equal(t, Object, obj.Kind)

	// delete middle key b
	b := obj.FindByPath([]any{"b"})
	require.NotNil(t, b)
	next, ok := DeleteNode(b)
	require.True(t, ok)
	require.NotNil(t, next)
	// after deleting b, next should be c (or its start)
	c := obj.FindByPath([]any{"c"})
	require.NotNil(t, c)

	// ensure size updated and comma on previous cleared
	require.Equal(t, 2, obj.Size)

	// delete last key c -> previous comma should be cleared and selection fallback
	next2, ok := DeleteNode(c)
	require.True(t, ok)
	require.NotNil(t, next2)
	// now only {"a":1}
	require.Equal(t, 1, obj.Size)

	// delete first/only key a -> object empty
	a := obj.FindByPath([]any{"a"})
	require.NotNil(t, a)
	_, ok = DeleteNode(a)
	require.True(t, ok)
	require.Equal(t, 0, obj.Size)
}

func TestDeleteNode_ArrayScenarios(t *testing.T) {
	root, err := Parse([]byte(`[10,20,30,40]`))
	require.NoError(t, err)
	arr := root
	require.Equal(t, Array, arr.Kind)

	// delete middle index 1 (20)
	idx1 := arr.FindByPath([]any{1})
	require.NotNil(t, idx1)
	_, ok := DeleteNode(idx1)
	require.True(t, ok)
	// remaining should be [10,30,40], indices 0..2
	zero := arr.FindByPath([]any{0})
	require.NotNil(t, zero)
	require.Equal(t, "10", zero.Value)
	one := arr.FindByPath([]any{1})
	require.NotNil(t, one)
	require.Equal(t, "30", one.Value)
	two := arr.FindByPath([]any{2})
	require.NotNil(t, two)
	require.Equal(t, "40", two.Value)
	require.Equal(t, 3, arr.Size)

	// delete last element (now index 2 -> 40)
	last := arr.FindByPath([]any{2})
	require.NotNil(t, last)
	_, ok = DeleteNode(last)
	require.True(t, ok)
	require.Equal(t, 2, arr.Size)

	// delete first element (10)
	first := arr.FindByPath([]any{0})
	require.NotNil(t, first)
	_, ok = DeleteNode(first)
	require.True(t, ok)
	require.Equal(t, 1, arr.Size)
	only := arr.FindByPath([]any{0})
	require.NotNil(t, only)
	require.Equal(t, "30", only.Value)

	// delete the only element
	_, ok = DeleteNode(only)
	require.True(t, ok)
	require.Equal(t, 0, arr.Size)
}

func TestDeleteNode_EdgeCases(t *testing.T) {
	root, err := Parse([]byte(`{"k": {"x":1}, "arr":[{"y":2},3]}`))
	require.NoError(t, err)

	// try delete root -> ignored
	next, ok := DeleteNode(root)
	require.False(t, ok)
	require.Nil(t, next)

	// delete nested object {"y":2} inside arr[0]
	nested := root.FindByPath([]any{"arr", 0})
	require.NotNil(t, nested)
	_, ok = DeleteNode(nested)
	require.True(t, ok)
	// arr should become [3]
	arr := root.FindByPath([]any{"arr"})
	require.NotNil(t, arr)
	require.Equal(t, 1, arr.Size)
	el := root.FindByPath([]any{"arr", 0})
	require.NotNil(t, el)
	require.Equal(t, Number, el.Kind)
	require.Equal(t, "3", el.Value)
}
