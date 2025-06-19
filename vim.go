package main

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	. "github.com/antonmedv/fx/internal/jsonx"
)

func (m *model) runCommand(s string) (tea.Model, tea.Cmd) {
	num, err := strconv.Atoi(s)
	if err == nil {
		gotoLine(m, num)
		return m, nil
	} else if s == "q" {
		return m, tea.Quit
	}
	return m, nil
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
