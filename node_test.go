package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNode_paths(t *testing.T) {
	n, err := parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, 4]}`))
	require.NoError(t, err)

	var paths []string
	var nodes []*node
	n.paths("", &paths, &nodes)
	assert.Equal(t, []string{".a", ".b", ".b.f", ".c", ".c[0]", ".c[1]"}, paths)
}

func TestNode_children(t *testing.T) {
	n, err := parse([]byte(`{"a": 1, "b": {"f": 2}, "c": [3, 4]}`))
	require.NoError(t, err)

	paths, _ := n.children()
	assert.Equal(t, []string{"a", "b", "c"}, paths)
}

func TestNode_expandRecursively(t *testing.T) {
	n, err := parse([]byte(`{"a": {"b": {"c": 1}}}`))
	require.NoError(t, err)

	n.collapseRecursively()
	n.expandRecursively(0, 3)
	assert.Equal(t, `"c"`, string(n.next.next.next.key))
}
