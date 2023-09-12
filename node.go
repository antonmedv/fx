package main

import (
	"strconv"
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
	comma           bool
	index           int
}

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

func (n *node) insertChild(child *node) {
	if n.end == nil {
		n.insertAfter(child)
	} else {
		n.end.insertAfter(child)
	}
	n.end = child
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
	if n.end == nil {
		return
	}

	n.chunk = nil

	n.next = n.end.next
	if n.next != nil {
		n.next.prev = n
	}

	n.end = nil
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

func (n *node) expandRecursively() {
	at := n
	for at != nil && at != n.end {
		at.expand()
		at = at.next
	}
}

func (n *node) findChildByKey(key string) *node {
	for at := n.next; at != nil && at != n.end; {
		k, err := strconv.Unquote(string(at.key))
		if err != nil {
			return nil
		}
		if k == key {
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
