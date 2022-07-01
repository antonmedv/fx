package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"runtime/pprof"
	"strings"

	. "github.com/antonmedv/fx/pkg/dict"
	. "github.com/antonmedv/fx/pkg/json"
	. "github.com/antonmedv/fx/pkg/reducer"
	. "github.com/antonmedv/fx/pkg/theme"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/muesli/termenv"
)

var (
	flagVersion   bool
	flagPrintCode bool
)

func main() {
	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-v", "-V", "--version":
			flagVersion = true
		case "--print-code":
			flagPrintCode = true
		default:
			args = append(args, arg)
		}

	}
	if flagVersion {
		fmt.Println(version)
		return
	}
	cpuProfile := os.Getenv("CPU_PROFILE")
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
	}
	themeId, ok := os.LookupEnv("FX_THEME")
	if !ok {
		themeId = "1"
	}
	theme, ok := Themes[themeId]
	if !ok {
		theme = Themes["1"]
	}
	if termenv.ColorProfile() == termenv.Ascii {
		theme = Themes["0"]
	}
	var showSize bool
	if s, ok := os.LookupEnv("FX_SHOW_SIZE"); ok {
		if s == "true" {
			showSize = true
		}
	}

	stdinIsTty := isatty.IsTerminal(os.Stdin.Fd())
	stdoutIsTty := isatty.IsTerminal(os.Stdout.Fd())
	filePath := ""
	fileName := ""
	var dec *json.Decoder
	if stdinIsTty {
		// Nothing was piped, maybe file argument?
		if len(args) >= 1 {
			filePath = args[0]
			f, err := os.Open(filePath)
			if err != nil {
				switch err.(type) {
				case *fs.PathError:
					fmt.Println(err)
					os.Exit(1)
				default:
					panic(err)
				}
			}
			fileName = path.Base(filePath)
			dec = json.NewDecoder(f)
			args = args[1:]
		}
	} else {
		dec = json.NewDecoder(os.Stdin)
	}
	if dec == nil {
		fmt.Println(usage(DefaultKeyMap()))
		os.Exit(1)
	}
	dec.UseNumber()
	object, err := Parse(dec)
	if err != nil {
		fmt.Println("JSON Parse Error:", err.Error())
		os.Exit(1)
	}
	lang, ok := os.LookupEnv("FX_LANG")
	if !ok {
		lang = "js"
	}
	var fxrc string
	if lang == "js" || lang == "node" {
		home, err := os.UserHomeDir()
		if err == nil {
			b, err := os.ReadFile(path.Join(home, ".fxrc.js"))
			if err == nil {
				fxrc = "\n" + string(b)
				if lang == "js" {
					parts := strings.SplitN(fxrc, "// nodejs:", 2)
					fxrc = parts[0]
				}
			}
		}
	}
	if dec.More() {
		os.Exit(stream(dec, object, lang, args, theme, fxrc))
	}
	if len(args) > 0 || !stdoutIsTty {
		if len(args) > 0 && flagPrintCode {
			fmt.Print(GenerateCode(lang, args, fxrc))
			return
		}
		if lang == "js" {
			simplePath, ok := SplitSimplePath(args)
			if ok {
				output := GetBySimplePath(object, simplePath)
				Echo(output, theme)
				os.Exit(0)
			}
			vm, fn, err := CreateJS(args, fxrc)
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			os.Exit(ReduceJS(vm, fn, object, theme))
		} else {
			os.Exit(Reduce(object, lang, args, theme, fxrc))
		}
	}

	// Start interactive mode.
	expand := map[string]bool{"": true}
	if array, ok := object.(Array); ok {
		for i := range array {
			expand[accessor("", i)] = true
		}
	}
	parents := map[string]string{}
	children := map[string][]string{}
	canBeExpanded := map[string]bool{}
	Dfs(object, func(it Iterator) {
		parents[it.Path] = it.Parent
		children[it.Parent] = append(children[it.Parent], it.Path)
		switch it.Object.(type) {
		case *Dict:
			canBeExpanded[it.Path] = len(it.Object.(*Dict).Keys) > 0
		case Array:
			canBeExpanded[it.Path] = len(it.Object.(Array)) > 0
		}
	})
	input := textinput.New()
	input.Prompt = ""
	m := &model{
		fileName:        fileName,
		theme:           theme,
		json:            object,
		showSize:        showSize,
		width:           80,
		height:          60,
		mouseWheelDelta: 3,
		keyMap:          DefaultKeyMap(),
		expandedPaths:   expand,
		canBeExpanded:   canBeExpanded,
		parents:         parents,
		children:        children,
		nextSiblings:    map[string]string{},
		prevSiblings:    map[string]string{},
		wrap:            true,
		searchInput:     input,
	}
	m.collectSiblings(m.json, "")
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if err := p.Start(); err != nil {
		panic(err)
	}
	if cpuProfile != "" {
		pprof.StopCPUProfile()
	}
	os.Exit(m.exitCode)
}

type model struct {
	exitCode      int
	width, height int
	windowHeight  int
	footerHeight  int
	wrap          bool
	theme         Theme
	showSize      bool // Show number of elements in preview

	fileName string
	json     interface{}
	lines    []string

	mouseWheelDelta int // Number of lines the mouse wheel will scroll
	offset          int // offset is the vertical scroll position

	keyMap   KeyMap
	showHelp bool

	expandedPaths              map[string]bool     // set of expanded paths
	canBeExpanded              map[string]bool     // set of path => can be expanded (i.e. dict or array)
	paths                      []string            // array of paths on screen
	pathToLineNumber           map[string]int      // map of path => line Number
	pathToIndex                map[string]int      // map of path => index in m.paths
	lineNumberToPath           map[int]string      // map of line Number => path
	parents                    map[string]string   // map of subpath => parent path
	children                   map[string][]string // map of path => child paths
	nextSiblings, prevSiblings map[string]string   // map of path => sibling path
	cursor                     int                 // cursor in [0, len(m.paths)]
	showCursor                 bool

	searchInput             textinput.Model
	searchRegexCompileError string
	showSearchResults       bool
	searchResults           []*searchResult
	searchResultsCursor     int
	highlightIndex          map[string]*rangeGroup
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
				m.clearSearchResults()
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
			m.moveCursorToView()
			m.render()
		case key.Matches(msg, m.keyMap.PageUp):
			m.ViewUp()
			m.moveCursorToView()
			m.render()
		case key.Matches(msg, m.keyMap.HalfPageDown):
			m.HalfViewDown()
			m.moveCursorToView()
			m.render()
		case key.Matches(msg, m.keyMap.HalfPageUp):
			m.HalfViewUp()
			m.moveCursorToView()
			m.render()
		case key.Matches(msg, m.keyMap.GotoTop):
			m.GotoTop()
			m.moveCursorToTop()
			m.render()
		case key.Matches(msg, m.keyMap.GotoBottom):
			m.GotoBottom()
			m.moveCursorToBottom()
			m.render()
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
			m.down()
			m.render()
			m.scrollDownToCursor()

		case key.Matches(msg, m.keyMap.Up):
			m.up()
			m.render()
			m.scrollUpToCursor()

		case key.Matches(msg, m.keyMap.NextSibling):
			nextSiblingPath, ok := m.nextSiblings[m.cursorPath()]
			if ok {
				m.showCursor = true
				m.cursor = m.pathToIndex[nextSiblingPath]
			} else {
				m.down()
			}
			m.render()
			m.scrollDownToCursor()

		case key.Matches(msg, m.keyMap.PrevSibling):
			prevSiblingPath, ok := m.prevSiblings[m.cursorPath()]
			if ok {
				m.showCursor = true
				m.cursor = m.pathToIndex[prevSiblingPath]
			} else {
				m.up()
			}
			m.render()
			m.scrollUpToCursor()

		case key.Matches(msg, m.keyMap.Expand):
			m.showCursor = true
			if m.canBeExpanded[m.cursorPath()] {
				m.expandedPaths[m.cursorPath()] = true
			}
			m.render()

		case key.Matches(msg, m.keyMap.ExpandRecursively):
			m.showCursor = true
			if m.canBeExpanded[m.cursorPath()] {
				m.expandRecursively(m.cursorPath())
			}
			m.render()

		case key.Matches(msg, m.keyMap.Collapse):
			m.showCursor = true
			if m.canBeExpanded[m.cursorPath()] && m.expandedPaths[m.cursorPath()] {
				m.expandedPaths[m.cursorPath()] = false
			} else {
				parentPath, ok := m.parents[m.cursorPath()]
				if ok {
					m.expandedPaths[parentPath] = false
					m.cursor = m.pathToIndex[parentPath]
				}
			}
			m.render()
			m.scrollUpToCursor()

		case key.Matches(msg, m.keyMap.CollapseRecursively):
			m.showCursor = true
			if m.canBeExpanded[m.cursorPath()] && m.expandedPaths[m.cursorPath()] {
				m.collapseRecursively(m.cursorPath())
			} else {
				parentPath, ok := m.parents[m.cursorPath()]
				if ok {
					m.collapseRecursively(parentPath)
					m.cursor = m.pathToIndex[parentPath]
				}
			}
			m.render()
			m.scrollUpToCursor()

		case key.Matches(msg, m.keyMap.ToggleWrap):
			m.wrap = !m.wrap
			m.render()

		case key.Matches(msg, m.keyMap.ExpandAll):
			Dfs(m.json, func(it Iterator) {
				switch it.Object.(type) {
				case *Dict, Array:
					m.expandedPaths[it.Path] = true
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
			m.searchRegexCompileError = ""
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
			if msg.Y >= m.height {
				// Clicked on status bar or search input.
				break
			}
			clickedPath, ok := m.lineNumberToPath[m.offset+msg.Y]
			if ok {
				if m.canBeExpanded[clickedPath] {
					m.expandedPaths[clickedPath] = !m.expandedPaths[clickedPath]
				}
				m.cursor = m.pathToIndex[clickedPath]
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
	if m.showHelp {
		statusBar := "Press Esc or q to close help."
		statusBar += strings.Repeat(" ", max(0, m.width-width(statusBar)))
		statusBar = m.theme.StatusBar(statusBar)
		return strings.Join(lines, "\n") + extraLines + "\n" + statusBar
	}
	statusBar := m.cursorPath() + " "
	statusBar += strings.Repeat(" ", max(0, m.width-width(statusBar)-width(m.fileName)))
	statusBar += m.fileName
	statusBar = m.theme.StatusBar(statusBar)
	output := strings.Join(lines, "\n") + extraLines + "\n" + statusBar
	if m.searchInput.Focused() {
		output += "\n/" + m.searchInput.View()
	}
	if len(m.searchRegexCompileError) > 0 {
		output += fmt.Sprintf("\n/%v/i  %v", m.searchInput.Value(), m.searchRegexCompileError)
	}
	if m.showSearchResults {
		if len(m.searchResults) == 0 {
			output += fmt.Sprintf("\n/%v/i  not found", m.searchInput.Value())
		} else {
			output += fmt.Sprintf("\n/%v/i  found: [%v/%v]", m.searchInput.Value(), m.searchResultsCursor+1, len(m.searchResults))
		}
	}
	return output
}

func (m *model) recalculateViewportHeight() {
	m.height = m.windowHeight
	m.height-- // status bar
	if !m.showHelp {
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
}

func (m *model) render() {
	m.recalculateViewportHeight()

	if m.showHelp {
		m.lines = keyMapInfo(m.keyMap, lipgloss.NewStyle().PaddingLeft(4).PaddingTop(2).PaddingBottom(2))
		return
	}

	m.paths = make([]string, 0)
	m.pathToIndex = make(map[string]int, 0)
	if m.pathToLineNumber == nil {
		m.pathToLineNumber = make(map[string]int, 0)
	} else {
		m.pathToLineNumber = make(map[string]int, len(m.pathToLineNumber))
	}
	if m.lineNumberToPath == nil {
		m.lineNumberToPath = make(map[int]string, 0)
	} else {
		m.lineNumberToPath = make(map[int]string, len(m.lineNumberToPath))
	}
	m.lines = m.print(m.json, 1, 0, 0, "", true)

	if m.offset > len(m.lines)-1 {
		m.GotoBottom()
	}
}

func (m *model) cursorPath() string {
	if m.cursor == 0 {
		return ""
	}
	if 0 <= m.cursor && m.cursor < len(m.paths) {
		return m.paths[m.cursor]
	}
	return "?"
}

func (m *model) cursorLineNumber() int {
	if 0 <= m.cursor && m.cursor < len(m.paths) {
		return m.pathToLineNumber[m.paths[m.cursor]]
	}
	return -1
}

func (m *model) expandRecursively(path string) {
	if m.canBeExpanded[path] {
		m.expandedPaths[path] = true
		for _, childPath := range m.children[path] {
			if childPath != "" {
				m.expandRecursively(childPath)
			}
		}
	}
}

func (m *model) collapseRecursively(path string) {
	m.expandedPaths[path] = false
	for _, childPath := range m.children[path] {
		if childPath != "" {
			m.collapseRecursively(childPath)
		}
	}
}

func (m *model) collectSiblings(v interface{}, path string) {
	switch v.(type) {
	case *Dict:
		prev := ""
		for _, k := range v.(*Dict).Keys {
			subpath := path + "." + k
			if prev != "" {
				m.nextSiblings[prev] = subpath
				m.prevSiblings[subpath] = prev
			}
			prev = subpath
			value, _ := v.(*Dict).Get(k)
			m.collectSiblings(value, subpath)
		}

	case Array:
		prev := ""
		for i, value := range v.(Array) {
			subpath := fmt.Sprintf("%v[%v]", path, i)
			if prev != "" {
				m.nextSiblings[prev] = subpath
				m.prevSiblings[subpath] = prev
			}
			prev = subpath
			m.collectSiblings(value, subpath)
		}
	}
}

func (m *model) down() {
	m.showCursor = true
	if m.cursor < len(m.paths)-1 { // scroll till last element in m.paths
		m.cursor++
	} else {
		// at the bottom of viewport maybe some hidden brackets, lets scroll to see them
		if !m.AtBottom() {
			m.LineDown(1)
		}
	}
	if m.cursor >= len(m.paths) {
		m.cursor = len(m.paths) - 1
	}
}

func (m *model) up() {
	m.showCursor = true
	if m.cursor > 0 {
		m.cursor--
	}
	if m.cursor >= len(m.paths) {
		m.cursor = len(m.paths) - 1
	}
}

func (m *model) moveCursorToView() {
	top := 0
	bottom := 0

	// find the first path in the viewport
	for i := 0; i < m.height && top == 0; i++ {
		top = m.pathToIndex[m.lineNumberToPath[m.offset+i]]
	}

	// find the last path in the viewport
	for i := 1; i < m.height && bottom == 0; i++ {
		bottom = m.pathToIndex[m.lineNumberToPath[m.offset+m.height-i]]
	}

	m.showCursor = true
	m.cursor = clamp(m.cursor, bottom, top)
}

func (m *model) moveCursorToTop() {
	m.showCursor = true
	m.cursor = 0
}

func (m *model) moveCursorToBottom() {
	m.showCursor = true
	m.cursor = len(m.paths) - 1
}
