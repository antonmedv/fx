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
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err = p.Run()
	if err != nil {
		panic(err)
	}
}

type model struct {
	termWidth, termHeight int
	head                  *node
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
		case tea.MouseWheelDown:
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
		m.cursor--
		if m.cursor < 0 {
			m.cursor = 0
			if m.head.prev != nil {
				m.head = m.head.prev
			}
		}
		return m, nil

	case key.Matches(msg, keyMap.Down):
		m.cursor++
		if m.cursor >= m.viewHeight() {
			m.cursor = m.viewHeight() - 1
			if m.head.next != nil && !m.isCursorAtEnd() {
				m.head = m.head.next
			}
		}
		return m, nil
	}
	return m, nil
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

	if len(screen) > 0 {
		screen = screen[:len(screen)-1]
	}
	return string(screen)
}

func (m *model) viewHeight() int {
	return m.termHeight
}

func (m *model) cursorPointsTo() *node {
	head := m.head
	for i := 0; i < m.cursor; i++ {
		if head == nil {
			return nil
		}
		head = head.next
	}
	return head
}

func (m *model) isCursorAtEnd() bool {
	n := m.cursorPointsTo()
	return n == nil || n.next == nil
}
