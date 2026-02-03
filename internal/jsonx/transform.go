package jsonx

import "strings"

// ReplaceNode copies the structure of parsed into target, reusing the target
// pointer while splicing parsed's children into the surrounding linked list.
// It preserves the target's key/index/parent relationships and trailing comma.
func ReplaceNode(target *Node, parsed *Node) {
	if target == nil || parsed == nil {
		return
	}

	comma := trailingComma(target)
	offset := int(target.Depth) - int(parsed.Depth)

	// Remove any wrapped chunks on the target so Next points to the next sibling.
	target.dropChunks()

	// Detach children/subtrees that may currently hang off the target.
	s := target.Next
	if target.HasChildren() && !target.IsCollapsed() {
		if target.End != nil {
			s = target.End.Next
		}
	}
	target.Next = s
	if s != nil {
		s.Prev = target
	}
	if target.IsCollapsed() {
		target.Collapsed = nil
	}
	target.End = nil
	target.Size = parsed.Size

	// Copy primitive fields from parsed.
	target.Kind = parsed.Kind
	target.Value = parsed.Value

	target.Comma = false

	if !parsed.HasChildren() {
		// Primitive replacement.
		target.End = nil
		if target.Next != nil {
			target.Next.Prev = target
		}
		target.Comma = comma
		return
	}

	first := parsed.Next
	last := parsed.End
	if first == nil || last == nil {
		// Safety: fall back to primitive replacement semantics.
		target.Comma = comma
		return
	}

	// Connect the new child chain after the target and before the old sibling.
	after := target.Next
	target.Next = first
	first.Prev = target
	last.Next = after
	if after != nil {
		after.Prev = last
	}
	target.End = last
	parsed.Next = nil
	parsed.End = nil

	reassignParentDepth(first, last, target, offset)

	last.Comma = comma
}

// ClearChildren removes all children from node and returns whether the
// original subtree ended with a trailing comma.
func ClearChildren(node *Node) bool {
	if node == nil {
		return false
	}
	comma := trailingComma(node)
	if !node.HasChildren() {
		node.Collapsed = nil
		node.End = nil
		node.Size = 0
		return comma
	}
	if node.IsCollapsed() {
		node.Collapsed = nil
		node.End = nil
		node.Size = 0
		return comma
	}
	start := node.Next
	end := node.End
	if start == nil || end == nil {
		node.End = nil
		node.Size = 0
		return comma
	}
	after := end.Next
	node.Next = after
	if after != nil {
		after.Prev = node
	}
	node.End = nil
	node.Size = 0
	return comma
}

// SerializeNode returns the JSON string representation of node's subtree.
func SerializeNode(node *Node) string {
	if node == nil {
		return ""
	}
	switch node.Kind {
	case Object:
		if !node.HasChildren() {
			return node.Value
		}
		builder := new(strings.Builder)
		builder.WriteString("{")
		writeObjectChildren(builder, node)
		builder.WriteString("}")
		return builder.String()
	case Array:
		if !node.HasChildren() {
			return node.Value
		}
		builder := new(strings.Builder)
		builder.WriteString("[")
		writeArrayChildren(builder, node)
		builder.WriteString("]")
		return builder.String()
	default:
		if node.Value != "" {
			return node.Value
		}
		return node.Chunk
	}
}

func writeObjectChildren(builder *strings.Builder, parent *Node) {
	first := true
	for child := firstChild(parent); child != nil && child != parent.End; child = nextSibling(child) {
		if child.IsWrap() {
			continue
		}
		if !first {
			builder.WriteString(",")
		}
		first = false
		builder.WriteString(child.Key)
		builder.WriteString(":")
		builder.WriteString(SerializeNode(child))
	}
}

func writeArrayChildren(builder *strings.Builder, parent *Node) {
	first := true
	for child := firstChild(parent); child != nil && child != parent.End; child = nextSibling(child) {
		if child.IsWrap() {
			continue
		}
		if !first {
			builder.WriteString(",")
		}
		first = false
		builder.WriteString(SerializeNode(child))
	}
}

func firstChild(parent *Node) *Node {
	if parent == nil || !parent.HasChildren() {
		return nil
	}
	if parent.IsCollapsed() {
		return parent.Collapsed
	}
	return parent.Next
}

func nextSibling(node *Node) *Node {
	if node == nil {
		return nil
	}
	if node.HasChildren() && node.End != nil {
		return node.End.Next
	}
	if node.ChunkEnd != nil {
		return node.ChunkEnd.Next
	}
	return node.Next
}

func reassignParentDepth(start *Node, end *Node, parent *Node, offset int) {
	if start == nil {
		return
	}
	for node := start; node != nil; {
		node.Parent = parent
		newDepth := int(node.Depth) + offset
		if newDepth < 0 {
			newDepth = 0
		}
		if newDepth > 255 {
			newDepth = 255
		}
		node.Depth = uint8(newDepth)
		var next *Node
		if node.HasChildren() && node.End != nil {
			childOffset := offset
			reassignParentDepth(node.Next, node.End, node, childOffset)
			next = node.End.Next
		} else {
			next = node.Next
		}
		if node == end {
			break
		}
		node = next
	}
}

func trailingComma(node *Node) bool {
	if node == nil {
		return false
	}
	if node.HasChildren() && node.End != nil {
		return node.End.Comma
	}
	return node.Comma
}
