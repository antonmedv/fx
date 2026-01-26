package main

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handlePreviewKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(tea.KeyMsg); ok {
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
		}
	}
	m.preview, cmd = m.preview.Update(msg)
	return m, cmd
}
