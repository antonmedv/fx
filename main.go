package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"math"
	"os"
	"path"
	"regexp"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/antonmedv/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/goccy/go-yaml"
	"github.com/mattn/go-isatty"
	"github.com/sahilm/fuzzy"

	jsonpath "github.com/antonmedv/fx/path"
)

var (
	flagHelp    bool
	flagVersion bool
	flagYaml    bool
)

func main() {
	if _, ok := os.LookupEnv("FX_PPROF"); ok {
		f, err := os.Create("cpu.prof")
		if err != nil {
			panic(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
		memProf, err := os.Create("mem.prof")
		if err != nil {
			panic(err)
		}
		defer pprof.WriteHeapProfile(memProf)
	}

	complete()

	var args []string
	for _, arg := range os.Args[1:] {
		switch arg {
		case "-h", "--help":
			flagHelp = true
		case "-v", "-V", "--version":
			flagVersion = true
		case "--themes":
			themeTester()
			return
		case "--export-themes":
			exportThemes()
			return
		case "--yaml":
			flagYaml = true
		default:
			args = append(args, arg)
		}

	}

	if flagHelp {
		fmt.Println(usage(keyMap))
		return
	}

	if flagVersion {
		fmt.Println(version)
		return
	}

	stdinIsTty := isatty.IsTerminal(os.Stdin.Fd())
	var fileName string
	var src io.Reader

	if stdinIsTty && len(args) == 0 {
		fmt.Println(usage(keyMap))
		return
	} else if stdinIsTty && len(args) == 1 {
		filePath := args[0]
		f, err := os.Open(filePath)
		if err != nil {
			var pathError *fs.PathError
			if errors.As(err, &pathError) {
				fmt.Println(err)
				os.Exit(1)
			} else {
				panic(err)
			}
		}
		fileName = path.Base(filePath)
		src = f
		hasYamlExt, _ := regexp.MatchString(`(?i)\.ya?ml$`, fileName)
		if !flagYaml && hasYamlExt {
			flagYaml = true
		}
	} else if !stdinIsTty && len(args) == 0 {
		src = os.Stdin
	} else {
		reduce(args)
		return
	}

	data, err := io.ReadAll(src)
	if err != nil {
		panic(err)
	}

	if flagYaml {
		data, err = yaml.YAMLToJSON(data)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
			return
		}
	}

	head, err := parse(data)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
		return
	}

	digInput := textinput.New()
	digInput.Prompt = ""
	digInput.TextStyle = lipgloss.NewStyle().
		Background(lipgloss.Color("7")).
		Foreground(lipgloss.Color("0"))
	digInput.Cursor.Style = lipgloss.NewStyle().
		Background(lipgloss.Color("15")).
		Foreground(lipgloss.Color("0"))

	searchInput := textinput.New()
	searchInput.Prompt = "/"

	help := viewport.New(80, 40)
	help.HighPerformanceRendering = false

	m := &model{
		head:        head,
		top:         head,
		showCursor:  true,
		wrap:        true,
		fileName:    fileName,
		digInput:    digInput,
		searchInput: searchInput,
		search:      newSearch(),
	}

	lipgloss.SetColorProfile(termOutput.ColorProfile())

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithOutput(os.Stderr),
	)
	_, err = p.Run()
	if err != nil {
		panic(err)
	}

	if m.printOnExit {
		fmt.Println(m.cursorValue())
	}
}

type model struct {
	termWidth, termHeight int
	head, top             *node
	cursor                int // cursor position [0, termHeight)
	showCursor            bool
	wrap                  bool
	margin                int
	fileName              string
	digInput              textinput.Model
	searchInput           textinput.Model
	search                *search
	yank                  bool
	showHelp              bool
	help                  viewport.Model
	showPreview           bool
	preview               viewport.Model
	printOnExit           bool
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.help.Width = m.termWidth
		m.help.Height = m.termHeight - 1
		m.preview.Width = m.termWidth
		m.preview.Height = m.termHeight - 1
		wrapAll(m.top, m.termWidth)
		m.redoSearch()
	}

	if m.showHelp {
		return m.handleHelpKey(msg)
	}

	if m.showPreview {
		return m.handlePreviewKey(msg)
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			m.up()

		case tea.MouseWheelDown:
			m.down()

		case tea.MouseLeft:
			m.digInput.Blur()
			m.showCursor = true
			if msg.Y < m.viewHeight() {
				if m.cursor == msg.Y {
					to := m.cursorPointsTo()
					if to != nil {
						if to.isCollapsed() {
							to.expand()
						} else {
							to.collapse()
						}
					}
				} else {
					to := m.at(msg.Y)
					if to != nil {
						m.cursor = msg.Y
						if to.isCollapsed() {
							to.expand()
						}
					}
				}
			}
		}

	case tea.KeyMsg:
		if m.digInput.Focused() {
			return m.handleDigKey(msg)
		}
		if m.searchInput.Focused() {
			return m.handleSearchKey(msg)
		}
		if m.yank {
			return m.handleYankKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) handleDigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case key.Matches(msg, arrowUp):
		m.up()
		m.digInput.SetValue(m.cursorPath())
		m.digInput.CursorEnd()

	case key.Matches(msg, arrowDown):
		m.down()
		m.digInput.SetValue(m.cursorPath())
		m.digInput.CursorEnd()

	case msg.Type == tea.KeyEscape:
		m.digInput.Blur()

	case msg.Type == tea.KeyTab:
		m.digInput.SetValue(m.cursorPath())
		m.digInput.CursorEnd()

	case msg.Type == tea.KeyEnter:
		m.digInput.Blur()
		digPath, ok := jsonpath.Split(m.digInput.Value())
		if ok {
			n := m.selectByPath(digPath)
			if n != nil {
				m.selectNode(n)
			}
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+w"))):
		digPath, ok := jsonpath.Split(m.digInput.Value())
		if ok {
			if len(digPath) > 0 {
				digPath = digPath[:len(digPath)-1]
			}
			n := m.selectByPath(digPath)
			if n != nil {
				m.selectNode(n)
				m.digInput.SetValue(m.cursorPath())
				m.digInput.CursorEnd()
			}
		}

	case key.Matches(msg, textinput.DefaultKeyMap.WordBackward):
		value := m.digInput.Value()
		pth, ok := jsonpath.Split(value[0:m.digInput.Position()])
		if ok {
			if len(pth) > 0 {
				pth = pth[:len(pth)-1]
				m.digInput.SetCursor(len(jsonpath.Join(pth)))
			} else {
				m.digInput.CursorStart()
			}
		}

	case key.Matches(msg, textinput.DefaultKeyMap.WordForward):
		value := m.digInput.Value()
		fullPath, ok1 := jsonpath.Split(value)
		pth, ok2 := jsonpath.Split(value[0:m.digInput.Position()])
		if ok1 && ok2 {
			if len(pth) < len(fullPath) {
				pth = append(pth, fullPath[len(pth)])
				m.digInput.SetCursor(len(jsonpath.Join(pth)))
			} else {
				m.digInput.CursorEnd()
			}
		}

	default:
		if key.Matches(msg, key.NewBinding(key.WithKeys("."))) {
			if m.digInput.Position() == len(m.digInput.Value()) {
				m.digInput.SetValue(m.cursorPath())
				m.digInput.CursorEnd()
			}
		}

		m.digInput, cmd = m.digInput.Update(msg)
		n := m.dig(m.digInput.Value())
		if n != nil {
			m.selectNode(n)
		}
	}
	return m, cmd
}

func (m *model) handleHelpKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keyMap.Quit), key.Matches(msg, keyMap.Help):
			m.showHelp = false
		}
	}
	m.help, cmd = m.help.Update(msg)
	return m, cmd
}

func (m *model) handlePreviewKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, keyMap.Quit),
			key.Matches(msg, keyMap.Preview):
			m.showPreview = false

		case key.Matches(msg, keyMap.Print):
			return m, m.print()
		}
	}
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}

func (m *model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case msg.Type == tea.KeyEscape:
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.doSearch("")
		m.showCursor = true

	case msg.Type == tea.KeyEnter:
		m.searchInput.Blur()
		m.doSearch(m.searchInput.Value())

	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
	}
	return m, cmd
}

func (m *model) handleYankKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, yankPath):
		_ = clipboard.WriteAll(m.cursorPath())
	case key.Matches(msg, yankKey):
		_ = clipboard.WriteAll(m.cursorKey())
	case key.Matches(msg, yankValue):
		_ = clipboard.WriteAll(m.cursorValue())
	}
	m.yank = false
	return m, nil
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyMap.Quit):
		return m, tea.Quit

	case key.Matches(msg, keyMap.Help):
		m.help.SetContent(help(keyMap))
		m.showHelp = true

	case key.Matches(msg, keyMap.Up):
		m.up()

	case key.Matches(msg, keyMap.Down):
		m.down()

	case key.Matches(msg, keyMap.PageUp):
		m.cursor = 0
		for i := 0; i < m.viewHeight(); i++ {
			m.up()
		}

	case key.Matches(msg, keyMap.PageDown):
		m.cursor = m.viewHeight() - 1
		for i := 0; i < m.viewHeight(); i++ {
			m.down()
		}
		m.scrollIntoView()

	case key.Matches(msg, keyMap.HalfPageUp):
		m.cursor = 0
		for i := 0; i < m.viewHeight()/2; i++ {
			m.up()
		}

	case key.Matches(msg, keyMap.HalfPageDown):
		m.cursor = m.viewHeight() - 1
		for i := 0; i < m.viewHeight()/2; i++ {
			m.down()
		}
		m.scrollIntoView()

	case key.Matches(msg, keyMap.GotoTop):
		m.head = m.top
		m.cursor = 0
		m.showCursor = true

	case key.Matches(msg, keyMap.GotoBottom):
		m.head = m.findBottom()
		m.cursor = 0
		m.showCursor = true
		m.scrollIntoView()

	case key.Matches(msg, keyMap.NextSibling):
		pointsTo := m.cursorPointsTo()
		var nextSibling *node
		if pointsTo.end != nil && pointsTo.end.next != nil {
			nextSibling = pointsTo.end.next
		} else {
			nextSibling = pointsTo.next
		}
		if nextSibling != nil {
			m.selectNode(nextSibling)
		}

	case key.Matches(msg, keyMap.PrevSibling):
		pointsTo := m.cursorPointsTo()
		var prevSibling *node
		if pointsTo.parent() != nil && pointsTo.parent().end == pointsTo {
			prevSibling = pointsTo.parent()
		} else if pointsTo.prev != nil {
			prevSibling = pointsTo.prev
			parent := prevSibling.parent()
			if parent != nil && parent.end == prevSibling {
				prevSibling = parent
			}
		}
		if prevSibling != nil {
			m.selectNode(prevSibling)
		}

	case key.Matches(msg, keyMap.Collapse):
		n := m.cursorPointsTo()
		if n.hasChildren() && !n.isCollapsed() {
			n.collapse()
		} else {
			if n.parent() != nil {
				n = n.parent()
			}
		}
		m.selectNode(n)

	case key.Matches(msg, keyMap.Expand):
		m.cursorPointsTo().expand()
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseRecursively):
		n := m.cursorPointsTo()
		if n.hasChildren() {
			n.collapseRecursively()
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.ExpandRecursively):
		n := m.cursorPointsTo()
		if n.hasChildren() {
			n.expandRecursively(0, math.MaxInt)
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseAll):
		n := m.top
		for n != nil {
			n.collapseRecursively()
			if n.end == nil {
				n = nil
			} else {
				n = n.end.next
			}
		}
		m.cursor = 0
		m.head = m.top
		m.showCursor = true

	case key.Matches(msg, keyMap.ExpandAll):
		at := m.cursorPointsTo()
		n := m.top
		for n != nil {
			n.expandRecursively(0, math.MaxInt)
			if n.end == nil {
				n = nil
			} else {
				n = n.end.next
			}
		}
		m.selectNode(at)

	case key.Matches(msg, keyMap.CollapseLevel):
		at := m.cursorPointsTo()
		if at != nil && at.hasChildren() {
			toLevel, _ := strconv.Atoi(msg.String())
			at.collapseRecursively()
			at.expandRecursively(0, toLevel)
			m.showCursor = true
		}

	case key.Matches(msg, keyMap.ToggleWrap):
		at := m.cursorPointsTo()
		m.wrap = !m.wrap
		if m.wrap {
			wrapAll(m.top, m.termWidth)
		} else {
			dropWrapAll(m.top)
		}
		if at.chunk != nil && at.value == nil {
			at = at.parent()
		}
		m.redoSearch()
		m.selectNode(at)

	case key.Matches(msg, keyMap.Yank):
		m.yank = true

	case key.Matches(msg, keyMap.Preview):
		m.showPreview = true
		content := lipgloss.NewStyle().Width(m.termWidth).Render(m.cursorValue())
		m.preview.SetContent(content)
		m.preview.GotoTop()

	case key.Matches(msg, keyMap.Print):
		return m, m.print()

	case key.Matches(msg, keyMap.Dig):
		m.digInput.SetValue(m.cursorPath() + ".")
		m.digInput.CursorEnd()
		m.digInput.Width = m.termWidth - 1
		m.digInput.Focus()

	case key.Matches(msg, keyMap.Search):
		m.searchInput.CursorEnd()
		m.searchInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
		m.searchInput.Focus()

	case key.Matches(msg, keyMap.SearchNext):
		m.selectSearchResult(m.search.cursor + 1)

	case key.Matches(msg, keyMap.SearchPrev):
		m.selectSearchResult(m.search.cursor - 1)
	}
	return m, nil
}

func (m *model) up() {
	m.showCursor = true
	m.cursor--
	if m.cursor < 0 {
		m.cursor = 0
		if m.head.prev != nil {
			m.head = m.head.prev
		}
	}
}

func (m *model) down() {
	m.showCursor = true
	m.cursor++
	n := m.cursorPointsTo()
	if n == nil {
		m.cursor--
		return
	}
	if m.cursor >= m.viewHeight() {
		m.cursor = m.viewHeight() - 1
		if m.head.next != nil {
			m.head = m.head.next
		}
	}
}

func (m *model) visibleLines() int {
	visibleLines := 0
	n := m.head
	for n != nil && visibleLines < m.viewHeight() {
		visibleLines++
		n = n.next
	}
	return visibleLines
}

func (m *model) scrollIntoView() {
	visibleLines := m.visibleLines()
	if m.cursor >= visibleLines {
		m.cursor = visibleLines - 1
	}
	for visibleLines < m.viewHeight() && m.head.prev != nil {
		visibleLines++
		m.cursor++
		m.head = m.head.prev
	}
}

func (m *model) View() string {
	if m.showHelp {
		statusBar := flex(m.termWidth, ": press q or ? to close help", "")
		return m.help.View() + "\n" + string(currentTheme.StatusBar([]byte(statusBar)))
	}

	if m.showPreview {
		statusBar := flex(m.termWidth, m.cursorPath(), m.fileName)
		return m.preview.View() + "\n" + string(currentTheme.StatusBar([]byte(statusBar)))
	}

	var screen []byte
	n := m.head

	printedLines := 0
	for lineNumber := 0; lineNumber < m.viewHeight(); lineNumber++ {
		if n == nil {
			break
		}
		for ident := 0; ident < int(n.depth); ident++ {
			screen = append(screen, ' ', ' ')
		}

		isSelected := m.cursor == lineNumber
		if !m.showCursor {
			isSelected = false // don't highlight the cursor while iterating search results
		}

		if n.key != nil {
			screen = append(screen, m.prettyKey(n, isSelected)...)
			screen = append(screen, colon...)
			isSelected = false // don't highlight the key's value
		}

		screen = append(screen, m.prettyPrint(n, isSelected)...)

		if n.isCollapsed() {
			if n.value[0] == '{' {
				if n.collapsed.key != nil {
					screen = append(screen, currentTheme.Preview(n.collapsed.key)...)
					screen = append(screen, colonPreview...)
				}
				screen = append(screen, dot3...)
				screen = append(screen, closeCurlyBracket...)
			} else if n.value[0] == '[' {
				screen = append(screen, dot3...)
				screen = append(screen, closeSquareBracket...)
			}
			if n.end != nil && n.end.comma {
				screen = append(screen, comma...)
			}
		}
		if n.comma {
			screen = append(screen, comma...)
		}

		screen = append(screen, '\n')
		printedLines++
		n = n.next
	}

	for i := printedLines; i < m.viewHeight(); i++ {
		screen = append(screen, empty...)
		screen = append(screen, '\n')
	}

	if m.digInput.Focused() {
		screen = append(screen, m.digInput.View()...)
	} else {
		statusBar := flex(m.termWidth, m.cursorPath(), m.fileName)
		screen = append(screen, currentTheme.StatusBar([]byte(statusBar))...)
	}

	if m.yank {
		screen = append(screen, '\n')
		screen = append(screen, []byte("(y)value  (p)path  (k)key")...)
	} else if m.searchInput.Focused() {
		screen = append(screen, '\n')
		screen = append(screen, m.searchInput.View()...)
	} else if m.searchInput.Value() != "" {
		screen = append(screen, '\n')
		re, ci := regexCase(m.searchInput.Value())
		re = "/" + re + "/"
		if ci {
			re += "i"
		}
		if m.search.err != nil {
			screen = append(screen, flex(m.termWidth, re, m.search.err.Error())...)
		} else if len(m.search.results) == 0 {
			screen = append(screen, flex(m.termWidth, re, "not found")...)
		} else {
			cursor := fmt.Sprintf("found: [%v/%v]", m.search.cursor+1, len(m.search.results))
			screen = append(screen, flex(m.termWidth, re, cursor)...)
		}
	}

	return string(screen)
}

func (m *model) prettyKey(node *node, selected bool) []byte {
	b := node.key

	style := currentTheme.Key
	if selected {
		style = currentTheme.Cursor
	}

	if indexes, ok := m.search.keys[node]; ok {
		var out []byte
		for i, p := range splitBytesByIndexes(b, indexes) {
			if i%2 == 0 {
				out = append(out, style(p.b)...)
			} else if p.index == m.search.cursor {
				out = append(out, currentTheme.Cursor(p.b)...)
			} else {
				out = append(out, currentTheme.Search(p.b)...)
			}
		}
		return out
	} else {
		return style(b)
	}
}

func (m *model) prettyPrint(node *node, selected bool) []byte {
	var b []byte
	if node.chunk != nil {
		b = node.chunk
	} else {
		b = node.value
	}

	if len(b) == 0 {
		return b
	}

	style := valueStyle(b, selected, node.chunk != nil)

	if indexes, ok := m.search.values[node]; ok {
		var out []byte
		for i, p := range splitBytesByIndexes(b, indexes) {
			if i%2 == 0 {
				out = append(out, style(p.b)...)
			} else if p.index == m.search.cursor {
				out = append(out, currentTheme.Cursor(p.b)...)
			} else {
				out = append(out, currentTheme.Search(p.b)...)
			}
		}
		return out
	} else {
		return style(b)
	}
}

func (m *model) viewHeight() int {
	if m.searchInput.Focused() || m.searchInput.Value() != "" {
		return m.termHeight - 2
	}
	if m.yank {
		return m.termHeight - 2
	}
	return m.termHeight - 1
}

func (m *model) cursorPointsTo() *node {
	return m.at(m.cursor)
}

func (m *model) at(pos int) *node {
	head := m.head
	for i := 0; i < pos; i++ {
		if head == nil {
			break
		}
		head = head.next
	}
	return head
}

func (m *model) findBottom() *node {
	n := m.head
	for n.next != nil {
		if n.end != nil {
			n = n.end
		} else {
			n = n.next
		}
	}
	return n
}

func (m *model) nodeInsideView(n *node) bool {
	if n == nil {
		return false
	}
	head := m.head
	for i := 0; i < m.viewHeight(); i++ {
		if head == nil {
			break
		}
		if head == n {
			return true
		}
		head = head.next
	}
	return false
}

func (m *model) selectNodeInView(n *node) {
	head := m.head
	for i := 0; i < m.viewHeight(); i++ {
		if head == nil {
			break
		}
		if head == n {
			m.cursor = i
			return
		}
		head = head.next
	}
}

func (m *model) selectNode(n *node) {
	m.showCursor = true
	if m.nodeInsideView(n) {
		m.selectNodeInView(n)
		m.scrollIntoView()
	} else {
		m.cursor = 0
		m.head = n
		m.scrollIntoView()
	}
	parent := n.parent()
	for parent != nil {
		parent.expand()
		parent = parent.parent()
	}
}

func (m *model) cursorPath() string {
	path := ""
	at := m.cursorPointsTo()
	for at != nil {
		if at.prev != nil {
			if at.chunk != nil && at.value == nil {
				at = at.parent()
			}
			if at.key != nil {
				quoted := string(at.key)
				unquoted, err := strconv.Unquote(quoted)
				if err == nil && jsonpath.Identifier.MatchString(unquoted) {
					path = "." + unquoted + path
				} else {
					path = "[" + quoted + "]" + path
				}
			} else if at.index >= 0 {
				path = "[" + strconv.Itoa(at.index) + "]" + path
			}
		}
		at = at.parent()
	}
	return path
}

func (m *model) cursorValue() string {
	at := m.cursorPointsTo()
	if at == nil {
		return ""
	}
	parent := at.parent()
	if parent != nil {
		// wrapped string part
		if at.chunk != nil && at.value == nil {
			at = parent
		}
		if len(at.value) == 1 && at.value[0] == '}' || at.value[0] == ']' {
			at = parent
		}
	}

	if len(at.value) > 0 && at.value[0] == '"' {
		str, err := strconv.Unquote(string(at.value))
		if err == nil {
			return str
		}
		return string(at.value)
	}

	var out strings.Builder
	out.Write(at.value)
	out.WriteString("\n")
	if at.hasChildren() {
		it := at.next
		if at.isCollapsed() {
			it = at.collapsed
		}
		for it != nil {
			out.WriteString(strings.Repeat("  ", int(it.depth-at.depth)))
			if it.key != nil {
				out.Write(it.key)
				out.WriteString(": ")
			}
			if it.value != nil {
				out.Write(it.value)
			}
			if it == at.end {
				break
			}
			if it.comma {
				out.WriteString(",")
			}
			out.WriteString("\n")
			if it.chunkEnd != nil {
				it = it.chunkEnd.next
			} else if it.isCollapsed() {
				it = it.collapsed
			} else {
				it = it.next
			}
		}
	}
	return out.String()
}

func (m *model) cursorKey() string {
	at := m.cursorPointsTo()
	if at == nil {
		return ""
	}
	if at.key != nil {
		var v string
		_ = json.Unmarshal(at.key, &v)
		return v
	}
	return strconv.Itoa(at.index)

}

func (m *model) selectByPath(path []any) *node {
	n := m.currentTopNode()
	for _, part := range path {
		if n == nil {
			return nil
		}
		switch part := part.(type) {
		case string:
			n = n.findChildByKey(part)
		case int:
			n = n.findChildByIndex(part)
		}
	}
	return n
}

func (m *model) currentTopNode() *node {
	at := m.cursorPointsTo()
	if at == nil {
		return nil
	}
	for at.parent() != nil {
		at = at.parent()
	}
	return at
}

func (m *model) doSearch(s string) {
	m.search = newSearch()

	if s == "" {
		return
	}

	code, ci := regexCase(s)
	if ci {
		code = "(?i)" + code
	}

	re, err := regexp.Compile(code)
	if err != nil {
		m.search.err = err
		return
	}

	n := m.top
	searchIndex := 0
	for n != nil {
		if n.key != nil {
			indexes := re.FindAllIndex(n.key, -1)
			if len(indexes) > 0 {
				for i, pair := range indexes {
					m.search.results = append(m.search.results, n)
					m.search.keys[n] = append(m.search.keys[n], match{start: pair[0], end: pair[1], index: searchIndex + i})
				}
				searchIndex += len(indexes)
			}
		}
		indexes := re.FindAllIndex(n.value, -1)
		if len(indexes) > 0 {
			for range indexes {
				m.search.results = append(m.search.results, n)
			}
			if n.chunk != nil {
				// String can be split into chunks, so we need to map the indexes to the chunks.
				chunks := [][]byte{n.chunk}
				chunkNodes := []*node{n}

				it := n.next
				for it != nil {
					chunkNodes = append(chunkNodes, it)
					chunks = append(chunks, it.chunk)
					if it == n.chunkEnd {
						break
					}
					it = it.next
				}

				chunkMatches := splitIndexesToChunks(chunks, indexes, searchIndex)
				for i, matches := range chunkMatches {
					m.search.values[chunkNodes[i]] = matches
				}
			} else {
				for i, pair := range indexes {
					m.search.values[n] = append(m.search.values[n], match{start: pair[0], end: pair[1], index: searchIndex + i})
				}
			}
			searchIndex += len(indexes)
		}

		if n.isCollapsed() {
			n = n.collapsed
		} else {
			n = n.next
		}
	}

	m.selectSearchResult(0)
}

func (m *model) selectSearchResult(i int) {
	if len(m.search.results) == 0 {
		return
	}
	if i < 0 {
		i = len(m.search.results) - 1
	}
	if i >= len(m.search.results) {
		i = 0
	}
	m.search.cursor = i
	result := m.search.results[i]
	m.selectNode(result)
	m.showCursor = false
}

func (m *model) redoSearch() {
	if m.searchInput.Value() != "" && len(m.search.results) > 0 {
		cursor := m.search.cursor
		m.doSearch(m.searchInput.Value())
		m.selectSearchResult(cursor)
	}
}

func (m *model) dig(v string) *node {
	p, ok := jsonpath.Split(v)
	if !ok {
		return nil
	}
	at := m.selectByPath(p)
	if at != nil {
		return at
	}

	lastPart := p[len(p)-1]
	searchTerm, ok := lastPart.(string)
	if !ok {
		return nil
	}
	p = p[:len(p)-1]

	at = m.selectByPath(p)
	if at == nil {
		return nil
	}

	keys, nodes := at.children()

	matches := fuzzy.Find(searchTerm, keys)
	if len(matches) == 0 {
		return nil
	}

	return nodes[matches[0].Index]
}

func (m *model) print() tea.Cmd {
	m.printOnExit = true
	return tea.Quit

}
