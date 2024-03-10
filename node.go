package main

import (
	"strconv"

	jsonpath "github.com/antonmedv/fx/path"
)

type node struct {
	prev, next, end *node
	directParent    *node
	indirectParent  *node
	collapsed       *node
	depth           uint8
	key             []byte
	value           []byte
	chunk           []byte
	chunkEnd        *node
	comma           bool
	index           int
}

// append ands a node as a child to the current node (body of {...} or [...]).
func (n *node) append(child *node) {
	if n.end == nil {
		n.end = n
	}
	n.end.next = child
	child.prev = n.end
	if child.end == nil {
		n.end = child
	} else {
		n.end = child.end
	}
}

// adjacent adds a node as a sibling to the current node ({}{}{} or [][][]).
func (n *node) adjacent(child *node) {
	end := n.end
	if end == nil {
		end = n
	}
	end.next = child
	child.prev = end
}

func (n *node) insertChunk(chunk *node) {
	if n.chunkEnd == nil {
		n.insertAfter(chunk)
	} else {
		n.chunkEnd.insertAfter(chunk)
	}
	n.chunkEnd = chunk
}

func (n *node) insertAfter(child *node) {
	if n.next == nil {
		n.next = child
		child.prev = n
	} else {
		old := n.next
		n.next = child
		child.prev = n
		child.next = old
		old.prev = child
	}
}

func (n *node) dropChunks() {
	if n.chunkEnd == nil {
		return
	}

	n.chunk = nil

	n.next = n.chunkEnd.next
	if n.next != nil {
		n.next.prev = n
	}

	n.chunkEnd = nil
}

func (n *node) hasChildren() bool {
	return n.end != nil
}

func (n *node) parent() *node {
	if n.directParent == nil {
		return nil
	}
	parent := n.directParent
	if parent.indirectParent != nil {
		parent = parent.indirectParent
	}
	return parent
}

func (n *node) isCollapsed() bool {
	return n.collapsed != nil
}

func (n *node) collapse() *node {
	if n.end != nil && !n.isCollapsed() {
		n.collapsed = n.next
		n.next = n.end.next
		if n.next != nil {
			n.next.prev = n
		}
	}
	return n
}

func (n *node) collapseRecursively() {
	var at *node
	if n.isCollapsed() {
		at = n.collapsed
	} else {
		at = n.next
	}
	for at != nil && at != n.end {
		if at.hasChildren() {
			at.collapseRecursively()
			at.collapse()
		}
		at = at.next
	}
}

func (n *node) expand() {
	if n.isCollapsed() {
		if n.next != nil {
			n.next.prev = n.end
		}
		n.next = n.collapsed
		n.collapsed = nil
	}
}

func (n *node) expandRecursively(level, maxLevel int) {
	if level >= maxLevel {
		return
	}
	if n.isCollapsed() {
		n.expand()
	}
	it := n.next
	for it != nil && it != n.end {
		if it.hasChildren() {
			it.expandRecursively(level+1, maxLevel)
			it = it.end.next
		} else {
			it = it.next
		}
	}
}

func (n *node) findChildByKey(key string) *node {
	it := n.next
	for it != nil && it != n.end {
		if it.key != nil {
			k, err := strconv.Unquote(string(it.key))
			if err != nil {
				return nil
			}
			if k == key {
				return it
			}
		}
		if it.chunkEnd != nil {
			it = it.chunkEnd.next
		} else if it.end != nil {
			it = it.end.next
		} else {
			it = it.next
		}
	}
	return nil
}

func (n *node) findChildByIndex(index int) *node {
	for at := n.next; at != nil && at != n.end; {
		if at.index == index {
			return at
		}
		if at.end != nil {
			at = at.end.next
		} else {
			at = at.next
		}
	}
	return nil
}

func (n *node) paths(prefix string, paths *[]string, nodes *[]*node) {
	it := n.next
	for it != nil && it != n.end {
		var path string

		if it.key != nil {
			quoted := string(it.key)
			unquoted, err := strconv.Unquote(quoted)
			if err == nil && jsonpath.Identifier.MatchString(unquoted) {
				path = prefix + "." + unquoted
			} else {
				path = prefix + "[" + quoted + "]"
			}
		} else if it.index >= 0 {
			path = prefix + "[" + strconv.Itoa(it.index) + "]"
		}

		*paths = append(*paths, path)
		*nodes = append(*nodes, it)

		if it.hasChildren() {
			it.paths(path, paths, nodes)
			it = it.end.next
		} else {
			it = it.next
		}
	}
}

func (n *node) children() ([]string, []*node) {
	if !n.hasChildren() {
		return nil, nil
	}

	var paths []string
	var nodes []*node

	var it *node
	if n.isCollapsed() {
		it = n.collapsed
	} else {
		it = n.next
	}

	for it != nil && it != n.end {
		if it.key != nil {
			key := string(it.key)
			unquoted, err := strconv.Unquote(key)
			if err == nil {
				key = unquoted
			}
			paths = append(paths, key)
			nodes = append(nodes, it)
		}

		if it.hasChildren() {
			it = it.end.next
		} else {
			it = it.next
		}
	}

	return paths, nodes
}
