package main

import (
	"bytes"
	"io"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/require"

	. "github.com/antonmedv/fx/internal/jsonx"
)

// No key press may crash fx, nor leave the list in a state where a later
// one loops forever.

// loadModel reads input the way the loader does: text that is not JSON is
// recovered as text lines, as with curl -i output.
func loadModel(t *testing.T, input string, width, height int) *model {
	m := newQueryModel(t)
	m.termWidth, m.termHeight = width, height
	p := NewJsonParser(bytes.NewReader([]byte(input)), false)
	for {
		node, err := p.Parse()
		if err == io.EOF {
			break
		}
		if err != nil {
			node = p.Recover()
			require.NotNil(t, node)
		}
		m.appendNode(node)
	}
	m.totalLines = m.bottom.Bottom().LineNumber
	return m
}

// within fails the test if f does not return in time, instead of hanging.
func within(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("did not return: the node list loops")
	}
}

// requireListSound checks the displayed list ends, links back, and holds head.
func requireListSound(t *testing.T, m *model) {
	t.Helper()
	seenHead := false
	n := 0
	for it := m.top; it != nil; it = it.Next {
		n++
		require.Less(t, n, 10_000, "the list loops")
		if it.Next != nil {
			require.Same(t, it, it.Next.Prev, "Next.Prev of %q", it.Key+it.Value)
		}
		if it == m.head {
			seenHead = true
		}
	}
	require.True(t, seenHead, "head is not a line of the list")
}

func keys(m *model, names ...string) {
	for _, k := range names {
		m.Update(press(k))
		m.View()
	}
}

func TestDelete_AfterCollapseAllHidHead(t *testing.T) {
	input := `{"x":{"a":1,"b":2}}` + "\n" + `{"y":{"c":1}}` + "\n" + `{"d":4}` + "\n" + `{"e":5}`
	for _, tail := range [][]string{nil, {"z"}, {"z", "z"}, {"G"}, {"g", "y", "y"}, {"E"}} {
		m := loadModel(t, input, 80, 7)
		within(t, func() {
			keys(m, "ctrl+e", "ctrl+e", "j", "j", "j", "j", "E", "k", "k", "k", "k", "d", "d")
			requireListSound(t, m)
			m.Update(tea.WindowSizeMsg{Width: 40, Height: 7})
			m.View()
			keys(m, tail...)
		})
		requireListSound(t, m)
	}
}

func TestDelete_HiddenNodeIsRefused(t *testing.T) {
	root, err := Parse([]byte(`{"o":{"a":1,"b":2},"z":3}`))
	require.NoError(t, err)
	o := root.Next
	a := o.Next
	o.Collapse()
	_, ok := DeleteNode(a)
	require.False(t, ok, "a is hidden in the collapsed o")
	n := 0
	for it := root; it != nil; it = it.Next {
		n++
		require.Less(t, n, 100, "the list loops")
	}
}

func TestDelete_HeadRow(t *testing.T) {
	m := loadModel(t, `[0,1,2,3,4,5,6,7,8,9]`, 80, 6)
	within(t, func() {
		keys(m, "space", "d", "d", "j", "k", "d", "d", "k", "k", "d", "d")
	})
	requireListSound(t, m)
	require.NotContains(t, string(m.viewJSON()), "2,", "the deleted 2 came back")
}

func TestCursorValue_BlankTextLine(t *testing.T) {
	input := "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"a\": 1}"
	for _, seq := range [][]string{{"j", "j", "p"}, {"j", "j", "y", "y"}, {"j", "j", "y", "b"}, {"j", "j", "P"}} {
		m := loadModel(t, input, 80, 24)
		m.sshSession = true // keeps the test off this machine's clipboard
		require.NotPanics(t, func() { keys(m, seq...) }, "%v", seq)
	}
}
