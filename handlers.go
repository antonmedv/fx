package main

import (
	"encoding/base64"
	"bytes"
	"math"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/jsonpath"
	"github.com/antonmedv/fx/internal/utils"
)

// Navigation key handlers

func (m *model) handlePageUp() (tea.Model, tea.Cmd) {
	m.cursor = m.viewHeight() - 1
	m.showCursor = true
	m.scrollBackward(max(0, m.viewHeight()-2))
	m.scrollIntoView() // As the cursor is at the bottom, and it may be empty.
	m.recordHistory()
	return m, nil
}

func (m *model) handlePageDown() (tea.Model, tea.Cmd) {
	m.cursor = 0
	m.showCursor = true
	m.scrollForward(max(0, m.viewHeight()-2))
	m.recordHistory()
	return m, nil
}

func (m *model) handleHalfPageUp() (tea.Model, tea.Cmd) {
	m.showCursor = true
	m.scrollBackward(m.viewHeight() / 2)
	m.scrollIntoView() // As the cursor stays at the same position, and it may be empty.
	m.recordHistory()
	return m, nil
}

func (m *model) handleHalfPageDown() (tea.Model, tea.Cmd) {
	m.showCursor = true
	m.scrollForward(m.viewHeight() / 2)
	m.scrollIntoView() // As the cursor stays at the same position, and it may be empty.
	m.recordHistory()
	return m, nil
}

func (m *model) handleGotoTop() (tea.Model, tea.Cmd) {
	m.head = m.top
	m.cursor = 0
	m.showCursor = true
	m.recordHistory()
	return m, nil
}

func (m *model) handleGotoBottom() (tea.Model, tea.Cmd) {
	m.scrollToBottom()
	m.recordHistory()
	return m, nil
}

// Sibling navigation handlers

func (m *model) handleNextSibling() (tea.Model, tea.Cmd) {
	pointsTo, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	var nextSibling *Node
	if pointsTo.End != nil && pointsTo.End.Next != nil {
		nextSibling = pointsTo.End.Next
	} else if pointsTo.ChunkEnd != nil && pointsTo.ChunkEnd.Next != nil {
		nextSibling = pointsTo.ChunkEnd.Next
	} else {
		nextSibling = pointsTo.Next
	}
	if nextSibling != nil {
		m.selectNode(nextSibling)
	}
	m.recordHistory()
	return m, nil
}

func (m *model) handlePrevSibling() (tea.Model, tea.Cmd) {
	pointsTo, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	var prevSibling *Node
	parent := pointsTo.Parent
	if parent != nil && parent.End == pointsTo {
		prevSibling = parent
	} else if pointsTo.Prev != nil {
		prevSibling = pointsTo.Prev
		parent := prevSibling.Parent
		if parent != nil && parent.End == prevSibling {
			prevSibling = parent
		} else if prevSibling.Chunk != "" {
			prevSibling = parent
		}
	}
	if prevSibling != nil {
		m.selectNode(prevSibling)
	}
	m.recordHistory()
	return m, nil
}

// Collapse/Expand handlers

func (m *model) handleCollapse() (tea.Model, tea.Cmd) {
	n, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	if n.HasChildren() && !n.IsCollapsed() {
		n.Collapse()
	} else {
		if n.Parent != nil {
			n = n.Parent
		}
	}
	m.selectNode(n)
	m.recordHistory()
	return m, nil
}

func (m *model) handleExpand() (tea.Model, tea.Cmd) {
	n, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	n.Expand()
	m.showCursor = true
	return m, nil
}

func (m *model) handleCollapseRecursively() (tea.Model, tea.Cmd) {
	n, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	if n.HasChildren() {
		n.CollapseRecursively()
	}
	m.showCursor = true
	return m, nil
}

func (m *model) handleExpandRecursively() (tea.Model, tea.Cmd) {
	n, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	if n.HasChildren() {
		n.ExpandRecursively(0, math.MaxInt)
	}
	m.showCursor = true
	return m, nil
}

func (m *model) handleCollapseAll() (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if ok {
		m.collapsed = true
		n := m.top
		for n != nil {
			if n.Kind != Err {
				n.CollapseRecursively()
			}
			if n.End == nil {
				n = nil
			} else {
				n = n.End.Next
			}
		}
		m.selectNode(at.Root())
		m.recordHistory()
	}
	return m, nil
}

func (m *model) handleExpandAll() (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	m.collapsed = false
	n := m.top
	for n != nil {
		n.ExpandRecursively(0, math.MaxInt)
		if n.End == nil {
			n = nil
		} else {
			n = n.End.Next
		}
	}
	m.selectNode(at)
	return m, nil
}

func (m *model) handleCollapseLevel(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if ok && at.HasChildren() {
		toLevel, _ := strconv.Atoi(msg.String())
		at.CollapseRecursively()
		at.ExpandRecursively(0, toLevel)
		m.showCursor = true
	}
	return m, nil
}

func (m *model) handleToggleWrap() (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	m.wrap = !m.wrap
	if m.wrap {
		Wrap(m.top, m.viewWidth())
	} else {
		DropWrapAll(m.top)
	}
	m.searchCache.invalidate()
	if at.Chunk != "" && at.Value == "" {
		at = at.Parent
	}
	m.redoSearch()
	m.selectNode(at)
	return m, nil
}

// Preview and action handlers

func (m *model) handlePreviewKey() (tea.Model, tea.Cmd) {
	m.showPreview = true
	content := ""
	value := m.cursorValue()
	if decodedValue, err := base64.StdEncoding.DecodeString(value); err == nil {
		img, err := utils.DrawImage(bytes.NewReader(decodedValue), m.termWidth, m.termHeight)
		if err == nil {
			content = strings.TrimRight(img, "\n")
		}
	}
	if content == "" {
		content = lipgloss.NewStyle().Width(m.termWidth).Render(value)
	}
	m.preview.SetContent(content)
	m.preview.GotoTop()
	return m, nil
}

// Input mode handlers

func (m *model) handleDigMode() (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	if at.Kind == Err {
		nextJson := at.FindNextNonErr()
		if nextJson != nil {
			m.selectNode(nextJson)
		}
	}
	m.digInput.SetValue(m.cursorPath() + ".")
	m.digInput.CursorEnd()
	m.digInput.Width = m.termWidth - 1
	m.digInput.Focus()
	return m, nil
}

func (m *model) handleGotoSymbolMode() (tea.Model, tea.Cmd) {
	m.gotoSymbolInput.CursorEnd()
	m.gotoSymbolInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
	m.gotoSymbolInput.Focus()
	m.createKeysIndex()
	return m, nil
}

func (m *model) handleGotoRefMode() (tea.Model, tea.Cmd) {
	at, ok := m.cursorPointsTo()
	if !ok {
		return m, nil
	}
	value, isRef := isRefNode(at)
	if isRef {
		refPath, ok := jsonpath.ParseSchemaRef(value)
		if ok {
			m.selectNode(m.findByPath(refPath))
			m.recordHistory()
		}
	}
	return m, nil
}

func (m *model) handleCommandLineMode() (tea.Model, tea.Cmd) {
	m.commandInput.CursorEnd()
	m.commandInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
	m.commandInput.Focus()
	return m, nil
}

func (m *model) handleSearchMode() (tea.Model, tea.Cmd) {
	m.searchInput.CursorEnd()
	m.searchInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
	m.searchInput.Focus()
	return m, nil
}

// Search navigation handlers

func (m *model) handleSearchNext() (tea.Model, tea.Cmd) {
	m.selectSearchResult(m.search.cursor + 1)
	m.recordHistory()
	return m, nil
}

func (m *model) handleSearchPrev() (tea.Model, tea.Cmd) {
	m.selectSearchResult(m.search.cursor - 1)
	m.recordHistory()
	return m, nil
}

// History navigation handlers

func (m *model) handleGoBack() (tea.Model, tea.Cmd) {
	if m.locationIndex > 0 {
		at, ok := m.cursorPointsTo()
		if !ok {
			return m, nil
		}
		m.locationIndex--

		loc := m.locationHistory[m.locationIndex]
		for loc.node == at && m.locationIndex > 0 {
			m.locationIndex--
			loc = m.locationHistory[m.locationIndex]
		}
		m.selectNode(loc.head)
		m.selectNode(loc.node)
	}
	return m, nil
}

func (m *model) handleGoForward() (tea.Model, tea.Cmd) {
	if m.locationIndex < len(m.locationHistory)-1 {
		m.locationIndex++
		loc := m.locationHistory[m.locationIndex]
		m.selectNode(loc.head)
		m.selectNode(loc.node)
	}
	return m, nil
}
