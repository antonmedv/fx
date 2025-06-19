package main

import (
	"strconv"

	. "github.com/antonmedv/fx/internal/jsonx"
)

type CommandType int

const (
	GotoLine CommandType = iota
	Unsupported
)

func (m *model) runCommand(s string) {
	num, err := strconv.Atoi(s)
	var commandType CommandType

	if err != nil {
		commandType = Unsupported
		m.commandInput.SetValue("")
		return
	}

	commandType = GotoLine

	switch commandType {
	case GotoLine:
		gotoLine(m, num)
		return
	case Unsupported:
		return
	}
}

func gotoLine(m *model, num int) {
	m.selectNode(findNode(m, num))

	m.commandInput.SetValue("")
	m.recordHistory()
}

func findNode(m *model, line int) *Node {
	if line >= m.totalLines {
		return m.top.Bottom()
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
