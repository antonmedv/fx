package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/antonmedv/clipboard"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"github.com/antonmedv/fx/internal/complete"
	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/fuzzy"
	"github.com/antonmedv/fx/internal/jsonpath"
	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
	"github.com/antonmedv/fx/internal/toml"
	"github.com/antonmedv/fx/internal/utils"
)

var (
	flagYaml     bool
	flagToml     bool
	flagRaw      bool
	flagSlurp    bool
	flagComp     bool
	flagStrict   bool
	flagNoInline bool
)

var flags = []string{
	"--help",
	"--raw",
	"--slurp",
	"--themes",
	"--version",
	"--yaml",
	"--toml",
	"--strict",
	"--no-inline",
}

func init() {
	for _, name := range flags {
		complete.Flags = append(complete.Flags, complete.Reply{name, name, "flag"})
	}
}

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
		defer f.Close()
		defer pprof.StopCPUProfile()
		memProf, err := os.Create("mem.prof")
		if err != nil {
			panic(err)
		}
		defer memProf.Close()
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
			fmt.Println(usage())
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
		case "--yaml":
			flagYaml = true
		case "--toml":
			flagToml = true
		case "--raw", "-r":
			flagRaw = true
		case "--slurp", "-s":
			flagSlurp = true
		case "-rs", "-sr":
			flagRaw = true
			flagSlurp = true
		case "--strict":
			flagStrict = true
		case "--no-inline":
			flagNoInline = true
		case "--game-of-life":
			utils.GameOfLife()
			return
		default:
			args = append(args, arg)
		}
	}

	if (flagYaml || flagToml) && flagRaw {
		println("Error: can't use --yaml/--toml and --raw flags together")
		os.Exit(1)
	}
	if flagYaml && flagToml {
		println("Error: can't use both --yaml and --toml flags together")
		os.Exit(1)
	}

	if flagComp {
		shell := flag.String("comp", "", "")
		flag.Parse()
		switch *shell {
		case "bash":
			fmt.Print(complete.Bash)
		case "zsh":
			fmt.Print(complete.Zsh)
		case "fish":
			fmt.Print(complete.Fish)
		default:
			fmt.Println("unknown shell type")
		}
		return
	}

	fd := os.Stdin.Fd()
	stdinIsTty := isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)

	var fileName string
	var src io.Reader

	if stdinIsTty {
		if len(args) == 0 {
			// $ fx
			fmt.Println(usage())
			return
		} else {
			// $ fx file.json arg*
			filePath := args[0]
			src = open(filePath, &flagYaml, &flagToml)
			engine.FilePath = filePath
			fileName = filepath.Base(filePath)
			args = args[1:]
		}
	} else {
		// cat file.json | fx arg*
		src = os.Stdin
	}

	var parser engine.Parser

	if flagYaml {
		b, err := io.ReadAll(src)
		if err != nil {
			panic(err)
		}
		jsonBytes, err := parseYAML(b)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
			return
		}
		parser = NewJsonParser(bytes.NewReader(jsonBytes), flagStrict)
	} else if flagToml {
		b, err := io.ReadAll(src)
		if err != nil {
			panic(err)
		}
		jsonBytes, err := toml.ToJSON(b)
		if err != nil {
			fmt.Print(err.Error())
			os.Exit(1)
			return
		}
		parser = NewJsonParser(bytes.NewReader(jsonBytes), flagStrict)
	} else if flagRaw {
		parser = NewLineParser(src)
	} else {
		parser = NewJsonParser(src, flagStrict)
	}

	if len(args) > 0 || flagSlurp {
		opts := engine.Options{
			Slurp:      flagSlurp,
			WithInline: !flagNoInline,
			WriteOut:   func(s string) { fmt.Println(s) },
			WriteErr:   func(s string) { fmt.Fprintln(os.Stderr, s) },
		}
		exitCode := engine.Start(parser, args, opts)
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return
	}

	commandInput := textinput.New()
	commandInput.Prompt = ":"

	searchInput := textinput.New()
	searchInput.Prompt = "/"

	gotoSymbolInput := textinput.New()
	gotoSymbolInput.Prompt = "@"

	previewSearchInput := textinput.New()
	previewSearchInput.Prompt = "/"

	spinnerModel := spinner.New()
	spinnerModel.Spinner = spinner.MiniDot

	collapsed := false
	if _, ok := os.LookupEnv("FX_COLLAPSED"); ok {
		collapsed = true
	}

	showLineNumbers := false
	if _, ok := os.LookupEnv("FX_LINE_NUMBERS"); ok {
		showLineNumbers = true
	}

	showSizes := false
	showSizesValue, ok := os.LookupEnv("FX_SHOW_SIZE")
	if ok {
		showSizesValue := strings.ToLower(showSizesValue)
		showSizes = showSizesValue == "true" || showSizesValue == "yes" || showSizesValue == "on" || showSizesValue == "1"
	}

	m := &model{
		suspending:          false,
		showCursor:          true,
		wrap:                true,
		collapsed:           collapsed,
		showSizes:           showSizes,
		showLineNumbers:     showLineNumbers,
		fileName:            fileName,
		gotoSymbolInput:     gotoSymbolInput,
		commandInput:        commandInput,
		searchInput:         searchInput,
		search:              newSearch(),
		previewSearchInput:  previewSearchInput,
		previewSearchCursor: -1,
		spinner:             spinnerModel,
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

	go func() {
		firstOk := false
		for {
			node, err := parser.Parse()
			if err != nil {
				if err == io.EOF {
					p.Send(eofMsg{})
					break
				}
				if flagStrict {
					p.Send(errorMsg{err: err})
					break
				}
				textNode := parser.Recover()
				if !firstOk && !strings.HasPrefix(textNode.Value, "HTTP") {
					p.Send(errorMsg{err: err})
					break
				}
				p.Send(nodeMsg{node: textNode})
			} else {
				firstOk = true
				p.Send(nodeMsg{node: node})
			}
		}
	}()

	_, err := p.Run()
	if err != nil {
		panic(err)
	}

	if m.printErrorOnExit != nil {
		fmt.Println(m.printErrorOnExit.Error())
	} else if m.printOnExit {
		fmt.Println(m.cursorValue())
	} else {
		exit()
	}
}

type model struct {
	termWidth, termHeight int
	head, top, bottom     *Node
	eof                   bool
	cursor                int // cursor position [0, termHeight)
	suspending            bool
	showCursor            bool
	wrap                  bool
	collapsed             bool
	showShowSelector      bool
	showSizes             bool
	showLineNumbers       bool
	totalLines            int
	fileName              string
	gotoSymbolInput       textinput.Model
	commandInput          textinput.Model
	searchInput           textinput.Model
	search                *search
	searching             bool          // search in progress
	searchCancel          chan struct{} // cancel channel for search
	searchID              uint64        // increments with each search to detect stale results
	yank                  bool
	showHelp              bool
	help                  viewport.Model
	showPreview           bool
	preview               viewport.Model
	previewValue          string
	previewSearchInput    textinput.Model
	previewSearchResults  []int
	previewSearchCursor   int
	printOnExit           bool
	printErrorOnExit      error
	spinner               spinner.Model
	locationHistory       []location
	locationIndex         int // position in locationHistory
	keysIndex             []string
	keysIndexNodes        []*Node
	fuzzyMatch            *fuzzy.Match
	deletePending         bool
}

type location struct {
	head *Node
	node *Node
}

type nodeMsg struct {
	node *Node
}

type errorMsg struct {
	err error
}

type eofMsg struct{}

type searchResultMsg struct {
	id     uint64
	query  string
	search *search
}

type searchCancelledMsg struct {
	id uint64
}

func (m *model) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.help.Width = m.termWidth
		m.help.Height = m.termHeight - 1
		m.preview.Width = m.termWidth
		m.preview.Height = m.termHeight - 1
		Wrap(m.top, m.viewWidth())
		m.redoSearch()

	case eofMsg:
		m.eof = true
		return m, nil

	case errorMsg:
		m.printErrorOnExit = msg.err
		return m, tea.Quit

	case nodeMsg:
		if m.wrap {
			Wrap(msg.node, m.viewWidth())
		}
		if m.collapsed {
			msg.node.CollapseRecursively()
		}
		m.totalLines = msg.node.Bottom().LineNumber

		if m.head == nil {
			m.head = msg.node
			m.top = msg.node
			m.bottom = msg.node
		} else {
			to, ok := m.cursorPointsTo()
			if !ok {
				return m, nil
			}
			scrollToBottom := to == m.bottom.Bottom()
			msg.node.Index = -1 // To fix the statusbar path (to show .key instead of [0].key).
			m.bottom.Adjacent(msg.node)
			m.bottom = msg.node
			if scrollToBottom {
				m.scrollToBottom()
			}
		}
		return m, nil

	case spinner.TickMsg:
		if !m.eof || m.searching {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case searchResultMsg:
		if msg.id != m.searchID {
			return m, nil
		}
		m.searching = false
		m.searchCancel = nil
		if msg.search != nil {
			m.search = msg.search
			m.selectSearchResult(0)
		}
		return m, nil

	case searchCancelledMsg:
		if msg.id != m.searchID {
			return m, nil
		}
		m.searching = false
		m.searchCancel = nil
		return m, nil

	case tea.ResumeMsg:
		m.suspending = false
		return m, nil
	}

	if m.showHelp {
		return m.handleHelpKey(msg)
	}

	if m.showPreview {
		return m.handlePreviewKey(msg)
	}

	switch msg := msg.(type) {
	case tea.MouseMsg:
		m.handlePendingDelete(msg)

		switch {
		case msg.Button == tea.MouseButtonWheelUp:
			m.up()

		case msg.Button == tea.MouseButtonWheelDown:
			m.down()

		case msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress:
			m.showCursor = true
			if msg.Y < m.viewHeight() {
				if m.cursor == msg.Y {
					to, ok := m.cursorPointsTo()
					if ok {
						if to.IsCollapsed() {
							to.Expand()
						} else {
							to.Collapse()
						}

						value, isRef := isRefNode(to)
						if isRef {
							refPath, ok := jsonpath.ParseSchemaRef(value)
							if ok {
								m.selectNode(m.findByPath(refPath))
								m.recordHistory()
							}
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
				m.recordHistory()
			}
		}

	case tea.KeyMsg:
		if m.commandInput.Focused() {
			return m.handleGotoLineKey(msg)
		}
		if m.searchInput.Focused() {
			return m.handleSearchKey(msg)
		}
		if m.gotoSymbolInput.Focused() {
			return m.handleGotoSymbolKey(msg)
		}
		if m.yank {
			return m.handleYankKey(msg)
		}
		if m.showShowSelector {
			return m.handleShowSelectorKey(msg)
		}
		return m.handleKey(msg)
	}
	return m, nil
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

func (m *model) handleGotoLineKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case msg.Type == tea.KeyEscape:
		m.commandInput.Blur()
		m.commandInput.SetValue("")
		m.showCursor = true

	case msg.Type == tea.KeyEnter:
		m.commandInput.Blur()
		command := m.commandInput.Value()
		m.commandInput.SetValue("")
		return m.runCommand(command)

	default:
		m.commandInput, cmd = m.commandInput.Update(msg)
	}
	return m, cmd
}

func (m *model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch {
	case msg.Type == tea.KeyEscape:
		m.cancelSearch()
		m.search = newSearch()
		m.searchInput.Blur()
		m.searchInput.SetValue("")
		m.showCursor = true

	case msg.Type == tea.KeyEnter:
		m.searchInput.Blur()
		m.cancelSearch()
		m.search = newSearch()
		return m, m.doSearch(m.searchInput.Value())

	default:
		m.searchInput, cmd = m.searchInput.Update(msg)
	}
	return m, cmd
}

func (m *model) handleGotoSymbolKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg.Type {
	case tea.KeyEscape, tea.KeyEnter, tea.KeyUp, tea.KeyDown:
		m.gotoSymbolInput.Blur()
		m.gotoSymbolInput.SetValue("")
		m.recordHistory()

	default:
		m.gotoSymbolInput, cmd = m.gotoSymbolInput.Update(msg)
		pattern := []rune(m.gotoSymbolInput.Value())
		found := fuzzy.Find(pattern, m.keysIndex)
		if found != nil {
			m.fuzzyMatch = found
			m.selectNode(m.keysIndexNodes[found.Index])
		}
	}

	switch msg.Type {
	case tea.KeyUp:
		m.up()

	case tea.KeyDown:
		m.down()
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
	case key.Matches(msg, yankKeyValue):
		k := m.cursorKey()
		v := m.cursorValue()
		keyValue := k + ": " + v
		_ = clipboard.WriteAll(keyValue)
	}
	m.yank = false
	return m, nil
}

func (m *model) handleShowSelectorKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, showSizes):
		m.showSizes = !m.showSizes
	case key.Matches(msg, showLineNumbers):
		m.showLineNumbers = !m.showLineNumbers
		Wrap(m.top, m.viewWidth())
	}
	m.showShowSelector = false
	return m, nil
}

func (m *model) handlePendingDelete(msg tea.Msg) {
	// Handle potential 'dd' sequence for delete
	if m.deletePending {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			if key.Matches(keyMsg, keyMap.Delete) {
				m.deleteAtCursor()
				m.deletePending = true
				return
			}
		}
		m.deletePending = false
	}
}

func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.handlePendingDelete(msg)

	switch {
	case key.Matches(msg, keyMap.Suspend):
		m.suspending = true
		return m, tea.Suspend

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
		m.cursor = m.viewHeight() - 1
		m.showCursor = true
		m.scrollBackward(max(0, m.viewHeight()-2))
		m.scrollIntoView() // As the cursor is at the bottom, and it may be empty.
		m.recordHistory()

	case key.Matches(msg, keyMap.PageDown):
		m.cursor = 0
		m.showCursor = true
		m.scrollForward(max(0, m.viewHeight()-2))
		m.recordHistory()

	case key.Matches(msg, keyMap.HalfPageUp):
		m.showCursor = true
		m.scrollBackward(m.viewHeight() / 2)
		m.scrollIntoView() // As the cursor stays at the same position, and it may be empty.
		m.recordHistory()

	case key.Matches(msg, keyMap.HalfPageDown):
		m.showCursor = true
		m.scrollForward(m.viewHeight() / 2)
		m.scrollIntoView() // As the cursor stays at the same position, and it may be empty.
		m.recordHistory()

	case key.Matches(msg, keyMap.GotoTop):
		m.head = m.top
		m.cursor = 0
		m.showCursor = true
		m.recordHistory()

	case key.Matches(msg, keyMap.GotoBottom):
		m.scrollToBottom()
		m.recordHistory()

	case key.Matches(msg, keyMap.NextSibling):
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

	case key.Matches(msg, keyMap.PrevSibling):
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

	case key.Matches(msg, keyMap.Collapse):
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

	case key.Matches(msg, keyMap.Expand):
		n, ok := m.cursorPointsTo()
		if !ok {
			return m, nil
		}
		n.Expand()
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseRecursively):
		n, ok := m.cursorPointsTo()
		if !ok {
			return m, nil
		}
		if n.HasChildren() {
			n.CollapseRecursively()
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.ExpandRecursively):
		n, ok := m.cursorPointsTo()
		if !ok {
			return m, nil
		}
		if n.HasChildren() {
			n.ExpandRecursively(0, math.MaxInt)
		}
		m.showCursor = true

	case key.Matches(msg, keyMap.CollapseAll):
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

	case key.Matches(msg, keyMap.ExpandAll):
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

	case key.Matches(msg, keyMap.CollapseLevel):
		at, ok := m.cursorPointsTo()
		if ok && at.HasChildren() {
			toLevel, _ := strconv.Atoi(msg.String())
			at.CollapseRecursively()
			at.ExpandRecursively(0, toLevel)
			m.showCursor = true
		}

	case key.Matches(msg, keyMap.ToggleWrap):
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
		if at.Chunk != "" && at.Value == "" {
			at = at.Parent
		}
		m.redoSearch()
		m.selectNode(at)

	case key.Matches(msg, keyMap.ShowSelector):
		m.showShowSelector = true

	case key.Matches(msg, keyMap.Yank):
		m.yank = true

	case key.Matches(msg, keyMap.Preview):
		m.showPreview = true
		value := m.cursorValue()
		var view string
		if decodedValue, err := base64.StdEncoding.DecodeString(value); err == nil {
			img, err := utils.DrawImage(bytes.NewReader(decodedValue), m.termWidth, m.termHeight)
			if err == nil {
				view = strings.TrimRight(img, "\n")
			}
		}
		if view == "" {
			view = lipgloss.NewStyle().Width(m.termWidth).Render(value)
		}
		m.previewValue = value
		m.previewSearchInput.SetValue("")
		m.previewSearchResults = nil
		m.previewSearchCursor = -1
		m.preview.SetContent(view)
		m.preview.GotoTop()

	case key.Matches(msg, keyMap.Print):
		return m, m.print()

	case key.Matches(msg, keyMap.Open):
		return m, m.open()

	case key.Matches(msg, keyMap.DecodeJSON):
		if m.decodeJSONString() {
			m.recordHistory()
		}

	case key.Matches(msg, keyMap.EncodeJSON):
		if m.encodeJSONValue() {
			m.recordHistory()
		}

	case key.Matches(msg, keyMap.GotoSymbol):
		m.gotoSymbolInput.CursorEnd()
		m.gotoSymbolInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
		m.gotoSymbolInput.Focus()
		m.createKeysIndex()

	case key.Matches(msg, keyMap.GotoRef):
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

	case key.Matches(msg, keyMap.CommandLine):
		m.commandInput.CursorEnd()
		m.commandInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
		m.commandInput.Focus()

	case key.Matches(msg, keyMap.Search):
		m.searchInput.CursorEnd()
		m.searchInput.Width = m.termWidth - 2 // -1 for the prompt, -1 for the cursor
		m.searchInput.Focus()

	case key.Matches(msg, keyMap.SearchNext):
		m.selectSearchResult(m.search.cursor + 1)
		m.recordHistory()

	case key.Matches(msg, keyMap.SearchPrev):
		m.selectSearchResult(m.search.cursor - 1)
		m.recordHistory()

	case key.Matches(msg, keyMap.GoBack):
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

	case key.Matches(msg, keyMap.GoForward):
		if m.locationIndex < len(m.locationHistory)-1 {
			m.locationIndex++
			loc := m.locationHistory[m.locationIndex]
			m.selectNode(loc.head)
			m.selectNode(loc.node)
		}

	case key.Matches(msg, keyMap.Delete):
		m.deletePending = true
	}
	return m, nil
}

func (m *model) up() {
	if m.head == nil {
		return
	}
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
	if m.head == nil {
		return
	}
	m.showCursor = true
	m.cursor++
	_, ok := m.cursorPointsTo()
	if !ok {
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

func (m *model) recordHistory() {
	at, ok := m.cursorPointsTo()
	if !ok {
		return
	}
	if at.Chunk != "" && at.Value == "" {
		// We at the wrapped string, save the location of the original string node.
		at = at.Parent
	}
	if len(m.locationHistory) > 0 && m.locationHistory[len(m.locationHistory)-1].node == at {
		return
	}
	if m.locationIndex < len(m.locationHistory) {
		m.locationHistory = m.locationHistory[:m.locationIndex+1]
	}
	m.locationHistory = append(m.locationHistory, location{
		head: m.head,
		node: at,
	})
	m.locationIndex = len(m.locationHistory)
}

func (m *model) scrollToBottom() {
	if m.bottom == nil {
		return
	}
	m.head = m.bottom.Bottom()
	m.cursor = 0
	m.showCursor = true
	m.scrollIntoView()
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
	if m.head == nil {
		return
	}
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

func (m *model) scrollBackward(lines int) {
	it := m.head
	for it.Prev != nil {
		it = it.Prev
		if lines--; lines == 0 {
			break
		}
	}
	m.head = it
}

func (m *model) scrollForward(lines int) {
	if m.head == nil {
		return
	}
	it := m.head
	for it.Next != nil {
		it = it.Next
		if lines--; lines == 0 {
			break
		}
	}
	m.head = it
}

func (m *model) prettyKey(node *Node, selected bool) []byte {
	b := node.Key

	style := theme.CurrentTheme.Key
	if selected {
		style = theme.CurrentTheme.Cursor
	}

	if indexes, ok := m.search.keys[node]; ok {
		var out []byte
		for i, p := range splitByIndexes(b, indexes) {
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
		return []byte(style(b))
	}
}

func (m *model) prettyPrint(node *Node, isSelected, isRef bool) string {
	var s string
	if node.Chunk != "" {
		s = node.Chunk
	} else {
		s = node.Value
	}

	if len(s) == 0 {
		if isSelected {
			return theme.CurrentTheme.Cursor(" ")
		} else {
			return s
		}
	}

	var style theme.Color

	if isSelected {
		style = theme.CurrentTheme.Cursor
	} else {
		style = theme.Value(node.Kind)
	}

	if isRef {
		style = theme.CurrentTheme.Ref
	}

	if indexes, ok := m.search.values[node]; ok {
		var out strings.Builder
		for i, p := range splitByIndexes(s, indexes) {
			if i%2 == 0 {
				out.WriteString(style(p.b))
			} else if p.index == m.search.cursor {
				out.WriteString(theme.CurrentTheme.Cursor(p.b))
			} else {
				out.WriteString(theme.CurrentTheme.Search(p.b))
			}
		}
		return out.String()
	} else {
		return style(s)
	}
}

func (m *model) viewWidth() int {
	width := m.termWidth
	if m.showLineNumbers {
		width -= len(strconv.Itoa(m.totalLines))
		width -= 2 // For margin between line numbers and JSON.
	}
	return width
}

func (m *model) viewHeight() int {
	if m.gotoSymbolInput.Focused() {
		return m.termHeight - 2
	}
	if m.commandInput.Focused() {
		return m.termHeight - 2
	}
	if m.searchInput.Focused() || m.searchInput.Value() != "" {
		return m.termHeight - 2
	}
	if m.yank {
		return m.termHeight - 2
	}
	if m.showShowSelector {
		return m.termHeight - 2
	}
	return m.termHeight - 1
}

func (m *model) cursorPointsTo() (*Node, bool) {
	n := m.at(m.cursor)
	return n, n != nil
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
	if n == nil {
		return
	}
	m.showCursor = true
	if m.nodeInsideView(n) {
		m.selectNodeInView(n)
		m.scrollIntoView()
	} else {
		m.cursor = 0
		m.head = n
		{
			parent := n.Parent
			for parent != nil {
				parent.Expand()
				parent = parent.Parent
			}
		}
		m.centerLine(n)
		m.scrollIntoView()
	}
}

func (m *model) cursorPath() string {
	at, ok := m.cursorPointsTo()
	if !ok {
		return ""
	}
	path := ""
	for at != nil {
		if at.Prev != nil {
			if at.Chunk != "" && at.Value == "" {
				at = at.Parent
			}
			if at.Key != "" {
				quoted := at.Key
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
		at = at.Parent
	}
	return path
}

func (m *model) cursorValue() string {
	at, ok := m.cursorPointsTo()
	if !ok {
		return ""
	}
	parent := at.Parent
	if parent != nil {
		// wrapped string part
		if at.Chunk != "" && at.Value == "" {
			at = parent
		}
		if len(at.Value) >= 1 && at.Value[0] == '}' || at.Value[0] == ']' {
			at = parent
		}
	}

	if at.Kind == String {
		str, err := strconv.Unquote(at.Value)
		if err == nil {
			return str
		}
		return at.Value
	}

	var out strings.Builder
	out.WriteString(at.Value)
	out.WriteString("\n")
	if at.HasChildren() {
		it := at.Next
		if at.IsCollapsed() {
			it = at.Collapsed
		}
		for it != nil {
			out.WriteString(strings.Repeat("  ", int(it.Depth-at.Depth)))
			if it.Key != "" {
				out.WriteString(it.Key)
				out.WriteString(": ")
			}
			if it.Value != "" {
				out.WriteString(it.Value)
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
	at, ok := m.cursorPointsTo()
	if !ok {
		return ""
	}
	if at.IsWrap() {
		at = at.Parent
	}
	if at.Key != "" {
		var v string
		_ = json.Unmarshal([]byte(at.Key), &v)
		return v
	}
	return strconv.Itoa(at.Index)
}

func (m *model) editableNode() (*Node, bool) {
	node, ok := m.cursorPointsTo()
	if !ok || node == nil {
		return nil, false
	}
	if node.IsWrap() && node.Parent != nil {
		node = node.Parent
	}
	return node, true
}

func (m *model) decodeJSONString() bool {
	node, ok := m.editableNode()
	if !ok || node.Kind != String {
		return false
	}
	raw, err := utils.Unquote(node.Value)
	if err != nil {
		return false
	}
	if !json.Valid([]byte(raw)) {
		return false
	}
	parsed, err := Parse([]byte(raw))
	if err != nil {
		return false
	}
	ReplaceNode(node, parsed)
	Wrap(node, m.viewWidth())
	m.redoSearch()
	m.selectNode(node)
	return true
}

func (m *model) encodeJSONValue() bool {
	node, ok := m.editableNode()
	if !ok || node.Kind == String {
		return false
	}
	serialized := SerializeNode(node)
	if serialized == "" {
		return false
	}
	comma := ClearChildren(node)
	node.Kind = String
	node.Value = strconv.Quote(serialized)
	node.Size = 0
	node.Collapsed = nil
	node.End = nil
	node.Comma = comma
	Wrap(node, m.viewWidth())
	m.redoSearch()
	m.selectNode(node)
	return true
}

func (m *model) findByPath(path []any) *Node {
	n := m.currentTopNode()
	return n.FindByPath(path)
}

func (m *model) currentTopNode() *Node {
	at, ok := m.cursorPointsTo()
	if !ok {
		return nil
	}
	for at.Parent != nil {
		at = at.Parent
	}
	return at
}

func (m *model) createKeysIndex() {
	at, ok := m.cursorPointsTo()
	if !ok {
		return
	}
	root := at.Root()
	if root == nil {
		return
	}
	paths := make([]string, 0, 100_000)
	nodes := make([]*Node, 0, 100_000)

	root.Paths(&paths, &nodes)

	m.keysIndex = paths
	m.keysIndexNodes = nodes
	m.fuzzyMatch = nil
}

func (m *model) dig(v string) *Node {
	p, ok := jsonpath.Split(v)
	if !ok {
		return nil
	}
	at := m.findByPath(p)
	if at != nil {
		return at
	}

	lastPart := p[len(p)-1]
	searchTerm, ok := lastPart.(string)
	if !ok {
		return nil
	}
	p = p[:len(p)-1]

	at = m.findByPath(p)
	if at == nil {
		return nil
	}

	keys, nodes := at.Children()

	found := fuzzy.Find([]rune(searchTerm), keys)
	if found == nil {
		return nil
	}

	return nodes[found.Index]
}

func (m *model) print() tea.Cmd {
	m.printOnExit = true
	return tea.Quit
}

func (m *model) open() tea.Cmd {
	if engine.FilePath == "" {
		return nil
	}
	command := append(
		strings.Split(lookup([]string{"FX_EDITOR", "EDITOR"}, "vim"), " "),
		engine.FilePath,
	)
	if command[0] == "vi" || command[0] == "vim" {
		at, ok := m.cursorPointsTo()
		if ok {
			tail := command[1:]
			command = append([]string{command[0]}, fmt.Sprintf("+%d", at.LineNumber))
			command = append(command, tail...)
		}
	}
	execCmd := exec.Command(command[0], command[1:]...)
	return tea.ExecProcess(execCmd, func(err error) tea.Msg {
		return nil
	})
}

// deleteAtCursor deletes the current key/value (node) from the view structure.
func (m *model) deleteAtCursor() {
	at, ok := m.cursorPointsTo()
	if !ok || at == nil {
		return
	}
	if next, ok := DeleteNode(at); ok {
		m.selectNode(next)
		m.recordHistory()
	}
}
