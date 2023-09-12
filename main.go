package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"github.com/antonmedv/fx/dig"
)

var (
	flagHelp    bool
	flagVersion bool
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

	if stdinIsTty && len(args) == 1 {
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
	} else {
		src = os.Stdin
	}

	data, err := io.ReadAll(src)
	if err != nil {
		panic(err)
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

	m := &model{
		head:     head,
		top:      head,
		wrap:     true,
		digInput: digInput,
		fileName: fileName,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	if err != nil {
		panic(err)
	}
}

type model struct {
	termWidth, termHeight int
	head, top             *node
	cursor                int // cursor position [0, termHeight)
	wrap                  bool
	margin                int
	fileName              string
	digInput              textinput.Model
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		wrapAll(m.top, m.termWidth)

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			m.up()

		case tea.MouseWheelDown:
			m.down()

		case tea.MouseLeft:
			m.digInput.Blur()
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
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *model) handleDigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case msg.Type == tea.KeyEscape, msg.Type == tea.KeyEnter:
		m.digInput.Blur()

	default:
		m.digInput, cmd = m.digInput.Update(msg)
		n := m.dig(m.digInput.Value())
		if n != nil {
			m.selectNode(n)
		}
	}
	return m, cmd
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keyMap.Quit):
		return m, tea.Quit

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

	case key.Matches(msg, keyMap.GotoTop):
		m.head = m.top
		m.cursor = 0

	case key.Matches(msg, keyMap.GotoBottom):
		m.head = m.findBottom()
		m.cursor = 0
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

	case key.Matches(msg, keyMap.CollapseRecursively):
		n := m.cursorPointsTo()
		if n.hasChildren() {
			n.collapseRecursively()
		}

	case key.Matches(msg, keyMap.ExpandRecursively):
		n := m.cursorPointsTo()
		if n.hasChildren() {
			n.expandRecursively()
		}

	case key.Matches(msg, keyMap.CollapseAll):
		m.top.collapseRecursively()
		m.cursor = 0
		m.head = m.top

	case key.Matches(msg, keyMap.ExpandAll):
		at := m.cursorPointsTo()
		m.top.expandRecursively()
		m.selectNode(at)

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
		m.selectNode(at)

	case key.Matches(msg, keyMap.Dig):
		m.digInput.SetValue(m.cursorPath())
		m.digInput.CursorEnd()
		m.digInput.Width = m.termWidth - 1
		m.digInput.Focus()

	}
	return m, nil
}

func (m *model) up() {
	m.cursor--
	if m.cursor < 0 {
		m.cursor = 0
		if m.head.prev != nil {
			m.head = m.head.prev
		}
	}
}

func (m *model) down() {
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
	var screen []byte
	n := m.head

	printedLines := 0
	for i := 0; i < m.viewHeight(); i++ {
		if n == nil {
			break
		}
		for ident := 0; ident < int(n.depth); ident++ {
			screen = append(screen, ' ', ' ')
		}

		valueOrChunk := n.value
		if n.chunk != nil {
			valueOrChunk = n.chunk
		}
		selected := m.cursor == i

		if n.key != nil {
			keyColor := currentTheme.Key
			if m.cursor == i {
				keyColor = currentTheme.Cursor
			}
			screen = append(screen, keyColor(n.key)...)
			screen = append(screen, colon...)
			selected = false // don't highlight the key's value
		}

		screen = append(screen, prettyPrint(valueOrChunk, selected, n.chunk != nil)...)

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
		var statusBar string
		statusBar += m.cursorPath() + " "
		statusBar += strings.Repeat(" ", max(0, m.termWidth-len(statusBar)-len(m.fileName)))
		statusBar += m.fileName
		screen = append(screen, currentTheme.StatusBar([]byte(statusBar))...)
	}

	return string(screen)
}

func (m *model) viewHeight() int {
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
	if m.nodeInsideView(n) {
		m.selectNodeInView(n)
		m.scrollIntoView()
	} else {
		m.cursor = 0
		m.head = n
		m.scrollIntoView()
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
				if err != nil {
					panic(err)
				}
				if identifier.MatchString(unquoted) {
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

func (m *model) dig(value string) *node {
	p, ok := dig.SplitPath(value)
	if !ok {
		return nil
	}
	n := m.top
	for _, part := range p {
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
