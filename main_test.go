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

	"github.com/antonmedv/fx/internal/jsonx"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

type options struct {
	showSizes       bool
	showLineNumbers bool
}

func prepare(t *testing.T, opts ...options) *teatest.TestModel {
	file, err := os.Open("testdata/example.json")
	require.NoError(t, err)

	json, err := io.ReadAll(file)
	require.NoError(t, err)

	head, err := jsonx.Parse(json)
	require.NoError(t, err)

	m := &model{
		top:          head,
		head:         head,
		bottom:       head,
		totalLines:   head.Bottom().LineNumber,
		eof:          true,
		wrap:         true,
		showCursor:   true,
		searchInput:  textinput.New(),
		search:       newSearch(),
		commandInput: textinput.New(),
	}

	if len(opts) > 0 {
		m.showSizes = opts[0].showSizes
		m.showLineNumbers = opts[0].showLineNumbers
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

func TestCollapseRecursive(t *testing.T) {
	tm := prepare(t)

	tm.Send(tea.KeyMsg{Type: tea.KeyShiftLeft})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestCollapseRecursiveWithSizes(t *testing.T) {
	tm := prepare(t, options{showSizes: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyShiftLeft})
	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestUndoRedoInteraction(t *testing.T) {
	tm := prepare(t)
	targetKey := []byte(`"title"`)

	// Leave root, then delete (first key)
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})

	// Verify the node is gone from the output
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return !bytes.Contains(b, targetKey)
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})

	// Verify the node reappeared
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, targetKey)
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlR})

	// Verify it is gone again
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return !bytes.Contains(b, targetKey)
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t)
}
