package main

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"
)

// hasChunks reports whether a string of the list is wrapped.
func hasChunks(m *model) bool {
	for it := m.top; it != nil; it = it.Next {
		if it.IsWrap() {
			return true
		}
	}
	return false
}

func TestWrap_OffStaysOff(t *testing.T) {
	m := loadModel(t, `{"s": "`+strings.Repeat("word ", 40)+`", "n": 1}`, 40, 10)
	keys(m, "z")
	require.True(t, m.wrap)
	require.True(t, hasChunks(m))

	keys(m, "z")
	require.False(t, m.wrap)
	require.False(t, hasChunks(m))

	m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	require.False(t, hasChunks(m), "resize must not wrap with wrap off")
	keys(m, "s", "l")
	require.True(t, m.showLineNumbers)
	require.False(t, hasChunks(m), "line numbers must not wrap with wrap off")

	keys(m, "z")
	require.True(t, m.wrap, "one z turns wrap back on")
	require.True(t, hasChunks(m))
}

func TestWrap_ResizeRewraps(t *testing.T) {
	m := loadModel(t, `{"s": "`+strings.Repeat("word ", 40)+`"}`, 40, 10)
	keys(m, "z")
	chunks := func() int {
		n := 0
		for it := m.top; it != nil; it = it.Next {
			if it.IsWrap() {
				n++
			}
		}
		return n
	}
	at40 := chunks()
	m.Update(tea.WindowSizeMsg{Width: 25, Height: 10})
	require.Greater(t, chunks(), at40, "narrower re-wraps into more lines")
	requireListSound(t, m)
}
