package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var reverseStyle = lipgloss.NewStyle().Reverse(true).Render

func (m *model) handlePreviewKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		if m.previewSearchInput.Focused() {
			return m.handlePreviewSearchInput(msg)
		}
		switch {
		case key.Matches(msg, keyMap.Quit),
			key.Matches(msg, keyMap.Preview):
			m.showPreview = false
			return m, nil

		case key.Matches(msg, keyMap.Print):
			return m, m.print()

		case key.Matches(msg, keyMap.GotoTop):
			m.preview.GotoTop()
			return m, nil

		case key.Matches(msg, keyMap.GotoBottom):
			m.preview.GotoBottom()
			return m, nil

		case key.Matches(msg, keyMap.HalfPageUp):
			m.preview.HalfPageUp()
			return m, nil

		case key.Matches(msg, keyMap.HalfPageDown):
			m.preview.HalfPageDown()
			return m, nil

		case key.Matches(msg, keyMap.PageUp):
			m.preview.PageUp()
			return m, nil

		case key.Matches(msg, keyMap.PageDown):
			m.preview.PageDown()
			return m, nil

		case key.Matches(msg, keyMap.Search):
			m.previewSearchInput.CursorEnd()
			m.previewSearchInput.Width = m.termWidth - 2
			m.previewSearchInput.Focus()
			return m, nil

		case key.Matches(msg, keyMap.SearchNext):
			m.selectPreviewSearchResult(m.previewSearchCursor + 1)
			return m, nil

		case key.Matches(msg, keyMap.SearchPrev):
			m.selectPreviewSearchResult(m.previewSearchCursor - 1)
			return m, nil
		}
	}
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m *model) handlePreviewSearchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case msg.Type == tea.KeyEscape:
		m.previewSearchInput.Blur()
		m.previewSearchInput.SetValue("")
		m.previewSearchResults = nil
		m.previewSearchCursor = -1
		m.preview.SetContent(m.wrapString(m.previewValue))
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.previewSearchInput.Blur()
		found := m.doPreviewSearch(m.previewSearchInput.Value())
		if !found {
			m.previewSearchResults = nil
			m.previewSearchCursor = -1
			m.preview.SetContent(m.wrapString(m.previewValue))
		}
		return m, nil

	default:
		m.previewSearchInput, cmd = m.previewSearchInput.Update(msg)
	}
	return m, cmd
}

func (m *model) doPreviewSearch(pattern string) bool {
	if pattern == "" {
		return false
	}

	code, ci := regexCase(pattern)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		return false
	}

	content := m.previewValue
	matches := re.FindAllStringIndex(content, -1)

	m.previewSearchResults = nil

	if len(matches) == 0 {
		return false
	}

	// Precalculate visual line numbers for each match, accounting for line wrapping
	lines := strings.Split(content, "\n")
	visualLineStarts := make([]int, len(lines))
	cumulative := 0
	for i, line := range lines {
		visualLineStarts[i] = cumulative
		wrapped := lipgloss.NewStyle().Width(m.termWidth).Render(line)
		cumulative += strings.Count(wrapped, "\n") + 1
	}

	for _, match := range matches {
		// Find original line number
		origLineNum := strings.Count(content[:match[0]], "\n")

		// Find position within the original line
		lastNewline := strings.LastIndex(content[:match[0]], "\n")
		var posInLine int
		if lastNewline == -1 {
			posInLine = match[0]
		} else {
			posInLine = match[0] - lastNewline - 1
		}

		// Calculate visual line: start of this original line + offset within wrapped line
		visualLineNum := visualLineStarts[origLineNum]

		// Add offset for wrapping within the line
		if posInLine > 0 && m.termWidth > 0 && posInLine <= len(lines[origLineNum]) {
			linePrefix := lines[origLineNum][:posInLine]
			wrappedPrefix := m.wrapString(linePrefix)
			visualLineNum += strings.Count(wrappedPrefix, "\n")
		}

		m.previewSearchResults = append(m.previewSearchResults, visualLineNum)
	}

	// Highlight all matches with Reverse style (once)
	var result strings.Builder
	lastEnd := 0
	for _, match := range matches {
		start, end := match[0], match[1]
		if start > lastEnd {
			result.WriteString(content[lastEnd:start])
		}
		result.WriteString(reverseStyle(content[start:end]))
		lastEnd = end
	}
	if lastEnd < len(content) {
		result.WriteString(content[lastEnd:])
	}

	m.preview.SetContent(m.wrapString(result.String()))

	// Jump to first match
	m.previewSearchCursor = 0
	m.preview.SetYOffset(m.previewSearchResults[0])
	return true
}

func (m *model) selectPreviewSearchResult(i int) {
	if len(m.previewSearchResults) == 0 {
		return
	}
	if i < 0 {
		i = len(m.previewSearchResults) - 1
	}
	if i >= len(m.previewSearchResults) {
		i = 0
	}
	m.previewSearchCursor = i

	// Scroll to the cached line number
	m.preview.SetYOffset(m.previewSearchResults[i])
}

func (m *model) previewSearchStatusBar() string {
	if m.previewSearchInput.Focused() {
		return m.previewSearchInput.View()
	}

	pattern := m.previewSearchInput.Value()
	if pattern == "" {
		return ""
	}

	re, ci := regexCase(pattern)
	re = "/" + re + "/"
	if ci {
		re += "i"
	}

	if len(m.previewSearchResults) == 0 {
		return flex(m.termWidth, re, "not found")
	}

	cursor := fmt.Sprintf("found: [%v/%v]", m.previewSearchCursor+1, len(m.previewSearchResults))
	return flex(m.termWidth, re, cursor)
}

func (m *model) wrapString(value string) string {
	return lipgloss.NewStyle().Width(m.termWidth).Render(value)
}
