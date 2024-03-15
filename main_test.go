package main

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

func prepare(t *testing.T) *teatest.TestModel {
	file, err := os.Open("testdata/example.json")
	require.NoError(t, err)

	json, err := io.ReadAll(file)
	require.NoError(t, err)

	head, err := parse(json)
	require.NoError(t, err)

	m := &model{
		top:         head,
		head:        head,
		wrap:        true,
		showCursor:  true,
		digInput:    textinput.New(),
		searchInput: textinput.New(),
		search:      newSearch(),
	}
	tm := teatest.NewTestModel(
		t, m,
		teatest.WithInitialTermSize(80, 40),
	)
	return tm
}

func read(t *testing.T, tm *teatest.TestModel) []byte {
	var out []byte
	teatest.WaitFor(t,
		tm.Output(),
		func(b []byte) bool {
			out = b
			return bytes.Contains(b, []byte("{"))
		},
		teatest.WithCheckInterval(time.Millisecond*100),
		teatest.WithDuration(time.Second),
	)
	return out
}

func TestOutput(t *testing.T) {
	tm := prepare(t)

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestNavigation(t *testing.T) {
	tm := prepare(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestDig(t *testing.T) {
	tm := prepare(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(".")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("year")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCollapseRecursive(t *testing.T) {
	tm := prepare(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyShiftLeft})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCollapseRecursiveWithSizes(t *testing.T) {
	showSizes = true
	defer func() { showSizes = true }()

	tm := prepare(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyShiftLeft})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
