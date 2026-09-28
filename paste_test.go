package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// Bracketed paste arrives as one message and goes to the focused input.
func TestPaste(t *testing.T) {
	m := newQueryModel(t, `{"items": [1]}`)

	typeKeys(m, ".")
	_, cmd := m.Update(tea.PasteMsg{Content: "items"})
	require.Equal(t, ".items", m.queryInput.Value())
	require.NotNil(t, cmd, "the query schedules its preview")
	m.Update(press("esc"))

	m.Update(press("/"))
	m.Update(tea.PasteMsg{Content: "it"})
	require.Equal(t, "it", m.searchInput.Value())
	m.Update(press("esc"))

	m.Update(press(":"))
	m.Update(tea.PasteMsg{Content: "5"})
	require.Equal(t, "5", m.commandInput.Value())
	m.Update(press("esc"))

	m.showPreview = true
	m.previewSearchInput = newInput()
	m.previewSearchInput.Focus()
	m.Update(tea.PasteMsg{Content: "x"})
	require.Equal(t, "x", m.previewSearchInput.Value())
	m.showPreview = false

	// Nothing focused: a paste is ignored.
	m.Update(tea.PasteMsg{Content: "zzz"})
	require.Equal(t, "", m.queryInput.Value())
}
