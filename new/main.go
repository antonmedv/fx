package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"io"
	"os"
	"runtime/pprof"
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
	windowWidth, windowHeight int
	head                      *node
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height

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
		m.head = m.head.prev
		return m, nil

	case key.Matches(msg, keyMap.Down):
		m.head = m.head.next
		return m, nil
	}
	return m, nil
}

func (m *model) View() string {
	var screen []byte
	head := m.head
	for i := 0; i < m.windowHeight; i++ {
		if head == nil {
			break
		}
		for ident := 0; ident < int(head.depth); ident++ {
			screen = append(screen, ' ', ' ')
		}
		if head.key != nil {
			screen = append(screen, head.key...)
			screen = append(screen, ':', ' ')
		}
		screen = append(screen, head.value...)
		if head.comma {
			screen = append(screen, ',')
		}
		screen = append(screen, '\n')
		head = head.next
	}
	if len(screen) > 0 {
		screen = screen[:len(screen)-1]
	}
	return string(screen)
}
