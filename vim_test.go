package main

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"

	. "github.com/antonmedv/fx/internal/jsonx"
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

func finalModel(t *testing.T, tm *teatest.TestModel) *model {
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	return tm.FinalModel(t, teatest.WithFinalTimeout(time.Second)).(*model)
}

func cursorNode(t *testing.T, m *model) *Node {
	n, ok := m.cursorPointsTo()
	require.True(t, ok)
	return n
}

func sendKeys(tm *teatest.TestModel, keys ...tea.KeyMsg) {
	for _, k := range keys {
		tm.Send(k)
	}
}

func command(s string) []tea.KeyMsg {
	return []tea.KeyMsg{
		{Type: tea.KeyRunes, Runes: []rune(":")},
		{Type: tea.KeyRunes, Runes: []rune(s)},
		{Type: tea.KeyEnter},
	}
}

var keyPercent = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("%")}

func requireTagsEnd(t *testing.T, m *model) {
	n := cursorNode(t, m)
	require.Equal(t, `"tags"`, n.Parent.Key)
	require.Equal(t, n.Parent.End, n)
}

func TestMatchBracketRoot(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, keyPercent)
	m := finalModel(t, tm)
	require.Equal(t, m.top.End, cursorNode(t, m))
}

func TestMatchBracketToggle(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, command("4")...)
	sendKeys(tm, keyPercent, keyPercent, keyPercent)
	requireTagsEnd(t, finalModel(t, tm))
}

func TestMatchBracketBackToOpening(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, command("4")...)
	sendKeys(tm, keyPercent, keyPercent)
	m := finalModel(t, tm)
	n := cursorNode(t, m)
	require.Equal(t, `"tags"`, n.Key)
	require.NotNil(t, n.End)
}

func TestMatchBracketFromElement(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, command("6")...)
	sendKeys(tm, keyPercent)
	requireTagsEnd(t, finalModel(t, tm))
}

func TestMatchBracketFromWrappedString(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, command("3")...)
	sendKeys(tm, tea.KeyMsg{Type: tea.KeyDown}, keyPercent)
	m := finalModel(t, tm)
	require.Equal(t, m.top.End, cursorNode(t, m))
}

func TestMatchBracketCollapsed(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, command("4")...)
	sendKeys(tm, tea.KeyMsg{Type: tea.KeyLeft}, keyPercent)
	m := finalModel(t, tm)
	requireTagsEnd(t, m)
	require.Nil(t, cursorNode(t, m).Parent.Collapsed)
}

var (
	keyCtrlE = tea.KeyMsg{Type: tea.KeyCtrlE}
	keyCtrlY = tea.KeyMsg{Type: tea.KeyCtrlY}
	keyDown  = tea.KeyMsg{Type: tea.KeyDown}
)

// smallWindow is a 6-row terminal, which leaves 5 lines for the view.
var smallWindow = tea.WindowSizeMsg{Width: 80, Height: 6}

func TestScrollDownKeepsCursorNode(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, keyDown, keyDown, keyDown, keyCtrlE)

	m := finalModel(t, tm)
	require.Equal(t, m.top.Next, m.head)
	require.Equal(t, 2, m.cursor)
	require.Equal(t, m.top.Next.Next.Next, cursorNode(t, m))
}

func TestScrollDownPushesCursorAtTop(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, keyCtrlE, keyCtrlE)

	m := finalModel(t, tm)
	require.Equal(t, m.top.Next.Next, m.head)
	require.Equal(t, 0, m.cursor)
	require.Equal(t, m.head, cursorNode(t, m))
}

func TestScrollDownStopsAtLastLine(t *testing.T) {
	tm := prepare(t)
	for range 30 {
		sendKeys(tm, keyCtrlE)
	}

	m := finalModel(t, tm)
	require.Equal(t, m.bottom.Bottom(), m.head)
	require.Equal(t, 0, m.cursor)
}

func TestScrollUpKeepsCursorNode(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, keyDown, keyDown, keyCtrlE, keyCtrlE, keyCtrlY)

	m := finalModel(t, tm)
	require.Equal(t, m.top.Next, m.head)
	require.Equal(t, 1, m.cursor)
	require.Equal(t, m.top.Next.Next, cursorNode(t, m))
}

func TestScrollUpAtTopIsNoop(t *testing.T) {
	tm := prepare(t)
	sendKeys(tm, keyDown, keyCtrlY)

	m := finalModel(t, tm)
	require.Equal(t, m.top, m.head)
	require.Equal(t, 1, m.cursor)
}

func TestScrollUpPushesCursorAtBottom(t *testing.T) {
	tm := prepare(t)
	tm.Send(smallWindow)
	sendKeys(tm, keyCtrlE, keyCtrlE, keyDown, keyDown, keyDown, keyDown, keyCtrlY)

	m := finalModel(t, tm)
	// Head moved back one line, the cursor stays on the last view row and now
	// points to the node above the one it was on.
	require.Equal(t, m.top.Next, m.head)
	require.Equal(t, 4, m.cursor)
	require.Equal(t, m.at(4), cursorNode(t, m))
	require.Equal(t, m.top.Next.Next.Next.Next.Next, cursorNode(t, m))
}
