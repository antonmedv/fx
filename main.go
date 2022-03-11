package main

import (
	"encoding/json"
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
	"os"
	"path"
	"strings"
)

var colors = struct {
	cursor    lipgloss.Style
	bracket   lipgloss.Style
	key       lipgloss.Style
	null      lipgloss.Style
	boolean   lipgloss.Style
	number    lipgloss.Style
	string    lipgloss.Style
	preview   lipgloss.Style
	statusBar lipgloss.Style
	search    lipgloss.Style
}{
	cursor:    lipgloss.NewStyle().Reverse(true),
	bracket:   lipgloss.NewStyle(),
	key:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")),
	null:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")),
	boolean:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("3")),
	number:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")),
	string:    lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2")),
	preview:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")),
	statusBar: lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0")),
	search:    lipgloss.NewStyle().Background(lipgloss.Color("11")).Foreground(lipgloss.Color("16")),
}

func main() {
	filePath := ""
	var dec *json.Decoder
	if term.IsTerminal(int(os.Stdin.Fd())) {
		if len(os.Args) >= 2 {
			filePath = os.Args[1]
			f, err := os.Open(os.Args[1])
			if err != nil {
				panic(err)
			}
			dec = json.NewDecoder(f)
		}
	} else {
		dec = json.NewDecoder(os.Stdin)
	}
	dec.UseNumber()
	jsonObject, err := parse(dec)
	if err != nil {
		panic(err)
	}

	expand := map[string]bool{
		"": true,
	}

	if array, ok := jsonObject.(array); ok {
		for i := range array {
			expand[accessor("", i)] = true
		}
	}

	parents := map[string]string{}
	dfs(jsonObject, func(it iterator) {
		parents[it.path] = it.parent
	})

	input := textinput.New()
	input.Prompt = ""

	m := &model{
		fileName:        path.Base(filePath),
		json:            jsonObject,
		width:           80,
		height:          60,
		mouseWheelDelta: 3,
		keyMap:          DefaultKeyMap(),
		expandedPaths:   expand,
		parents:         parents,
		wrap:            true,
		searchInput:     input,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if err := p.Start(); err != nil {
		panic(err)
	}
	os.Exit(m.exitCode)
}

type model struct {
	exitCode      int
	width, height int
	windowHeight  int
	footerHeight  int

	fileName string
	json     interface{}
	lines    []string

	mouseWheelDelta int // number of lines the mouse wheel will scroll
	offset          int // offset is the vertical scroll position

	keyMap   KeyMap
	showHelp bool

	expandedPaths    map[string]bool   // a set with expanded paths
	canBeExpanded    map[string]bool   // a set for path => can be expanded (i.e. dict or array)
	pathToLineNumber *dict             // dict with path => line number
	lineNumberToPath map[int]string    // map of line number => path
	parents          map[string]string // map of subpath => parent path
	cursor           int               // cursor in range of m.pathToLineNumber.keys slice
	showCursor       bool

	wrap                    bool
	searchInput             textinput.Model
	searchRegexCompileError string
	searchResults           *dict // path => searchResult
	showSearchResults       bool
	resultsCursor           int // [0, searchResults length)
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.windowHeight = msg.Height
		m.searchInput.Width = msg.Width - 2 // minus prompt
		m.render()

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			m.LineUp(m.mouseWheelDelta)
		case tea.MouseWheelDown:
			m.LineDown(m.mouseWheelDelta)
		}
	}

	if m.searchInput.Focused() {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.Type {
			case tea.KeyEsc:
				m.searchInput.Blur()
				m.searchResults = newDict()
				m.render()

			case tea.KeyEnter:
				m.doSearch(m.searchInput.Value())
			}
		}
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.PageDown):
			m.ViewDown()
		case key.Matches(msg, m.keyMap.PageUp):
			m.ViewUp()
		case key.Matches(msg, m.keyMap.HalfPageDown):
			m.HalfViewDown()
		case key.Matches(msg, m.keyMap.HalfPageUp):
			m.HalfViewUp()
		case key.Matches(msg, m.keyMap.GotoTop):
			m.GotoTop()
		case key.Matches(msg, m.keyMap.GotoBottom):
			m.GotoBottom()
		}
	}

	if m.showHelp {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, m.keyMap.Quit):
				m.showHelp = false
				m.render()
			case key.Matches(msg, m.keyMap.Down):
				m.LineDown(1)
			case key.Matches(msg, m.keyMap.Up):
				m.LineUp(1)
			}
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			m.exitCode = 0
			return m, tea.Quit

		case key.Matches(msg, m.keyMap.Help):
			m.GotoTop()
			m.showHelp = !m.showHelp
			m.render()

		case key.Matches(msg, m.keyMap.Down):
			m.showCursor = true
			if m.cursor < len(m.pathToLineNumber.keys)-1 { // scroll till last element in m.pathToLineNumber
				m.cursor++
			} else {
				// at the bottom of viewport maybe some hidden brackets, lets scroll to see them
				if !m.AtBottom() {
					m.LineDown(1)
				}
			}
			if m.cursor >= len(m.pathToLineNumber.keys) {
				m.cursor = len(m.pathToLineNumber.keys) - 1
			}
			m.render()
			at := m.cursorLineNumber()
			if m.offset <= at { // cursor is lower
				m.LineDown(max(0, at-(m.offset+m.height-1))) // minus one is due to cursorLineNumber() starts from 0
			} else {
				m.SetOffset(at)
			}

		case key.Matches(msg, m.keyMap.Up):
			m.showCursor = true
			if m.cursor > 0 {
				m.cursor--
			}
			if m.cursor >= len(m.pathToLineNumber.keys) {
				m.cursor = len(m.pathToLineNumber.keys) - 1
			}
			m.render()
			at := m.cursorLineNumber()
			if at < m.offset+m.height { // cursor is above
				m.LineUp(max(0, m.offset-at))
			} else {
				m.SetOffset(at)
			}

		case key.Matches(msg, m.keyMap.Expand):
			m.showCursor = true
			m.expandedPaths[m.cursorPath()] = true
			m.render()

		case key.Matches(msg, m.keyMap.Collapse):
			m.showCursor = true
			if m.expandedPaths[m.cursorPath()] {
				m.expandedPaths[m.cursorPath()] = false
			} else {
				parentPath, ok := m.parents[m.cursorPath()]
				if ok {
					m.expandedPaths[parentPath] = false
					index, _ := m.pathToLineNumber.index(parentPath)
					m.cursor = index
				}
			}
			m.render()
			at := m.cursorLineNumber()
			if at < m.offset+m.height { // cursor is above
				m.LineUp(max(0, m.offset-at))
			} else {
				m.SetOffset(at)
			}

		case key.Matches(msg, m.keyMap.ToggleWrap):
			m.wrap = !m.wrap
			m.render()

		case key.Matches(msg, m.keyMap.ExpandAll):
			dfs(m.json, func(it iterator) {
				switch it.object.(type) {
				case *dict, array:
					m.expandedPaths[it.path] = true
				}
			})
			m.render()

		case key.Matches(msg, m.keyMap.CollapseAll):
			m.expandedPaths = map[string]bool{
				"": true,
			}
			m.render()

		case key.Matches(msg, m.keyMap.Search):
			m.showSearchResults = false
			m.searchInput.Focus()
			m.render()
			return m, textinput.Blink

		case key.Matches(msg, m.keyMap.Next):
			if m.showSearchResults {
				m.nextSearchResult()
			}

		case key.Matches(msg, m.keyMap.Prev):
			if m.showSearchResults {
				m.prevSearchResult()
			}
		}

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseLeft:
			m.showCursor = true
			clickedPath, ok := m.lineNumberToPath[m.offset+msg.Y]
			if ok {
				if m.canBeExpanded[clickedPath] {
					m.expandedPaths[clickedPath] = !m.expandedPaths[clickedPath]
				}
				index, _ := m.pathToLineNumber.index(clickedPath)
				m.cursor = index
				m.render()
			}
		}
	}
	return m, nil
}

func (m *model) View() string {
	lines := m.visibleLines()
	extraLines := ""
	if len(lines) < m.height {
		extraLines = strings.Repeat("\n", max(0, m.height-len(lines)))
	}
	statusBar := m.cursorPath() + " "
	if m.showHelp {
		statusBar = "Press Esc or q to close help."
	}
	statusBar += strings.Repeat(" ", max(0, m.width-width(statusBar)-width(m.fileName)))
	statusBar += m.fileName
	statusBar = colors.statusBar.Render(statusBar)
	output := strings.Join(lines, "\n") + extraLines + "\n" + statusBar
	if m.searchInput.Focused() {
		output += "\n/" + m.searchInput.View()
	}
	if len(m.searchRegexCompileError) > 0 {
		output += fmt.Sprintf("\n/%v/i  %v", m.searchInput.Value(), m.searchRegexCompileError)
	}
	if m.showSearchResults {
		if len(m.searchResults.keys) == 0 {
			output += fmt.Sprintf("\n/%v/i  not found", m.searchInput.Value())
		} else {
			output += fmt.Sprintf("\n/%v/i  found: [%v/%v]", m.searchInput.Value(), m.resultsCursor+1, len(m.searchResults.keys))
		}
	}
	return output
}

func (m *model) recalculateViewportHeight() {
	m.height = m.windowHeight
	m.height-- // status bar
	if m.searchInput.Focused() {
		m.height--
	}
	if m.showSearchResults {
		m.height--
	}
	if len(m.searchRegexCompileError) > 0 {
		m.height--
	}
}

func (m *model) render() {
	m.recalculateViewportHeight()

	if m.showHelp {
		m.lines = m.helpView()
		return
	}

	m.pathToLineNumber = newDict()
	m.canBeExpanded = map[string]bool{}
	m.lineNumberToPath = map[int]string{}
	m.lines = m.print(m.json, 1, 0, 0, "", false)

	if m.offset > len(m.lines)-1 {
		m.GotoBottom()
	}
}

func (m *model) cursorPath() string {
	if m.cursor == 0 {
		return ""
	}
	if 0 <= m.cursor && m.cursor < len(m.pathToLineNumber.keys) {
		return m.pathToLineNumber.keys[m.cursor]
	}
	return "?"
}

func (m *model) cursorLineNumber() int {
	if 0 <= m.cursor && m.cursor < len(m.pathToLineNumber.keys) {
		return m.pathToLineNumber.values[m.pathToLineNumber.keys[m.cursor]].(int)
	}
	return -1
}
