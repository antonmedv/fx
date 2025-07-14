package jsonx

import (
	"strconv"

	"github.com/antonmedv/fx/internal/jsonpath"
)

type Kind byte

const (
	Err Kind = iota
	Null
	Bool
	Number
	String
	Object
	Array
	NaN
	Infinity
	Undefined
)

type Node struct {
	Prev, Next, End *Node
	Parent          *Node
	Collapsed       *Node
	Depth           uint8
	Kind            Kind
	Key             string
	Value           string
	Size            int
	Chunk           string
	ChunkEnd        *Node
	Comma           bool
	Index           int
	LineNumber      int
}

// Append ands a node as a child to the current node (body of {...} or [...]).
func (n *Node) Append(child *Node) {
	if n.End == nil {
		n.End = n
	}
	n.End.Next = child
	child.Prev = n.End
	if child.End == nil {
		n.End = child
	} else {
		n.End = child.End
	}
}

// Adjacent adds a node as a sibling to the current node ({}{}{} or [][][]).
func (n *Node) Adjacent(child *Node) {
	end := n.End
	if end == nil {
		end = n
	}
	end.Next = child
	child.Prev = end
	if n.IsCollapsed() {
		// Also attach to collapsed node.
		n.Next = child
		child.Prev = n
	}
}

func (n *Node) insertChunk(chunk *Node) {
	if n.ChunkEnd == nil {
		n.insertAfter(chunk)
	} else {
		n.ChunkEnd.insertAfter(chunk)
	}
	n.ChunkEnd = chunk
}

func (n *Node) insertAfter(child *Node) {
	if n.Next == nil {
		n.Next = child
		child.Prev = n
	} else {
		old := n.Next
		n.Next = child
		child.Prev = n
		child.Next = old
		old.Prev = child
	}
}

func (n *Node) dropChunks() {
	if n.ChunkEnd == nil {
		return
	}

	n.Chunk = ""

	n.Next = n.ChunkEnd.Next
	if n.Next != nil {
		n.Next.Prev = n
	}

	n.ChunkEnd = nil
}

func (n *Node) HasChildren() bool {
	return n.End != nil
}

func (n *Node) Root() *Node {
	parent := n.Parent
	for parent != nil {
		n = parent
		parent = n.Parent
	}
	return n
}

func (n *Node) IsWrap() bool {
	return n.Value == "" && n.Chunk != ""
}

func (n *Node) IsCollapsed() bool {
	return n.Collapsed != nil
}

func (n *Node) Collapse() *Node {
	if n.End != nil && !n.IsCollapsed() {
		n.Collapsed = n.Next
		n.Next = n.End.Next
		if n.Next != nil {
			n.Next.Prev = n
		}
	}
	return n
}

func (n *Node) CollapseRecursively() {
	var at *Node
	if n.IsCollapsed() {
		at = n.Collapsed
	} else {
		at = n.Next
	}
	for at != nil && at != n.End {
		if at.HasChildren() {
			at.CollapseRecursively()
			at.Collapse()
		}
		at = at.Next
	}
}

func (n *Node) Expand() {
	if n.IsCollapsed() {
		if n.Next != nil {
			n.Next.Prev = n.End
		}
		n.Next = n.Collapsed
		n.Collapsed = nil
	}
}

func (n *Node) ExpandRecursively(level, maxLevel int) {
	if level >= maxLevel {
		return
	}
	if n.IsCollapsed() {
		n.Expand()
	}
	it := n.Next
	for it != nil && it != n.End {
		if it.HasChildren() {
			it.ExpandRecursively(level+1, maxLevel)
			it = it.End.Next
		} else {
			it = it.Next
		}
	}
}

func (n *Node) FindByPath(path []any) *Node {
	it := n
	for _, part := range path {
		if it == nil {
			return nil
		}
		switch part := part.(type) {
		case string:
			it = it.FindChildByKey(part)
		case int:
			it = it.FindChildByIndex(part)
		}
	}
	return it
}

func (n *Node) FindChildByKey(key string) *Node {
	it := n.Next
	for it != nil && it != n.End {
		if it.Key != "" {
			k, err := strconv.Unquote(it.Key)
			if err != nil {
				return nil
			}
			if k == key {
				return it
			}
		}
		if it.ChunkEnd != nil {
			it = it.ChunkEnd.Next
		} else if it.End != nil {
			it = it.End.Next
		} else {
			it = it.Next
		}
	}
	return nil
}

func (n *Node) FindChildByIndex(index int) *Node {
	for at := n.Next; at != nil && at != n.End; {
		if at.Index == index {
			return at
		}
		if at.End != nil {
			at = at.End.Next
		} else {
			at = at.Next
		}
	}
	return nil
}

func (n *Node) FindNextNonErr() *Node {
	it := n
	for it != nil && it.Kind == Err {
		it = it.Next
	}
	return it
}

func (n *Node) Children() ([]string, []*Node) {
	if !n.HasChildren() {
		return nil, nil
	}

	var paths []string
	var nodes []*Node

	var it *Node
	if n.IsCollapsed() {
		it = n.Collapsed
	} else {
		it = n.Next
	}

	for it != nil && it != n.End {
		if it.Key != "" {
			key := string(it.Key)
			unquoted, err := strconv.Unquote(key)
			if err == nil {
				key = unquoted
			}
			paths = append(paths, key)
			nodes = append(nodes, it)
		}

		if it.HasChildren() {
			it = it.End.Next
		} else {
			it = it.Next
		}
	}

	return paths, nodes
}

func (n *Node) Bottom() *Node {
	it := n
	for it.Next != nil {
		if it.End != nil {
			it = it.End
		} else {
			it = it.Next
		}
	}
	return it
}

func (n *Node) Paths(paths *[]string, nodes *[]*Node) {
	joinPath := func(prefix string, n *Node) string {
		var path string
		if n.Key != "" {
			quoted := n.Key
			unquoted, err := strconv.Unquote(quoted)
			if err == nil && jsonpath.Identifier.MatchString(unquoted) {
				path = prefix + "." + unquoted
			} else {
				path = prefix + "[" + quoted + "]"
			}
		} else if n.Index >= 0 {
			path = prefix + "[" + strconv.Itoa(n.Index) + "]"
		}
		return path
	}

	type item struct {
		node *Node
		path string
	}
	var queue []item
	queue = append(queue, item{node: n, path: ""})

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		it := curr.node
		prefix := curr.path

		if it.IsCollapsed() {
			it = it.Collapsed
		} else {
			it = it.Next
		}

		for it != nil && it != curr.node.End {
			path := joinPath(prefix, it)
			if path != "" {
				if len(*paths) == cap(*paths) {
					return
				}
				*paths = append(*paths, path)
				*nodes = append(*nodes, it)
			}

			if it.HasChildren() {
				queue = append(queue, item{node: it, path: path})
				it = it.End.Next
			} else {
				it = it.Next
			}
		}
	}
}
