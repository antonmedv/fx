package main

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"
)

// last returns the last line of the rendered screen, where the inputs are.
func last(m *model) string {
	s := view(m)
	return s[strings.LastIndex(s, "\n")+1:]
}

// The inputs draw their own cursor; the terminal cursor stays hidden.
func TestCursor_DrawnByInputs(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	require.Nil(t, m.View().Cursor)

	typeKeys(m, ".a")
	require.Nil(t, m.View().Cursor)
	require.True(t, strings.HasSuffix(ansi.Strip(last(m)), "> .a "), "cursor block after the query")
	pressKey(m, press("left"))
	require.Contains(t, last(m), reverseStyle("a"), "cursor over the character")
	m.Update(press("esc"))

	m.Update(press("/"))
	typeKeys(m, "ab")
	require.Nil(t, m.View().Cursor)
	require.Contains(t, last(m), m.searchInput.Prompt+"ab"+reverseStyle(" "), "cursor block after the search")
	m.Update(press("esc"))

	m.Update(press(":"))
	typeKeys(m, "5")
	require.Nil(t, m.View().Cursor)
	require.Contains(t, last(m), m.commandInput.Prompt+"5"+reverseStyle(" "), "cursor block after the command")
}
