package main

type node struct {
	prev, next, end *node
	depth           uint8
	key             []byte
	value           []byte
	comma           bool
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
