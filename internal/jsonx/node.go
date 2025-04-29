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
)

type Node struct {
	Prev, Next, End *Node
	directParent    *Node
	indirectParent  *Node
	Collapsed       *Node
	Depth           uint8
	Kind            Kind
	Key             []byte
	Value           []byte
	Size            int
	Chunk           []byte
	ChunkEnd        *Node
	Comma           bool
	Index           int
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

	n.Chunk = nil

	n.Next = n.ChunkEnd.Next
	if n.Next != nil {
		n.Next.Prev = n
	}

	n.ChunkEnd = nil
}

func (n *Node) HasChildren() bool {
	return n.End != nil
}

func (n *Node) Parent() *Node {
	if n.directParent == nil {
		return nil
	}
	parent := n.directParent
	if parent.indirectParent != nil {
		parent = parent.indirectParent
	}
	return parent
}

func (n *Node) Root() *Node {
	parent := n.Parent()
	for parent != nil {
		n = parent
		parent = n.Parent()
	}
	return n
}

func (n *Node) IsWrap() bool {
	return n.Value == nil && n.Chunk != nil
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

func (n *Node) FindChildByKey(key string) *Node {
	it := n.Next
	for it != nil && it != n.End {
		if it.Key != nil {
			k, err := strconv.Unquote(string(it.Key))
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

func (n *Node) paths(prefix string, paths *[]string, nodes *[]*Node) {
	it := n.Next
	for it != nil && it != n.End {
		var path string

		if it.Key != nil {
			quoted := string(it.Key)
			unquoted, err := strconv.Unquote(quoted)
			if err == nil && jsonpath.Identifier.MatchString(unquoted) {
				path = prefix + "." + unquoted
			} else {
				path = prefix + "[" + quoted + "]"
			}
		} else if it.Index >= 0 {
			path = prefix + "[" + strconv.Itoa(it.Index) + "]"
		}

		*paths = append(*paths, path)
		*nodes = append(*nodes, it)

		if it.HasChildren() {
			it.paths(path, paths, nodes)
			it = it.End.Next
		} else {
			it = it.Next
		}
	}
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
		if it.Key != nil {
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
