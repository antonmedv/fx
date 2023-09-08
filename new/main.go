package main

import (
	"fmt"
	"io"
	"os"
	"runtime/pprof"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	head, err := parse(data)
	if err != nil {
		fmt.Print(err.Error())
		os.Exit(1)
		return
	}

	m := &model{
		head: head,
		top:  head,
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
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

	case tea.MouseMsg:
		switch msg.Type {
		case tea.MouseWheelUp:
			m.up()

		case tea.MouseWheelDown:
			m.down()
		}

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
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
		m.cursor = 0
		m.head = m.top

	case key.Matches(msg, keyMap.GotoBottom):
		m.cursor = 0
		m.head = m.findBottom()
		m.scrollIntoView()
		m.cursor = m.visibleLines() - 1

	case key.Matches(msg, keyMap.Collapse):
		m.cursorPointsTo().collapse()
		m.scrollIntoView()

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
		if m.head.next != nil && !n.atEnd() {
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

func (m *model) scrollIntoView() string {
	visibleLines := m.visibleLines()
	for visibleLines < m.viewHeight() && m.head.prev != nil {
		visibleLines++
		m.cursor++
		m.head = m.head.prev
	}
	return fmt.Sprintf("visible lines: %d", visibleLines)
}

func (m *model) View() string {
	var screen []byte
	head := m.head

	// Prerender syntax
	colon := currentTheme.Syntax([]byte{':', ' '})
	comma := currentTheme.Syntax([]byte{','})

	for i := 0; i < m.viewHeight(); i++ {
		if head == nil {
			break
		}
		for ident := 0; ident < int(head.depth); ident++ {
			screen = append(screen, ' ', ' ')
		}
		if head.key != nil {
			keyColor := currentTheme.Key
			if m.cursor == i {
				keyColor = currentTheme.Cursor
			}
			screen = append(screen, keyColor(head.key)...)
			screen = append(screen, colon...)
			screen = append(screen, colorForValue(head.value)(head.value)...)
		} else {
			colorize := colorForValue(head.value)
			if m.cursor == i {
				colorize = currentTheme.Cursor
			}
			screen = append(screen, colorize(head.value)...)
		}
		if head.comma {
			screen = append(screen, comma...)
		}

		screen = append(screen, '\n')
		head = head.next
	}

	n := m.cursorPointsTo()
	if n == nil {
		screen = append(screen, '-')
	} else {
		screen = append(screen, currentTheme.StatusBar(n.value)...)
	}
	screen = append(screen, m.scrollIntoView()...)

	return string(screen)
}

func (m *model) viewHeight() int {
	return m.termHeight - 1
}

func (m *model) cursorPointsTo() *node {
	head := m.head
	for i := 0; i < m.cursor; i++ {
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
