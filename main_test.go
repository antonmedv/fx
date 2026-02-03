package main

import (
	"bytes"
	"io"
	"os"
	"strconv"
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

func newTestModelFromJSON(t *testing.T, data string) *model {
	head, err := jsonx.Parse([]byte(data))
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
	return m
}

func findChildByKey(n *jsonx.Node, key string) *jsonx.Node {
	if n == nil {
		return nil
	}
	keys, nodes := n.Children()
	for i, k := range keys {
		if k == key {
			return nodes[i]
		}
	}
	return nil
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

func TestDecodeEncodeJSONNode(t *testing.T) {
	m := newTestModelFromJSON(t, `{"args":"{\"requestId\":\"id\",\"content\":\"123\"}","cost":16}`)
	args := findChildByKey(m.top, "args")
	require.NotNil(t, args)
	m.selectNode(args)
	require.True(t, m.decodeJSONString())
	require.Equal(t, jsonx.Object, args.Kind)
	require.True(t, args.HasChildren())
	require.True(t, m.encodeJSONValue())
	require.Equal(t, jsonx.String, args.Kind)
	expected := `{"requestId":"id","content":"123"}`
	require.Equal(t, strconv.Quote(expected), args.Value)
}
