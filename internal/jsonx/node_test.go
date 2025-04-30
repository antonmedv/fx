package jsonx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_paths(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, 4]}`))
	require.NoError(t, err)

	var paths []string
	var nodes []*Node
	n.paths("", &paths, &nodes)
	assert.Equal(t, []string{".a", ".b", ".b.f", ".c", ".c[0]", ".c[1]"}, paths)
}

func TestNode_children(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, 4]}`))
	require.NoError(t, err)

	paths, _ := n.Children()
	assert.Equal(t, []string{"a", "b", "c"}, paths)
}

func TestNode_expandRecursively(t *testing.T) {
	n, err := Parse([]byte(`{"a": {"b": {"c": 1}}}`))
	require.NoError(t, err)

	n.CollapseRecursively()
	n.ExpandRecursively(0, 3)
	assert.Equal(t, `"c"`, string(n.Next.Next.Next.Key))
}

func TestNode_Symbols(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, {"d": 4}]}`))
	require.NoError(t, err)

	paths := make([]string, 0, 10)
	nodes := make([]*Node, 0, 10)
	n.Symbols("", &paths, &nodes)
	assert.Equal(t, []string{".a", ".b", ".b.f", ".c", ".c.0", ".c.1", ".c.1.d"}, paths)
}
