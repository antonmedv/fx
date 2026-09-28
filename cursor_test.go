package main

import (
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
	"github.com/stretchr/testify/require"
)

// lastLine is the row of the last line of the rendered screen, where every
// text input is drawn.
func lastLine(m *model) int {
	return strings.Count(view(m), "\n")
}

func TestCursor_HiddenOutsideInputs(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	require.Nil(t, m.View().Cursor)

	m.Update(press("?"))
	require.True(t, m.showHelp)
	require.Nil(t, m.View().Cursor)
}

func TestCursor_Query(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	m.queryInput.Prompt = "" // as in main
	typeKeys(m, ".a")
	c := m.View().Cursor
	require.NotNil(t, c)
	require.Equal(t, 2, c.X)
	require.Equal(t, lastLine(m), c.Y)

	// Wider than the input: the window scrolls to keep the cursor visible.
	typeKeys(m, strings.Repeat("b", 100))
	width := m.termWidth - 1
	require.Equal(t, width, m.queryInput.Width())
	require.Equal(t, width, m.View().Cursor.X)

	pressKey(m, press("left"))
	pressKey(m, press("left"))
	pressKey(m, press("left"))
	require.Equal(t, width-3, m.View().Cursor.X)

	pressKey(m, press("home"))
	require.Equal(t, 0, m.View().Cursor.X)
}

func TestCursor_SearchAndCommand(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	m.Update(press("/"))
	typeKeys(m, "ab")
	c := m.View().Cursor
	require.NotNil(t, c)
	require.Equal(t, runewidth.StringWidth(m.searchInput.Prompt)+2, c.X)
	require.Equal(t, lastLine(m), c.Y)
	m.Update(press("esc"))

	m.Update(press(":"))
	typeKeys(m, "5")
	c = m.View().Cursor
	require.NotNil(t, c)
	require.Equal(t, runewidth.StringWidth(m.commandInput.Prompt)+1, c.X)
	require.Equal(t, lastLine(m), c.Y)

	// A question in the command line hides the cursor.
	m.confirm = &confirmation{prompt: "sure?"}
	require.Nil(t, m.View().Cursor)
}
