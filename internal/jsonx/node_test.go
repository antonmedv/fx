package jsonx

import (
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	assert.Equal(t, `"c"`, n.Next.Next.Next.Key)
}

func TestNode_Paths(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, {"d": 4}]}`))
	require.NoError(t, err)

	paths := make([]string, 0, 10)
	nodes := make([]*Node, 0, 10)
	n.Paths(&paths, &nodes)
	assert.Equal(t, []string{
		".a",
		".b",
		".c",
		".b.f",
		".c[0]",
		".c[1]",
		".c[1].d",
	}, paths)
}

func TestNode_Paths_Collapsed(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, {"d": 4}]}`))
	require.NoError(t, err)
	n.CollapseRecursively()

	paths := make([]string, 0, 10)
	nodes := make([]*Node, 0, 10)
	n.Paths(&paths, &nodes)
	assert.Equal(t, []string{
		".a",
		".b",
		".c",
		".b.f",
		".c[0]",
		".c[1]",
		".c[1].d",
	}, paths)
}

func TestNode_ForEach(t *testing.T) {
	n, err := Parse([]byte(`{"a": 1, "b": 2, "c": 3}`))
	require.NoError(t, err)

	var keys []string
	n.ForEach(func(node *Node) {
		if k, err := strconv.Unquote(node.Key); err == nil {
			keys = append(keys, k)
		}
	})
	assert.Equal(t, []string{"a", "b", "c"}, keys)
}

func TestNode_ForEach_Empty(t *testing.T) {
	n, err := Parse([]byte(`{}`))
	require.NoError(t, err)

	called := false
	n.ForEach(func(node *Node) {
		called = true
	})
	assert.False(t, called)
}

func TestNode_ForEach_SkipsNested(t *testing.T) {
	n, err := Parse([]byte(`{"a": {"b": 1}, "c": [2, {"d": 3}]}`))
	require.NoError(t, err)

	var keys []string
	n.ForEach(func(node *Node) {
		if k, err := strconv.Unquote(node.Key); err == nil {
			keys = append(keys, k)
		}
	})
	assert.Equal(t, []string{"a", "c"}, keys)
}

func TestNode_AdjacentKeepsWrapChunks(t *testing.T) {
	n, err := Parse([]byte(`"` + strings.Repeat("word ", 30) + `"`))
	require.NoError(t, err)
	Wrap(n, 20)
	require.NotNil(t, n.ChunkEnd)
	chunkEnd := n.ChunkEnd

	next, err := Parse([]byte(`1`))
	require.NoError(t, err)
	n.Adjacent(next)

	require.Same(t, next, chunkEnd.Next)
	require.Same(t, chunkEnd, next.Prev)
	require.NotSame(t, next, n.Next, "chunks must stay between the string and the next document")
}

// A key Go can't unquote (JSON's \/ escape) used to loop forever.
func TestNode_FindByPathSkipsKeyGoCantUnquote(t *testing.T) {
	n, err := Parse([]byte(`{"a\/b": 1, "c": 2}`))
	require.NoError(t, err)
	c := n.FindByPath([]any{"c"})
	require.NotNil(t, c)
	require.Equal(t, "2", c.Value)
}
