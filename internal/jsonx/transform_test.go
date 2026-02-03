package jsonx

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReplaceNodeAndSerialize(t *testing.T) {
	root, err := Parse([]byte(`{"args":"{\"requestId\":\"746089c3b583ebb2f58a687cdaf925a8\",\"content\":\"123\"}","cost":16}`))
	require.NoError(t, err)
	args := root.findChildByKey("args")
	require.NotNil(t, args)
	require.Equal(t, String, args.Kind)
	require.True(t, args.Comma, "string node should keep trailing comma")

	parsed, err := Parse([]byte(`{"requestId":"746089c3b583ebb2f58a687cdaf925a8","content":"123"}`))
	require.NoError(t, err)

	ReplaceNode(args, parsed)

	require.Equal(t, Object, args.Kind)
	require.True(t, args.HasChildren())
	require.NotNil(t, args.End)
	require.True(t, args.End.Comma, "closing bracket should inherit comma")

	child := args.findChildByKey("requestId")
	require.NotNil(t, child)
	require.Equal(t, String, child.Kind)

	next := args.End.Next
	require.NotNil(t, next)
	unquoted := strings.Trim(next.Key, "\"")
	require.Equal(t, "cost", unquoted)

	serialized := SerializeNode(args)
	require.Equal(t, `{"requestId":"746089c3b583ebb2f58a687cdaf925a8","content":"123"}`, serialized)
}

func TestClearChildren(t *testing.T) {
	root, err := Parse([]byte(`{"args":{"a":1,"b":2},"cost":16}`))
	require.NoError(t, err)
	args := root.findChildByKey("args")
	require.NotNil(t, args)
	require.Equal(t, Object, args.Kind)
	require.True(t, args.HasChildren())
	require.True(t, args.End.Comma)

	comma := ClearChildren(args)
	require.True(t, comma)
	require.False(t, args.HasChildren())
	require.Nil(t, args.End)
	next := args.Next
	require.NotNil(t, next)
	unquoted := strings.Trim(next.Key, "\"")
	require.Equal(t, "cost", unquoted)
}
