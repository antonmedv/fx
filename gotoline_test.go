package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
)

func init() {
	lipgloss.SetColorProfile(termenv.ANSI)
}

func TestGotoLine(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestGotoLineCollapsed(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("5")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestGotoLineInputInvalid(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("E")})

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("invalid")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestGotoLineInputGreaterThanTotalLines(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("500")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestGotoLineInputLessThanOne(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("-2")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestGotoLineKeepsHistory(t *testing.T) {
	tm := prepare(t, options{showLineNumbers: true})

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("14")})
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})

	teatest.RequireEqualOutput(t, read(t, tm))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
