package main

import (
	"bytes"
	"io"
	"strings"
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
		require.Less(t, n, 1_000_000, "the list loops")
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

// findKey returns the first node of the list with key.
func findKey(m *model, key string) *Node {
	for it := m.top; it != nil; it = it.Next {
		if it.Key == key {
			return it
		}
	}
	return nil
}

// visit selects the node with key and records it, as a jump to it does.
func visit(t *testing.T, m *model, key string) {
	n := findKey(m, key)
	require.NotNil(t, n, key)
	m.selectNode(n)
	m.recordHistory()
}

// requireNotDeleted checks the cursor and every line on screen are in the
// document.
func requireNotDeleted(t *testing.T, m *model) {
	t.Helper()
	at, ok := m.cursorPointsTo()
	require.True(t, ok, "cursor on a line")
	require.True(t, at.InDocument(), "cursor on deleted %q", at.Key+at.Value)
	it := m.head
	for range m.viewHeight() {
		if it == nil {
			break
		}
		require.True(t, it.IsLinked(), "deleted %q on screen", it.Key+it.Value)
		it = it.Next
	}
}

const nestedInput = `{"a":{"b":{"k0":0,"k1":1,"k2":2,"k3":3,"k4":4,"k5":5,"k6":6,"k7":7,"k8":8,"c":9,"d":10},"e":3},"f":4}`

func TestHistory_SkipsDeletedSubtree(t *testing.T) {
	for _, height := range []int{7, 20} {
		m := loadModel(t, nestedInput, 80, height)
		visit(t, m, `"f"`)
		visit(t, m, `"c"`)
		visit(t, m, `"b"`)
		keys(m, "d", "d") // Deletes .a.b
		requireListSound(t, m)

		keys(m, "[")
		requireNotDeleted(t, m)
		at, _ := m.cursorPointsTo()
		require.Equal(t, `"f"`, at.Key, "back past the deleted .a.b and .a.b.c, height %d", height)

		keys(m, "[", "]")
		requireNotDeleted(t, m)
	}
}

func TestHistory_ForwardSkipsDeleted(t *testing.T) {
	m := loadModel(t, nestedInput, 80, 7)
	visit(t, m, `"f"`)
	visit(t, m, `"c"`)
	visit(t, m, `"e"`)
	keys(m, "[", "[") // Back to "f", with "c" and "e" ahead.
	require.Equal(t, 0, m.locationIndex)
	require.Len(t, m.locationHistory, 3)

	// Delete .a.b without recording history, which would drop what's ahead.
	_, ok := DeleteNode(findKey(m, `"b"`))
	require.True(t, ok)

	keys(m, "]")
	requireNotDeleted(t, m)
	at, _ := m.cursorPointsTo()
	require.Equal(t, `"e"`, at.Key, "forward past the deleted .a.b.c")
	require.Equal(t, 2, m.locationIndex)
}

func TestSearch_SkipsDeletedResults(t *testing.T) {
	m := loadModel(t, `{"a":{"x":"needle","y":"needle"},"b":"needle","c":{"z":"needle"}}`, 80, 7)
	doSearch(m, "needle")
	require.Len(t, m.search.results, 4)

	m.selectNode(findKey(m, `"a"`))
	keys(m, "d", "d") // Deletes the first two results.
	onMatch := func() {
		t.Helper()
		requireNotDeleted(t, m)
		at, _ := m.cursorPointsTo()
		require.Equal(t, `"needle"`, at.Value, "n lands on a match, not on %q", at.Key+at.Value)
	}
	for range 6 {
		keys(m, "n")
		onMatch()
	}
	for range 6 {
		keys(m, "N")
		onMatch()
	}

	m.selectNode(findKey(m, `"b"`))
	keys(m, "d", "d")
	m.selectNode(findKey(m, `"c"`))
	keys(m, "d", "d") // No result left in the document.
	require.NotPanics(t, func() { keys(m, "n", "N") })
	requireListSound(t, m)
}

// Deleting a wrapped string whose continuation line is the first on screen
// must not leave its lines displayed: deleting such a ghost line deleted the
// string again, and the array size dropped below its length.
func TestDelete_WrappedStringUnderHead(t *testing.T) {
	m := loadModel(t, `["`+strings.Repeat("a", 400)+`", 1, 2, 3]`, 40, 6)
	keys(m, "z")
	for range 20 {
		if at, _ := m.cursorPointsTo(); at.Value == "1" {
			break
		}
		keys(m, "j")
	}
	require.True(t, m.head.IsWrap(), "the first screen line is a continuation of the string")
	keys(m, "k", "d", "d")
	requireListSound(t, m)
	requireNotDeleted(t, m)
	keys(m, "k", "k", "d", "d")
	require.Equal(t, 3, m.top.Size)
	requireListSound(t, m)
}
