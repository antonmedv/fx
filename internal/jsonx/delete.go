package jsonx

// DeleteNode removes the node at from the linked structure and returns a node to select next.
// It returns (nextToSelect, true) if deletion happened, or (nil, false) if nothing was deleted.
// Rules:
// - Do nothing if at is nil, points to root (no parent), or is a bracket/closing node (Index == -1).
// - If at is a wrap placeholder (Chunk set, Value empty), operate on its parent value.
// - Maintain Prev/Next links skipping the deleted range [at..endOf].
// - Clear trailing comma on the previous sibling when deleting the last child before parent's End.
// - Decrement parent.Size and reindex subsequent array siblings.
// - Choose selection: prefer next; if next is nil or parent.End, prefer prev; else parent.
func DeleteNode(at *Node) (*Node, bool) {
	if at == nil {
		return nil, false
	}
	// Avoid closing bracket nodes (Index == -1 used for brackets)
	if at.Index == -1 {
		return nil, false
	}
	parent := at.Parent
	if parent == nil { // avoid deleting root
		return nil, false
	}
	// If current points to a wrap placeholder, move to its parent value
	if at.Chunk != "" && at.Value == "" && at.Parent != nil {
		at = at.Parent
		parent = at.Parent
		if parent == nil {
			return nil, false
		}
	}

	// Determine the last node of this item (to skip its subtree or chunks)
	endOf := at
	if at.End != nil {
		endOf = at.End
	} else if at.ChunkEnd != nil {
		endOf = at.ChunkEnd
	}
	prev := at.Prev
	next := endOf.Next

	// If deleting the last child before parent's closing bracket, clear trailing comma on previous sibling
	isLast := next == parent.End
	if isLast && prev != nil && prev != parent {
		prev.Comma = false
	}

	// Relink to remove [at..endOf] from the chain
	if prev != nil {
		prev.Next = next
	}
	if next != nil {
		next.Prev = prev
	}

	// Update parent size and array indices if needed
	if parent.Size > 0 {
		parent.Size--
	}
	if parent.Kind == Array {
		for it := next; it != nil && it != parent.End; {
			if it.Parent == parent && it.Index >= 0 {
				it.Index = it.Index - 1
			}
			if it.HasChildren() {
				it = it.End.Next
			} else {
				it = it.Next
			}
		}
	}

	// Select a sensible node after deletion
	selectTo := next
	if selectTo == nil || selectTo == parent.End {
		if prev != nil && prev != parent {
			selectTo = prev
		} else {
			selectTo = parent
		}
	}
	return selectTo, true
}
