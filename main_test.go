package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func init() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

func TestOutput(t *testing.T) {
	file, err := os.Open("testdata/example.json")
	require.NoError(t, err)

	json, err := io.ReadAll(file)
	require.NoError(t, err)

	head, err := parse(json)
	require.NoError(t, err)

	m := &model{
		top:        head,
		head:       head,
		wrap:       true,
		showCursor: true,
		search:     newSearch(),
	}
	tm := teatest.NewTestModel(
		t, m,
		teatest.WithInitialTermSize(300, 100),
	)

	teatest.WaitFor(t,
		tm.Output(),
		func(b []byte) bool {
			return bytes.Contains(b, []byte("Fox and Dog"))
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
