package main

type node struct {
	prev, next, end *node
	directParent    *node
	indirectParent  *node
	collapsed       *node
	depth           uint8
	key             []byte
	value           []byte
	comma           bool
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

func (n *node) collapseThisOrParent() *node {
	if n.end == nil || n.isCollapsed() {
		if n.parent() != nil {
			return n.parent().collapseThisOrParent()
		}
		return n
	}
	return n.collapse()
}

func (n *node) collapse() *node {
	if n.end != nil {
		n.collapsed = n.next
		n.next = n.end.next
		if n.next != nil {
			n.next.prev = n
		}
	}
	return n
}

func (n *node) isCollapsed() bool {
	return n.collapsed != nil
}

func (n *node) expand() {
	if n.collapsed != nil {
		n.next.prev = n.end
		n.next = n.collapsed
		n.collapsed = nil
	}
}

func (n *node) atEnd() bool {
	return n.next == nil
}
