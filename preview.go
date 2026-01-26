package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/antonmedv/fx/internal/theme"
)

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
		m.preview.SetContent(m.previewContent)
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.previewSearchInput.Blur()
		m.doPreviewSearch(m.previewSearchInput.Value())
		return m, nil

	default:
		m.previewSearchInput, cmd = m.previewSearchInput.Update(msg)
	}
	return m, cmd
}

func (m *model) doPreviewSearch(pattern string) {
	if pattern == "" {
		m.previewSearchResults = nil
		m.previewSearchCursor = -1
		m.preview.SetContent(m.previewContent)
		return
	}

	code, ci := regexCase(pattern)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		return
	}

	lines := strings.Split(m.previewContent, "\n")
	m.previewSearchResults = nil
	var highlightedLines []string

	for i, line := range lines {
		matches := re.FindAllStringIndex(line, -1)
		if len(matches) > 0 {
			m.previewSearchResults = append(m.previewSearchResults, i)
		}
		highlightedLines = append(highlightedLines, m.highlightLine(line, matches, -1))
	}

	m.preview.SetContent(strings.Join(highlightedLines, "\n"))

	if len(m.previewSearchResults) > 0 {
		m.selectPreviewSearchResult(0)
	} else {
		m.previewSearchCursor = -1
	}
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
	lineNum := m.previewSearchResults[i]

	// Re-highlight with current match emphasized
	m.rehighlightPreview()

	// Scroll to the line
	m.preview.SetYOffset(lineNum)
}

func (m *model) rehighlightPreview() {
	pattern := m.previewSearchInput.Value()
	if pattern == "" {
		return
	}

	code, ci := regexCase(pattern)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		return
	}

	lines := strings.Split(m.previewContent, "\n")
	var highlightedLines []string
	resultIndex := 0

	for i, line := range lines {
		matches := re.FindAllStringIndex(line, -1)
		currentMatchLine := -1
		if resultIndex < len(m.previewSearchResults) && m.previewSearchResults[resultIndex] == i {
			if resultIndex == m.previewSearchCursor {
				currentMatchLine = i
			}
			resultIndex++
		}
		highlightedLines = append(highlightedLines, m.highlightLine(line, matches, currentMatchLine))
	}

	m.preview.SetContent(strings.Join(highlightedLines, "\n"))
}

func (m *model) highlightLine(line string, matches [][]int, currentMatchLine int) string {
	if len(matches) == 0 {
		return line
	}

	var result strings.Builder
	lastEnd := 0

	for matchIdx, match := range matches {
		start, end := match[0], match[1]

		// Add text before the match
		if start > lastEnd {
			result.WriteString(line[lastEnd:start])
		}

		// Highlight the match
		matchText := line[start:end]
		if currentMatchLine >= 0 && matchIdx == 0 {
			// Current search result - use cursor style
			result.WriteString(theme.CurrentTheme.Cursor(matchText))
		} else {
			// Other matches - use search style
			result.WriteString(theme.CurrentTheme.Search(matchText))
		}

		lastEnd = end
	}

	// Add remaining text after last match
	if lastEnd < len(line) {
		result.WriteString(line[lastEnd:])
	}

	return result.String()
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
