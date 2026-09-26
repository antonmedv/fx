package main

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/jsonx"
)

// docList links documents the same way nodeMsg does.
type docList struct {
	head, bottom *jsonx.Node
}

func (l *docList) add(t *testing.T, json string) *jsonx.Node {
	node, err := jsonx.Parse([]byte(json))
	require.NoError(t, err)
	if l.head == nil {
		l.head = node
	} else {
		l.bottom.Adjacent(node)
	}
	l.bottom = node
	return node
}

func parseAll(t *testing.T, p *nodesParser) []string {
	var got []string
	for {
		node, err := p.Parse()
		if err == io.EOF {
			return got
		}
		require.NoError(t, err)
		got = append(got, node.Value)
	}
}

func TestNodesParser_Replay(t *testing.T) {
	var l docList
	l.add(t, `{"a": 1}`)
	l.add(t, `[1, 2]`)
	l.add(t, `"s"`)
	l.add(t, `3`)

	p := newNodesParser(l.head, l.bottom, true)
	require.Equal(t, []string{"{", "[", `"s"`, "3"}, parseAll(t, p))
}

func TestNodesParser_WrappedAndCollapsed(t *testing.T) {
	var l docList
	first := l.add(t, `{"a": {"b": 1}}`)
	first.Collapse() // Collapsed before the next doc is attached.
	l.add(t, `"`+strings.Repeat("long ", 20)+`"`)
	third := l.add(t, `[1, [2]]`)
	l.add(t, `null`)
	third.Collapse() // Collapsed after the next doc is attached.
	jsonx.Wrap(l.head, 10)

	p := newNodesParser(l.head, l.bottom, true)
	got := parseAll(t, p)
	require.Len(t, got, 4)
	require.Equal(t, "{", got[0])
	require.True(t, strings.HasPrefix(got[1], `"long`))
	require.Equal(t, "[", got[2])
	require.Equal(t, "null", got[3])
}

func TestNodesParser_BlocksUntilPublish(t *testing.T) {
	var l docList
	l.add(t, `1`)
	p := newNodesParser(l.head, l.bottom, false)

	results := make(chan string)
	go func() {
		for {
			node, err := p.Parse()
			if err == io.EOF {
				close(results)
				return
			}
			results <- node.Value
		}
	}()

	require.Equal(t, "1", <-results)
	select {
	case v := <-results:
		t.Fatalf("expected Parse to block, got %q", v)
	case <-time.After(50 * time.Millisecond):
	}

	p.publish(l.add(t, `2`))
	require.Equal(t, "2", <-results)

	p.setEOF()
	_, ok := <-results
	require.False(t, ok, "expected EOF after setEOF")
}

func TestNodesParser_EmptyUntilFirstPublish(t *testing.T) {
	var l docList
	p := newNodesParser(nil, nil, false)

	done := make(chan string)
	go func() {
		node, err := p.Parse()
		require.NoError(t, err)
		done <- node.Value
	}()

	p.publish(l.add(t, `"first"`))
	require.Equal(t, `"first"`, <-done)
}

func TestNodesParser_StopUnblocks(t *testing.T) {
	var l docList
	l.add(t, `1`)
	p := newNodesParser(l.head, l.bottom, false)
	_, err := p.Parse()
	require.NoError(t, err)

	done := make(chan error)
	go func() {
		_, err := p.Parse()
		done <- err
	}()

	p.stop()
	select {
	case err := <-done:
		require.Equal(t, io.EOF, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Parse did not return after stop")
	}
}

// newQueryModel creates a model with the given documents fully loaded.
func newQueryModel(t *testing.T, docs ...string) *model {
	m := &model{
		termWidth:  80,
		termHeight: 40,
		eof:        true,
		showCursor: true,
		queryInput: textinput.New(),
		search:     newSearch(),
	}
	for _, doc := range docs {
		node, err := jsonx.Parse([]byte(doc))
		require.NoError(t, err)
		m.appendNode(node)
	}
	if m.bottom != nil {
		m.totalLines = m.bottom.Bottom().LineNumber
	}
	return m
}

// drain runs cmd and feeds resulting messages back into Update until
// there are no more commands.
func drain(m *model, cmd tea.Cmd) {
	for cmd != nil {
		msg := cmd()
		_, cmd = m.Update(msg)
	}
}

func typeKeys(m *model, s string) {
	for _, r := range s {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
}

func enter(m *model) {
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	drain(m, cmd)
}

// lines returns the displayed list, one entry per node.
func lines(m *model) []string {
	var out []string
	for it := m.top; it != nil; it = it.Next {
		out = append(out, it.Key+it.Value)
	}
	return out
}

func TestQuery_ApplyOnEnter(t *testing.T) {
	m := newQueryModel(t, `{"a": [1, 2]}`)
	original := m.top

	typeKeys(m, ".")
	require.True(t, m.queryInput.Focused())
	require.Equal(t, ".", m.queryInput.Value())

	typeKeys(m, "a")
	enter(m)

	require.False(t, m.queryInput.Focused())
	require.Equal(t, ".a", m.queryInput.Value())
	require.Equal(t, []string{"[", "1", "2", "]"}, lines(m))
	require.NotNil(t, m.original)
	require.Same(t, original, m.original.top)
}

func TestQuery_DotDoesNotStartEngine(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	original := m.top

	typeKeys(m, ".")
	enter(m)

	require.Nil(t, m.query)
	require.Nil(t, m.original)
	require.Same(t, original, m.top)
}

func TestQuery_StringResult(t *testing.T) {
	m := newQueryModel(t, `{"s": "hi"}`)
	typeKeys(m, ".s")
	enter(m)
	require.Equal(t, []string{`"hi"`}, lines(m))
	require.Equal(t, jsonx.String, m.top.Kind)
}

func TestQuery_LineNumbersAcrossDocuments(t *testing.T) {
	m := newQueryModel(t, `1`, `2`, `{"x": 3}`)
	typeKeys(m, ".")
	m.queryInput.SetValue("x => [x]")
	enter(m)

	var nums []int
	for it := m.top; it != nil; it = it.Next {
		nums = append(nums, it.LineNumber)
	}
	require.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}, nums)
	require.Equal(t, 11, m.totalLines)
}

func TestQuery_ReapplyIgnoresStaleRun(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": 2}`)

	cmd1 := m.doQuery(".a")
	run1 := m.query
	cmd2 := m.doQuery(".b")
	require.NotSame(t, run1, m.query)
	require.True(t, run1.stopped)

	drain(m, cmd1)
	drain(m, cmd2)

	require.Equal(t, []string{"2"}, lines(m))
	require.True(t, run1.done)
	require.True(t, m.query.done)
}

func TestQuery_ReapplyUsesOriginal(t *testing.T) {
	m := newQueryModel(t, `{"a": {"b": 1}}`)
	drain(m, m.doQuery(".a"))
	drain(m, m.doQuery(".a.b"))
	require.Equal(t, []string{"1"}, lines(m))
}

func TestQuery_ExitIsError(t *testing.T) {
	m := newQueryModel(t, `1`)
	drain(m, m.doQuery("x => exit(2)"))
	require.Equal(t, []string{"exit(2) is not allowed in interactive mode"}, lines(m))
	require.Equal(t, jsonx.Err, m.top.Kind)
}

func TestQuery_PrintlnAndError(t *testing.T) {
	m := newQueryModel(t, `1`)
	drain(m, m.doQuery("x => (console.log('a\\nb'), x.foo.bar)"))
	got := lines(m)
	require.Equal(t, []string{"a", "b"}, got[:2])
	require.Greater(t, len(got), 2, "expected error lines")
	require.True(t, m.query.gotErr)
}
