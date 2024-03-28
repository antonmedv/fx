package main

import (
	"encoding/json"
	"errors"
	"flag"
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

	"github.com/antonmedv/fx/internal/complete"
	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
	jsonpath "github.com/antonmedv/fx/path"
)

var (
	flagYaml bool
	flagComp bool
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

	if complete.Complete() {
		os.Exit(0)
		return
	}

	var args []string
	for _, arg := range os.Args[1:] {
		if strings.HasPrefix(arg, "--comp") {
			flagComp = true
			continue
		}
		switch arg {
		case "-h", "--help":
			fmt.Println(usage(keyMap))
			return
		case "-v", "-V", "--version":
			fmt.Println(version)
			return
		case "--themes":
			theme.ThemeTester()
			return
		case "--export-themes":
			theme.ExportThemes()
			return
		default:
			args = append(args, arg)
		}
	}

	if flagComp {
		shell := flag.String("comp", "", "")
		flag.Parse()
		switch *shell {
		case "bash":
			fmt.Print(complete.Bash())
		case "zsh":
			fmt.Print(complete.Zsh())
		case "fish":
			fmt.Print(complete.Fish())
		default:
			fmt.Println("unknown shell type")
		}
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
		reduce(os.Args[1:])
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

	head, err := Parse(data)
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

	lipgloss.SetColorProfile(theme.TermOutput.ColorProfile())

	withMouse := tea.WithMouseCellMotion()
	if _, ok := os.LookupEnv("FX_NO_MOUSE"); ok {
		withMouse = tea.WithAltScreen()
	}

	p := tea.NewProgram(m,
		tea.WithAltScreen(),
		withMouse,
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
	head, top             *Node
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
		WrapAll(m.top, m.termWidth)
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
						if to.IsCollapsed() {
							to.Expand()
						} else {
							to.Collapse()
						}
					}
				} else {
					to := m.at(msg.Y)
					if to != nil {
						m.cursor = msg.Y
						if to.IsCollapsed() {
							to.Expand()
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
	case key.Matches(msg, yankValueY, yankValueV):
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
		var nextSibling *Node
		if pointsTo.End != nil && pointsTo.End.Next != nil {
			nextSibling = pointsTo.End.Next
		} else {
			nextSibling = pointsTo.Next
		}
		if nextSibling != nil {
			m.selectNode(nextSibling)
		}

	case key.Matches(msg, keyMap.PrevSibling):
		pointsTo := m.cursorPointsTo()
		var prevSibling *Node
		if pointsTo.Parent() != nil && pointsTo.Parent().End == pointsTo {
			prevSibling = pointsTo.Parent()
		} else if pointsTo.Prev != nil {
			prevSibling = pointsTo.Prev
			parent := prevSibling.Parent()
			if parent != nil && parent.End == prevSibling {
				prevSibling = parent
			}
		}
		if prevSibling != nil {
			m.selectNode(prevSibling)
		}

	case key.Matches(msg, keyMap.Collapse):
		n := m.cursorPointsTo()
		if n.HasChildren() && !n.IsCollapsed() {
			n.Collapse()
		} else {
			if n.Parent() != nil {
				n = n.Parent()
			}
		}
		m.selectNode(n)

	case key.Matches(msg, keyMap.Expand):
		m.cursorPointsTo().Expand()
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseRecursively):
		n := m.cursorPointsTo()
		if n.HasChildren() {
			n.CollapseRecursively()
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.ExpandRecursively):
		n := m.cursorPointsTo()
		if n.HasChildren() {
			n.ExpandRecursively(0, math.MaxInt)
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseAll):
		n := m.top
		for n != nil {
			n.CollapseRecursively()
			if n.End == nil {
				n = nil
			} else {
				n = n.End.Next
			}
		}
		m.cursor = 0
		m.head = m.top
		m.showCursor = true

	case key.Matches(msg, keyMap.ExpandAll):
		at := m.cursorPointsTo()
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

	case key.Matches(msg, keyMap.CollapseLevel):
		at := m.cursorPointsTo()
		if at != nil && at.HasChildren() {
			toLevel, _ := strconv.Atoi(msg.String())
			at.CollapseRecursively()
			at.ExpandRecursively(0, toLevel)
			m.showCursor = true
		}

	case key.Matches(msg, keyMap.ToggleWrap):
		at := m.cursorPointsTo()
		m.wrap = !m.wrap
		if m.wrap {
			WrapAll(m.top, m.termWidth)
		} else {
			DropWrapAll(m.top)
		}
		if at.Chunk != nil && at.Value == nil {
			at = at.Parent()
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
		if m.head.Prev != nil {
			m.head = m.head.Prev
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
		if m.head.Next != nil {
			m.head = m.head.Next
		}
	}
}

func (m *model) visibleLines() int {
	visibleLines := 0
	n := m.head
	for n != nil && visibleLines < m.viewHeight() {
		visibleLines++
		n = n.Next
	}
	return visibleLines
}

func (m *model) scrollIntoView() {
	visibleLines := m.visibleLines()
	if m.cursor >= visibleLines {
		m.cursor = visibleLines - 1
	}
	for visibleLines < m.viewHeight() && m.head.Prev != nil {
		visibleLines++
		m.cursor++
		m.head = m.head.Prev
	}
}

func (m *model) View() string {
	if m.showHelp {
		statusBar := flex(m.termWidth, ": press q or ? to close help", "")
		return m.help.View() + "\n" + string(theme.CurrentTheme.StatusBar([]byte(statusBar)))
	}

	if m.showPreview {
		statusBar := flex(m.termWidth, m.cursorPath(), m.fileName)
		return m.preview.View() + "\n" + string(theme.CurrentTheme.StatusBar([]byte(statusBar)))
	}

	var screen []byte
	n := m.head

	printedLines := 0
	for lineNumber := 0; lineNumber < m.viewHeight(); lineNumber++ {
		if n == nil {
			break
		}
		for ident := 0; ident < int(n.Depth); ident++ {
			screen = append(screen, ' ', ' ')
		}

		isSelected := m.cursor == lineNumber
		if !m.showCursor {
			isSelected = false // don't highlight the cursor while iterating search results
		}

		if n.Key != nil {
			screen = append(screen, m.prettyKey(n, isSelected)...)
			screen = append(screen, theme.Colon...)
			isSelected = false // don't highlight the key's value
		}

		screen = append(screen, m.prettyPrint(n, isSelected)...)

		if n.IsCollapsed() {
			if n.Value[0] == '{' {
				if n.Collapsed.Key != nil {
					screen = append(screen, theme.CurrentTheme.Preview(n.Collapsed.Key)...)
					screen = append(screen, theme.ColonPreview...)
				}
				screen = append(screen, theme.Dot3...)
				screen = append(screen, theme.CloseCurlyBracket...)
			} else if n.Value[0] == '[' {
				screen = append(screen, theme.Dot3...)
				screen = append(screen, theme.CloseSquareBracket...)
			}
			if n.End != nil && n.End.Comma {
				screen = append(screen, theme.Comma...)
			}
		}
		if n.Comma {
			screen = append(screen, theme.Comma...)
		}

		if theme.ShowSizes && len(n.Value) > 0 && (n.Value[0] == '{' || n.Value[0] == '[') {
			if n.IsCollapsed() || n.Size > 1 {
				screen = append(screen, theme.CurrentTheme.Size([]byte(fmt.Sprintf(" // %d", n.Size)))...)
			}
		}

		screen = append(screen, '\n')
		printedLines++
		n = n.Next
	}

	for i := printedLines; i < m.viewHeight(); i++ {
		screen = append(screen, theme.Empty...)
		screen = append(screen, '\n')
	}

	if m.digInput.Focused() {
		screen = append(screen, m.digInput.View()...)
	} else {
		statusBar := flex(m.termWidth, m.cursorPath(), m.fileName)
		screen = append(screen, theme.CurrentTheme.StatusBar([]byte(statusBar))...)
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

func (m *model) prettyKey(node *Node, selected bool) []byte {
	b := node.Key

	style := theme.CurrentTheme.Key
	if selected {
		style = theme.CurrentTheme.Cursor
	}

	if indexes, ok := m.search.keys[node]; ok {
		var out []byte
		for i, p := range splitBytesByIndexes(b, indexes) {
			if i%2 == 0 {
				out = append(out, style(p.b)...)
			} else if p.index == m.search.cursor {
				out = append(out, theme.CurrentTheme.Cursor(p.b)...)
			} else {
				out = append(out, theme.CurrentTheme.Search(p.b)...)
			}
		}
		return out
	} else {
		return style(b)
	}
}

func (m *model) prettyPrint(node *Node, selected bool) []byte {
	var b []byte
	if node.Chunk != nil {
		b = node.Chunk
	} else {
		b = node.Value
	}

	if len(b) == 0 {
		return b
	}

	style := theme.Value(b, selected, node.Chunk != nil)

	if indexes, ok := m.search.values[node]; ok {
		var out []byte
		for i, p := range splitBytesByIndexes(b, indexes) {
			if i%2 == 0 {
				out = append(out, style(p.b)...)
			} else if p.index == m.search.cursor {
				out = append(out, theme.CurrentTheme.Cursor(p.b)...)
			} else {
				out = append(out, theme.CurrentTheme.Search(p.b)...)
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

func (m *model) cursorPointsTo() *Node {
	return m.at(m.cursor)
}

func (m *model) at(pos int) *Node {
	head := m.head
	for i := 0; i < pos; i++ {
		if head == nil {
			break
		}
		head = head.Next
	}
	return head
}

func (m *model) findBottom() *Node {
	n := m.head
	for n.Next != nil {
		if n.End != nil {
			n = n.End
		} else {
			n = n.Next
		}
	}
	return n
}

func (m *model) nodeInsideView(n *Node) bool {
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
		head = head.Next
	}
	return false
}

func (m *model) selectNodeInView(n *Node) {
	head := m.head
	for i := 0; i < m.viewHeight(); i++ {
		if head == nil {
			break
		}
		if head == n {
			m.cursor = i
			return
		}
		head = head.Next
	}
}

func (m *model) selectNode(n *Node) {
	m.showCursor = true
	if m.nodeInsideView(n) {
		m.selectNodeInView(n)
		m.scrollIntoView()
	} else {
		m.cursor = 0
		m.head = n
		m.scrollIntoView()
	}
	parent := n.Parent()
	for parent != nil {
		parent.Expand()
		parent = parent.Parent()
	}
}

func (m *model) cursorPath() string {
	path := ""
	at := m.cursorPointsTo()
	for at != nil {
		if at.Prev != nil {
			if at.Chunk != nil && at.Value == nil {
				at = at.Parent()
			}
			if at.Key != nil {
				quoted := string(at.Key)
				unquoted, err := strconv.Unquote(quoted)
				if err == nil && jsonpath.Identifier.MatchString(unquoted) {
					path = "." + unquoted + path
				} else {
					path = "[" + quoted + "]" + path
				}
			} else if at.Index >= 0 {
				path = "[" + strconv.Itoa(at.Index) + "]" + path
			}
		}
		at = at.Parent()
	}
	return path
}

func (m *model) cursorValue() string {
	at := m.cursorPointsTo()
	if at == nil {
		return ""
	}
	parent := at.Parent()
	if parent != nil {
		// wrapped string part
		if at.Chunk != nil && at.Value == nil {
			at = parent
		}
		if len(at.Value) == 1 && at.Value[0] == '}' || at.Value[0] == ']' {
			at = parent
		}
	}

	if len(at.Value) > 0 && at.Value[0] == '"' {
		str, err := strconv.Unquote(string(at.Value))
		if err == nil {
			return str
		}
		return string(at.Value)
	}

	var out strings.Builder
	out.Write(at.Value)
	out.WriteString("\n")
	if at.HasChildren() {
		it := at.Next
		if at.IsCollapsed() {
			it = at.Collapsed
		}
		for it != nil {
			out.WriteString(strings.Repeat("  ", int(it.Depth-at.Depth)))
			if it.Key != nil {
				out.Write(it.Key)
				out.WriteString(": ")
			}
			if it.Value != nil {
				out.Write(it.Value)
			}
			if it == at.End {
				break
			}
			if it.Comma {
				out.WriteString(",")
			}
			out.WriteString("\n")
			if it.ChunkEnd != nil {
				it = it.ChunkEnd.Next
			} else if it.IsCollapsed() {
				it = it.Collapsed
			} else {
				it = it.Next
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
	if at.IsWrap() {
		at = at.Parent()
	}
	if at.Key != nil {
		var v string
		_ = json.Unmarshal(at.Key, &v)
		return v
	}
	return strconv.Itoa(at.Index)

}

func (m *model) selectByPath(path []any) *Node {
	n := m.currentTopNode()
	for _, part := range path {
		if n == nil {
			return nil
		}
		switch part := part.(type) {
		case string:
			n = n.FindChildByKey(part)
		case int:
			n = n.FindChildByIndex(part)
		}
	}
	return n
}

func (m *model) currentTopNode() *Node {
	at := m.cursorPointsTo()
	if at == nil {
		return nil
	}
	for at.Parent() != nil {
		at = at.Parent()
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
		if n.Key != nil {
			indexes := re.FindAllIndex(n.Key, -1)
			if len(indexes) > 0 {
				for i, pair := range indexes {
					m.search.results = append(m.search.results, n)
					m.search.keys[n] = append(m.search.keys[n], match{start: pair[0], end: pair[1], index: searchIndex + i})
				}
				searchIndex += len(indexes)
			}
		}
		indexes := re.FindAllIndex(n.Value, -1)
		if len(indexes) > 0 {
			for range indexes {
				m.search.results = append(m.search.results, n)
			}
			if n.Chunk != nil {
				// String can be split into chunks, so we need to map the indexes to the chunks.
				chunks := [][]byte{n.Chunk}
				chunkNodes := []*Node{n}

				it := n.Next
				for it != nil {
					chunkNodes = append(chunkNodes, it)
					chunks = append(chunks, it.Chunk)
					if it == n.ChunkEnd {
						break
					}
					it = it.Next
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

		if n.IsCollapsed() {
			n = n.Collapsed
		} else {
			n = n.Next
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

func (m *model) dig(v string) *Node {
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

	keys, nodes := at.Children()

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
