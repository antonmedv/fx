package main

import (
	. "github.com/antonmedv/fx/internal/jsonx"
)

func gotoLine(m *model, num int) {
	m.selectNode(findNode(m, num))

	m.commandInput.SetValue("")
	m.recordHistory()
}

func findNode(m *model, line int) *Node {
	if m.top == nil {
		return nil
	}

	if line >= m.totalLines {
		return m.top.Bottom()
	}

	if line <= 1 {
		return m.top
	}

	node := m.top

	for {
		if node.ChunkEnd != nil {
			node = node.ChunkEnd.Next
		} else if node.Collapsed != nil {
			node = node.Collapsed
		} else {
			node = node.Next
		}

		if node == nil {
			return nil
		}

		if node.LineNumber == line {
			return node
		}
	}
}
