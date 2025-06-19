package main

import (
	"strconv"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func (m *model) doGotoLine(s string) {
	num, err := strconv.Atoi(s)
	if err != nil {
		m.commandInput.SetValue("")
		return
	}

	m.selectNode(findNode(m, num))

	m.commandInput.SetValue("")
	m.recordHistory()
}

func findNode(m *model, line int) *Node {
	if line >= m.totalLines {
		return m.top.Bottom()
	}

	node := m.top

	for range line - 1 {
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
	}

	return node
}
