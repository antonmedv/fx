package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/engine"
	"github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
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
		termWidth:    80,
		termHeight:   40,
		eof:          true,
		showCursor:   true,
		queryInput:   textinput.New(),
		searchInput:  textinput.New(),
		commandInput: textinput.New(),
		viewState:    viewState{search: newSearch()},
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

func TestQuery_MultipleArgs(t *testing.T) {
	m := newQueryModel(t, `{"users": [{"name": "a b"}, {"name": "c"}]}`)
	drain(m, m.doQuery(`.users @.name .join(", ") x => x + "!"`))
	require.Equal(t, []string{`"a b, c!"`}, lines(m))
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

func TestQuery_ClearRestoresOriginal(t *testing.T) {
	m := newQueryModel(t, `{"a": {"b": 1}, "c": [1, 2, 3], "d": 4}`)
	m.top.Next.Collapse() // "a"
	m.cursor = 2
	head, top, bottom := m.head, m.top, m.bottom
	history := []location{{head: m.head, node: m.top}}
	m.locationHistory, m.locationIndex = history, 1
	totalLines := m.totalLines

	typeKeys(m, ".")
	typeKeys(m, "c")
	enter(m)
	require.NotNil(t, m.original)
	m.cursor = 1 // Move around in the result view.

	typeKeys(m, ".")
	m.queryInput.SetValue(".")
	enter(m)

	require.Nil(t, m.original)
	require.Nil(t, m.query)
	require.False(t, m.restoring)
	require.Equal(t, "", m.queryInput.Value())
	require.Same(t, head, m.head)
	require.Same(t, top, m.top)
	require.Same(t, bottom, m.bottom)
	require.Equal(t, 2, m.cursor)
	require.Equal(t, totalLines, m.totalLines)
	require.Equal(t, history, m.locationHistory)
	require.Equal(t, 1, m.locationIndex)
	require.True(t, m.top.Next.IsCollapsed())
}

func TestQuery_ClearWaitsForRunningEngine(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	m.eof = false // Streaming: the engine blocks waiting for more documents.
	top := m.top

	cmd := m.doQuery(".a")
	require.Equal(t, 1, m.runningQueries)

	m.doQuery(".")
	require.True(t, m.restoring)
	require.NotNil(t, m.original, "must not restore while the engine still runs")

	drain(m, cmd)

	require.Equal(t, 0, m.runningQueries)
	require.False(t, m.restoring)
	require.Nil(t, m.original)
	require.Same(t, top, m.top)
}

func TestQuery_ApplyWhileRestoring(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": 2}`)
	m.eof = false

	cmd1 := m.doQuery(".a")
	m.doQuery(".")
	require.True(t, m.restoring)

	m.eof = true
	cmd2 := m.doQuery(".b")
	require.False(t, m.restoring)

	drain(m, cmd1)
	drain(m, cmd2)

	require.NotNil(t, m.original)
	require.Equal(t, []string{"2"}, lines(m))
}

func TestQuery_ClearRewrapsOnWidthChange(t *testing.T) {
	m := newQueryModel(t, `{"s": "`+strings.Repeat("word ", 30)+`"}`)
	m.wrap = true
	jsonx.Wrap(m.top, m.viewWidth())
	chunks := func() int {
		n := 0
		for it := m.top; it != nil; it = it.Next {
			if it.Value == "" && it.Chunk != "" {
				n++
			}
		}
		return n
	}
	before := chunks()

	drain(m, m.doQuery(".s"))
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 40})
	drain(m, m.doQuery("."))

	require.Greater(t, chunks(), before)
}

// step runs cmd once and feeds its message into Update.
func step(m *model, cmd tea.Cmd) tea.Cmd {
	_, next := m.Update(cmd())
	return next
}

func parseDoc(t *testing.T, json string) *jsonx.Node {
	node, err := jsonx.Parse([]byte(json))
	require.NoError(t, err)
	return node
}

func TestQuery_Streaming(t *testing.T) {
	m := newQueryModel(t, `{"id": 1}`)
	m.eof = false

	cmd := m.doQuery(".id")
	cmd = step(m, cmd)
	require.Equal(t, []string{"1"}, lines(m))

	doc2 := parseDoc(t, `{"id": 2}`)
	m.Update(nodeMsg{node: doc2})
	cmd = step(m, cmd)
	require.Equal(t, []string{"1", "2"}, lines(m))
	require.Same(t, doc2, m.original.bottom)

	m.Update(nodeMsg{node: parseDoc(t, `{"id": 3}`)})
	m.Update(eofMsg{})
	drain(m, cmd)

	require.Equal(t, []string{"1", "2", "3"}, lines(m))
	require.True(t, m.query.done)
	require.Equal(t, 3, m.totalLines)
}

func TestQuery_StreamingStartsWithNoDocuments(t *testing.T) {
	m := newQueryModel(t)
	m.eof = false

	cmd := m.doQuery(".id")
	m.Update(nodeMsg{node: parseDoc(t, `{"id": 1}`)})
	cmd = step(m, cmd)
	require.Equal(t, []string{"1"}, lines(m))
	require.NotNil(t, m.original.top)

	m.Update(eofMsg{})
	drain(m, cmd)
}

func TestQuery_StreamingAfterWrappedString(t *testing.T) {
	m := newQueryModel(t)
	m.eof = false
	m.wrap = true
	m.Update(nodeMsg{node: parseDoc(t, `"`+strings.Repeat("word ", 30)+`"`)})

	cmd := m.doQuery("x => x.length")
	cmd = step(m, cmd)
	m.Update(nodeMsg{node: parseDoc(t, `"b"`)})
	cmd = step(m, cmd)
	m.Update(eofMsg{})
	drain(m, cmd)

	require.Equal(t, []string{"150", "1"}, lines(m))
}

func TestQuery_RestoreWrapsAndCollapsesStreamedDocuments(t *testing.T) {
	m := newQueryModel(t, `{"a": {"b": 1}}`)
	m.eof = false
	m.wrap = true
	m.collapsed = true

	cmd := m.doQuery(".a")
	cmd = step(m, cmd)
	streamed := parseDoc(t, `{"a": {"b": "`+strings.Repeat("word ", 30)+`"}}`)
	m.Update(nodeMsg{node: streamed})
	cmd = step(m, cmd)
	require.False(t, streamed.Next.IsCollapsed(), "hidden original must not be mutated")
	require.Nil(t, streamed.Next.Next.ChunkEnd, "hidden original must not be wrapped")

	m.Update(eofMsg{})
	drain(m, cmd)
	drain(m, m.doQuery("."))

	require.Nil(t, m.original)
	require.True(t, streamed.Next.IsCollapsed())
	require.NotNil(t, streamed.Next.Collapsed.ChunkEnd, "streamed string must be wrapped")
}

func TestNodesParser_SkipsRecoveredText(t *testing.T) {
	var l docList
	l.add(t, `1`)
	text := &jsonx.Node{Kind: jsonx.Err, Value: "HTTP/1.1 200 OK"}
	l.bottom.Adjacent(text)
	l.bottom = text
	l.add(t, `2`)

	p := newNodesParser(l.head, l.bottom, true)
	require.Equal(t, []string{"1", "2"}, parseAll(t, p))
}

func TestQuery_ErrorStopsAndIsShownBelowResults(t *testing.T) {
	m := newQueryModel(t, `{"n": {"x": 1}}`, `{"n": {"x": 2}}`, `{"id": 3}`, `{"n": {"x": 4}}`)

	typeKeys(m, ".")
	m.queryInput.SetValue("x => x.n.x * 10")
	enter(m)

	got := lines(m)
	require.Equal(t, []string{"10", "20"}, got[:2])
	require.Greater(t, len(got), 2)
	require.NotContains(t, got, "40", "engine must stop at the failing document")
	require.Contains(t, strings.Join(got[2:], "\n"), "TypeError")

	require.False(t, m.isQueryError(m.top))
	errLine := m.top.Next.Next
	require.True(t, m.isQueryError(errLine))
	require.Equal(t, theme.CurrentTheme.Error(errLine.Value), m.prettyPrint(errLine, false, false))

	// The query is kept for fixing.
	typeKeys(m, ".")
	require.True(t, m.queryInput.Focused())
	require.Equal(t, "x => x.n.x * 10", m.queryInput.Value())
}

func TestQuery_PrintlnIsNotStyledAsError(t *testing.T) {
	m := newQueryModel(t, `1`)
	drain(m, m.doQuery("x => (console.log('hello'), x)"))
	require.Equal(t, []string{"hello", "1"}, lines(m))
	require.False(t, m.isQueryError(m.top))
}

func TestQuery_WrappedErrorLineIsStyled(t *testing.T) {
	m := newQueryModel(t, `1`)
	m.wrap = true
	m.termWidth = 20
	drain(m, m.doQuery("x => { throw new Error('"+strings.Repeat("long ", 20)+"') }"))

	wrapped := 0
	for it := m.top; it != nil; it = it.Next {
		if it.IsWrap() {
			wrapped++
			require.True(t, m.isQueryError(it))
		}
	}
	require.Greater(t, wrapped, 0)
}

func TestQuery_EscCancelsLikeSearch(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "ab": 2}`)
	top := m.top
	typeKeys(m, ".a")
	enter(m)
	require.Equal(t, []string{"1"}, lines(m))

	typeKeys(m, ".b")
	require.Equal(t, ".ab", m.queryInput.Value())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	drain(m, cmd)

	require.False(t, m.queryInput.Focused())
	require.Equal(t, "", m.queryInput.Value())
	require.Nil(t, m.original)
	require.Same(t, top, m.top)
	require.Equal(t, []string{".a"}, m.queryHistory, "Esc must not add to history")
}

func TestQuery_EscDropsPreviewResult(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	top := m.top
	preview(m, ".a")
	require.NotNil(t, m.original)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	drain(m, cmd)
	require.Nil(t, m.original)
	require.Same(t, top, m.top)
}

func TestQuery_ClearDropsErrorStyles(t *testing.T) {
	m := newQueryModel(t, `1`)
	drain(m, m.doQuery("x => x.y.z"))
	require.NotNil(t, m.queryErrors)
	drain(m, m.doQuery("."))
	require.Nil(t, m.queryErrors)
}

func TestQuery_IdentityQueriesShowOriginal(t *testing.T) {
	for _, q := range []string{"", ".", "x", "this", " x ", ". x"} {
		m := newQueryModel(t, `{"a": 1}`)
		top := m.top
		require.Nil(t, m.doQuery(q), q)
		require.Nil(t, m.original, q)
		require.Same(t, top, m.top, q)
	}
}

func TestQuery_ViewEveryLine(t *testing.T) {
	m := newQueryModel(t, `{"n": {"x": 1}}`, `{"n": {"x": 2}}`, `{"id": 3}`)
	m.wrap = true
	drain(m, m.doQuery(".n.x.toFixed(1)"))
	for it := m.top; it != nil; it = it.Next {
		require.NotZero(t, it.LineNumber, "%q", it.Value)
	}
	for m.cursor = 0; m.cursor < len(lines(m)); m.cursor++ {
		require.NotPanics(t, func() { m.View() })
	}
}

// preview types s into the focused query input and runs the debounced preview.
func preview(m *model, s string) {
	if !m.queryInput.Focused() {
		typeKeys(m, ".")
	}
	m.queryInput.SetValue(s)
	m.schedulePreview()
	drain(m, m.handlePreviewTick(previewTickMsg{seq: m.previewSeq}))
}

func TestPreview_AppliesOnSmallInput(t *testing.T) {
	m := newQueryModel(t, `{"a": [1, 2], "b": null}`)
	preview(m, ".a")
	require.True(t, m.queryInput.Focused(), "preview must not blur the input")
	require.Equal(t, []string{"[", "1", "2", "]"}, lines(m))
	require.NotNil(t, m.original)
}

func TestPreview_KeepsLastGoodResult(t *testing.T) {
	m := newQueryModel(t, `{"a": [1, 2], "b": null}`)
	preview(m, ".a")
	for _, q := range []string{".nonexistent", ".b", "x => { throw 1 }", ".a[", "x => skip"} {
		preview(m, q)
		require.Equal(t, []string{"[", "1", "2", "]"}, lines(m), q)
	}
}

func TestPreview_ErrorShownOnEnter(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	preview(m, ".a")
	preview(m, ".nonexistent")
	require.Equal(t, []string{"1"}, lines(m))

	enter(m)
	require.Equal(t, []string{"undefined"}, lines(m))
	require.True(t, m.isQueryError(m.top))
}

func TestPreview_OriginalUntouchedUntilGoodResult(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	top := m.top
	preview(m, ".nonexistent")
	require.Nil(t, m.original)
	require.Same(t, top, m.top)
}

func TestPreview_Gate(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	require.Equal(t, previewRun, m.previewAction(".a"))
	require.Equal(t, previewNone, m.previewAction("."))
	require.Equal(t, previewNone, m.previewAction("x"))
	require.Equal(t, previewRun, m.previewAction(".exit"), "keys named like functions")
	require.Equal(t, previewRun, m.previewAction(".save"), "keys named like functions")

	m.totalLines = previewMaxLines + 1
	require.Equal(t, previewNone, m.previewAction(".a"), "large input")

	m.totalLines = 1
	m.eof = false
	require.Equal(t, previewNone, m.previewAction(".a"), "input still streaming")
}

func TestPreview_ExitNotApplied(t *testing.T) {
	for _, q := range []string{
		`x => exit()`,
		`x => exit(0)`,
		`x => exit(-1)`,
		`x => exit(1)`,
		`x => x.a == 1 ? x : exit(0)`, // After partial output.
		`x => (println("a"), exit())`,
	} {
		t.Run(q, func(t *testing.T) {
			m := newQueryModel(t, `{"a": 1}`, `{"a": 2}`)
			preview(m, q)
			require.Nil(t, m.original, "failed preview must not be applied")
		})
	}
}

func TestQuery_NegativeExitIsError(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	drain(m, m.doQuery("x => exit(-1)"))
	require.Equal(t, []string{"exit(-1) is not allowed in interactive mode"}, lines(m))
}

func TestPreview_GateUsesOriginalSize(t *testing.T) {
	m := newQueryModel(t, `1`)
	m.totalLines = previewMaxLines + 1
	drain(m, m.doQuery("x => 0"))
	require.Equal(t, 1, m.totalLines, "result view is small")
	require.Equal(t, previewNone, m.previewAction(".a"), "original is large")
}

func TestPreview_StaleTickIgnored(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ".")
	m.queryInput.SetValue(".a")
	m.schedulePreview()
	stale := m.previewSeq
	m.schedulePreview()
	require.Nil(t, m.handlePreviewTick(previewTickMsg{seq: stale}))
	require.Nil(t, m.livePreview)
}

func TestPreview_EscDropsRunningPreview(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ".")
	m.queryInput.SetValue(".a")
	m.schedulePreview()
	cmd := m.handlePreviewTick(previewTickMsg{seq: m.previewSeq})
	run := m.livePreview
	require.NotNil(t, run)

	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	require.Nil(t, m.livePreview)
	require.True(t, run.stopped)

	drain(m, cmd)
	require.Nil(t, m.original, "stopped preview must not be applied")
}

func TestPreview_TypingSchedulesTick(t *testing.T) {
	m := newQueryModel(t, `{"a": 1}`)
	typeKeys(m, ".")
	seq := m.previewSeq
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.NotNil(t, cmd)
	require.Equal(t, seq+1, m.previewSeq)
}

func TestSnapshot(t *testing.T) {
	var l docList
	l.add(t, `{"a": {"b": [1, "`+strings.Repeat("word ", 30)+`"]}, "c": true}`)
	l.bottom.Next.Collapse() // "a"
	text := &jsonx.Node{Kind: jsonx.Err, Value: "not json"}
	l.bottom.Adjacent(text)
	l.bottom = text
	l.add(t, `"s"`)
	l.add(t, `2`)
	jsonx.Wrap(l.head, 20)

	var docs []any
	dec := json.NewDecoder(bytes.NewReader(snapshot(l.head)))
	for dec.More() {
		var v any
		require.NoError(t, dec.Decode(&v))
		docs = append(docs, v)
	}
	require.Equal(t, []any{
		map[string]any{"a": map[string]any{"b": []any{1.0, strings.Repeat("word ", 30)}}, "c": true},
		"s",
		2.0,
	}, docs)
}

func up(m *model)   { m.Update(tea.KeyMsg{Type: tea.KeyUp}) }
func down(m *model) { m.Update(tea.KeyMsg{Type: tea.KeyDown}) }

func TestHistory_UpDownAndDraft(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": 2}`)
	typeKeys(m, ".a")
	enter(m)
	typeKeys(m, ".")
	m.queryInput.SetValue(".b")
	enter(m)

	typeKeys(m, ".")
	m.queryInput.SetValue(".dra")
	up(m)
	require.Equal(t, ".b", m.queryInput.Value())
	up(m)
	require.Equal(t, ".a", m.queryInput.Value())
	up(m)
	require.Equal(t, ".a", m.queryInput.Value(), "stays at oldest")
	down(m)
	require.Equal(t, ".b", m.queryInput.Value())
	down(m)
	require.Equal(t, ".dra", m.queryInput.Value(), "draft restored")
	down(m)
	require.Equal(t, ".dra", m.queryInput.Value())
	require.Equal(t, []string{"2"}, lines(m), "browsing must not apply")
}

func TestHistory_AddRules(t *testing.T) {
	m := newQueryModel(t)
	for _, q := range []string{".a", ".a", " .a ", ".", "", "x", ".b", ".a"} {
		m.addQueryHistory(q)
	}
	require.Equal(t, []string{".a", ".b", ".a"}, m.queryHistory)

	for i := 0; i < queryHistorySize+10; i++ {
		m.addQueryHistory(fmt.Sprintf(".k%d", i))
	}
	require.Len(t, m.queryHistory, queryHistorySize)
	require.Equal(t, fmt.Sprintf(".k%d", queryHistorySize+9), m.queryHistory[queryHistorySize-1])
}

func TestHistory_BrowsingDoesNotPreview(t *testing.T) {
	m := newQueryModel(t, `{"a": 1, "b": 2}`)
	m.addQueryHistory(".a")
	typeKeys(m, ".")
	m.queryInput.SetValue(".b")
	m.schedulePreview()
	pending := m.previewSeq

	up(m)
	require.Equal(t, ".a", m.queryInput.Value())
	require.Nil(t, m.handlePreviewTick(previewTickMsg{seq: pending}), "pending tick dropped")
	require.Nil(t, m.livePreview)
	require.Nil(t, m.original)
}

func TestQuery_StatusBarKept(t *testing.T) {
	m := newQueryModel(t, `{"a": [1, 2]}`)
	m.fileName = "file.json"
	m.termHeight = 10

	screenLines := func() []string {
		return strings.Split(m.View(), "\n")
	}

	typeKeys(m, ".a")
	got := screenLines()
	require.Len(t, got, m.termHeight)
	require.Contains(t, got[len(got)-2], "file.json", "status bar while editing")
	require.Contains(t, got[len(got)-1], ".a", "query line while editing")

	enter(m)
	got = screenLines()
	require.Len(t, got, m.termHeight)
	require.Contains(t, got[len(got)-2], "file.json", "status bar while query applied")
	require.Contains(t, got[len(got)-1], ".a", "query line while query applied")

	typeKeys(m, ".")
	m.queryInput.SetValue(".")
	enter(m)
	got = screenLines()
	require.Len(t, got, m.termHeight)
	require.Contains(t, got[len(got)-1], "file.json", "status bar is last line after clear")
}

func TestQuery_ClearNeverEndingQuery(t *testing.T) {
	m := newQueryModel(t, `1`)
	top := m.top
	cmd := m.doQuery("x => { while (true) {} }")
	time.Sleep(50 * time.Millisecond)
	m.doQuery(".")

	done := make(chan struct{})
	go func() {
		drain(m, cmd)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("restore blocked by a never-ending query")
	}
	require.Nil(t, m.original)
	require.Same(t, top, m.top)
	require.Empty(t, lines(m)[1:], "no error for a cancelled run")
}

func TestPreview_NeverEndingQueryStops(t *testing.T) {
	m := newQueryModel(t, `1`)
	typeKeys(m, ".")
	m.queryInput.SetValue("x => { while (true) {} }")
	m.schedulePreview()
	cmd := m.handlePreviewTick(previewTickMsg{seq: m.previewSeq})
	run := m.livePreview
	time.Sleep(50 * time.Millisecond)
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})

	done := make(chan struct{})
	go func() {
		drain(m, cmd)
		close(done)
	}()
	select {
	case <-done:
		require.True(t, run.done)
	case <-time.After(2 * time.Second):
		t.Fatal("preview goroutine still running")
	}
}

func TestQuery_RestoreAfterWrapToggle(t *testing.T) {
	long := `{"s": "` + strings.Repeat("word ", 30) + `"}`
	hasChunks := func(top *jsonx.Node) bool {
		for it := top; it != nil; it = it.Next {
			if it.IsWrap() {
				return true
			}
		}
		return false
	}
	toggleWrap := func(m *model) {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyMap.ToggleWrap.Keys()[0])})
	}

	t.Run("turned off", func(t *testing.T) {
		m := newQueryModel(t, long)
		m.wrap = true
		jsonx.Wrap(m.top, m.viewWidth())
		m.head = m.top.Next.Next // Scrolled to a chunk.
		m.cursor = 0
		require.True(t, m.head.IsWrap())

		drain(m, m.doQuery(".s"))
		toggleWrap(m)
		require.False(t, m.wrap)
		drain(m, m.doQuery("."))

		require.False(t, hasChunks(m.top), "original must be unwrapped")
		require.False(t, m.head.IsWrap(), "scroll position must not be a dropped chunk")
		require.NotPanics(t, func() { m.View() })
	})

	t.Run("unchanged", func(t *testing.T) {
		m := newQueryModel(t, long)
		m.wrap = true
		jsonx.Wrap(m.top, m.viewWidth())
		chunk := m.top.Next.Next
		m.head = chunk // Scrolled to a chunk.
		m.cursor = 0
		require.True(t, m.head.IsWrap())

		drain(m, m.doQuery(".s"))
		drain(m, m.doQuery("."))

		require.Same(t, chunk, m.head, "scroll position must be kept")
		require.Equal(t, 0, m.cursor)
	})

	t.Run("turned on", func(t *testing.T) {
		m := newQueryModel(t, long)
		m.wrap = false
		drain(m, m.doQuery(".s"))
		toggleWrap(m)
		require.True(t, m.wrap)
		drain(m, m.doQuery("."))
		require.True(t, hasChunks(m.top), "original must be wrapped")
	})
}

func TestPreview_CannotSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"a": 1}`), 0644))
	old := engine.FilePath
	engine.FilePath = path
	t.Cleanup(func() { engine.FilePath = old })

	m := newQueryModel(t, `{"a": 1}`)
	preview(m, `x => (__save__("{}"), x)`)

	data, _ := os.ReadFile(path)
	require.Equal(t, `{"a": 1}`, string(data))
	require.Nil(t, m.original, "failed preview must not be applied")
}

func TestQuery_StaleSearchResultIgnoredAfterSwap(t *testing.T) {
	stale := func(m *model) searchResultMsg {
		s := newSearch()
		s.results = []*jsonx.Node{m.top}
		return searchResultMsg{id: m.searchID, search: s}
	}

	t.Run("apply", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		msg := stale(m) // Search of the original, finished but not yet delivered.
		drain(m, m.doQuery(".a"))
		m.Update(msg)
		require.Empty(t, m.search.results)
		require.Equal(t, []string{"1"}, lines(m))
	})

	t.Run("restore", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		drain(m, m.doQuery(".a"))
		msg := stale(m) // Search of the result view.
		drain(m, m.doQuery("."))
		m.Update(msg)
		require.Empty(t, m.search.results)
	})
}

func TestPreview_IdentityShowsOriginal(t *testing.T) {
	t.Run("after preview", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		top := m.top
		preview(m, ".a")
		require.NotNil(t, m.original)

		preview(m, ".")
		require.Nil(t, m.original)
		require.Same(t, top, m.top)
		require.True(t, m.queryInput.Focused(), "input stays open")
		require.Equal(t, ".", m.queryInput.Value())
	})

	t.Run("after enter", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		typeKeys(m, ".a")
		enter(m)
		typeKeys(m, ".")
		preview(m, ".")
		require.Nil(t, m.original)
	})

	t.Run("large input waits for enter", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		m.totalLines = previewMaxLines + 1
		drain(m, m.doQuery(".a"))
		typeKeys(m, ".")
		m.queryInput.SetValue(".")
		require.Nil(t, m.schedulePreview())
		require.NotNil(t, m.original)
	})

	t.Run("nothing to clear", func(t *testing.T) {
		m := newQueryModel(t, `{"a": 1}`)
		typeKeys(m, ".")
		require.Nil(t, m.schedulePreview())
	})
}

func TestQuery_EmptyResultView(t *testing.T) {
	keys := []tea.KeyMsg{
		{Type: tea.KeyPgUp}, {Type: tea.KeyPgDown}, {Type: tea.KeyHome}, {Type: tea.KeyEnd},
		{Type: tea.KeyUp}, {Type: tea.KeyDown}, {Type: tea.KeyLeft}, {Type: tea.KeyRight},
		{Type: tea.KeyCtrlU}, {Type: tea.KeyCtrlD}, {Type: tea.KeyShiftUp}, {Type: tea.KeyShiftDown},
		{Type: tea.KeyShiftLeft}, {Type: tea.KeyShiftRight}, {Type: tea.KeyCtrlG},
	}
	for _, r := range "bfgGjkhlJKHLeE123zsnN[]" {
		keys = append(keys, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	for _, k := range keys {
		t.Run(k.String(), func(t *testing.T) {
			m := newQueryModel(t, `{"a": 1}`)
			drain(m, m.doQuery("x => skip"))
			require.Nil(t, m.top)
			_, cmd := m.Update(k)
			drain(m, cmd)
			m.View()
		})
	}
	for _, input := range []string{":1", ":5", "/a"} {
		t.Run(input, func(t *testing.T) {
			m := newQueryModel(t, `{"a": 1}`)
			drain(m, m.doQuery("x => skip"))
			typeKeys(m, input)
			enter(m)
			m.View()
		})
	}
}

// runSearch runs a search and delivers its result, skipping the spinner.
func runSearch(m *model, s string) {
	typeKeys(m, "/"+s)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for _, c := range cmd().(tea.BatchMsg) {
		if msg, ok := c().(searchResultMsg); ok {
			m.Update(msg)
		}
	}
}

func TestQuery_SearchTextSavedWithResults(t *testing.T) {
	m := newQueryModel(t, `{"foo": 1, "bar": 2}`)
	runSearch(m, "bar")
	require.NotEmpty(t, m.search.results)
	original := m.search

	drain(m, m.doQuery("x => ({foo: x.foo})"))
	require.Empty(t, m.searchInput.Value(), "result view starts without a search")
	runSearch(m, "foo")
	require.NotEmpty(t, m.search.results)

	drain(m, m.doQuery("."))
	require.Same(t, original, m.search)
	require.Equal(t, "bar", m.searchInput.Value())
}

// markTokenColors makes the query highlight colors visible in the output.
func markTokenColors(t *testing.T) {
	saved := theme.CurrentTheme
	t.Cleanup(func() { theme.CurrentTheme = saved })
	mark := func(name string) theme.Color {
		return func(s string) string { return name + "(" + s + ")" }
	}
	theme.CurrentTheme.Key = mark("key")
	theme.CurrentTheme.String = mark("str")
	theme.CurrentTheme.Number = mark("num")
	theme.CurrentTheme.Boolean = mark("bool")
	theme.CurrentTheme.Null = mark("null")
	theme.CurrentTheme.Syntax = mark("syn")
	theme.CurrentTheme.Preview = mark("dim")
}

func newQueryInputModel(t *testing.T, value string) *model {
	markTokenColors(t)
	m := newQueryModel(t, `{}`)
	m.queryInput.Prompt = ""
	m.queryInput.SetValue(value)
	return m
}

func TestQueryInputView_Highlights(t *testing.T) {
	m := newQueryInputModel(t, `@.name x => x.a ?? "b" typeof 1 null // c`)
	require.Equal(t,
		`syn(@.)key(name) x syn(=>) xsyn(.)key(a) syn(??) str("b") bool(typeof) num(1) null(null) dim(// c)`,
		m.queryInputView())
}

func TestQueryInputView_Unicode(t *testing.T) {
	m := newQueryInputModel(t, `.ключ "日本 語" ok`)
	require.Equal(t, `syn(.)key(ключ) str("日本 語") ok`, m.queryInputView())
}

func TestQueryInputView_Scrolls(t *testing.T) {
	m := newQueryInputModel(t, "aaaaa bbbbb")
	m.queryInput.Width = 5

	m.queryInput.CursorEnd()
	require.Equal(t, "bbbbb", m.queryInputView(), "end: last 5 columns, cursor after")

	m.queryInput.SetCursor(8)
	require.Equal(t, "bbbbb", m.queryInputView(), "cursor inside the window: no scroll")

	m.queryInput.SetCursor(2)
	require.Equal(t, "aaa b", m.queryInputView(), "cursor left of the window: scroll to it")

	m.queryInput.SetCursor(0)
	require.Equal(t, "aaaaa", m.queryInputView())

	m.queryInput.SetCursor(7)
	require.Equal(t, "aa bb", m.queryInputView(), "cursor right of the window: scroll to it")
}

func TestQueryInputView_WideRunes(t *testing.T) {
	m := newQueryInputModel(t, `"日本語"`)
	m.queryInput.Width = 4
	m.queryInput.CursorEnd()
	require.Equal(t, `str(語")`, m.queryInputView())
}

// saveFile makes path the file argument, as `fx path`.
func saveFile(t *testing.T, content string) string {
	path := filepath.Join(t.TempDir(), "file.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	old := engine.FilePath
	engine.FilePath = path
	t.Cleanup(func() { engine.FilePath = old })
	return path
}

func requireFile(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, want, string(data))
}

func TestQuery_SaveRefusesSeveralDocuments(t *testing.T) {
	const input = "{\"a\": 1}\n{\"a\": 2}\n"
	path := saveFile(t, input)
	m := newQueryModel(t, `{"a": 1}`, `{"a": 2}`)
	drain(m, m.doQuery("save"))
	requireFile(t, path, input)
	require.Contains(t, strings.Join(lines(m), "\n"), "save supports a single JSON value")
}

// save() of the first document waits for the rest of a still loading file.
func TestQuery_SaveWaitsForStreamedDocuments(t *testing.T) {
	const input = "{\"a\": 1}\n{\"a\": 2}\n"
	path := saveFile(t, input)
	m := newQueryModel(t, `{"a": 1}`)
	m.eof = false

	cmd := m.doQuery("save")
	time.Sleep(50 * time.Millisecond) // The engine is blocked in save().
	requireFile(t, path, input)

	m.Update(nodeMsg{node: parseDoc(t, `{"a": 2}`)})
	drain(m, cmd)
	requireFile(t, path, input)
	require.Contains(t, strings.Join(lines(m), "\n"), "save supports a single JSON value")
}

func TestQuery_SaveSingleStreamedDocument(t *testing.T) {
	path := saveFile(t, `{"a": 1}`)
	m := newQueryModel(t, `{"a": 1}`)
	m.eof = false

	cmd := m.doQuery("x.a = 2, save(x)")
	m.Update(eofMsg{})
	drain(m, cmd)
	requireFile(t, path, "{\n  \"a\": 2\n}\n")
	require.False(t, m.query.gotErr)
}

// Recovered text is not passed to the engine, so save() would drop it.
func TestQuery_SaveRefusesRecoveredText(t *testing.T) {
	text := func() *jsonx.Node { return &jsonx.Node{Kind: jsonx.Err, Value: `{"b":`} }
	for name, docs := range map[string][]*jsonx.Node{
		"after":  {parseDoc(t, `{"a": 1}`), text()},
		"before": {text(), parseDoc(t, `{"a": 1}`)},
	} {
		t.Run(name, func(t *testing.T) {
			const input = "{\"a\":1}\n{\"b\":"
			path := saveFile(t, input)
			m := newQueryModel(t)
			m.eof = false
			for _, doc := range docs {
				m.Update(nodeMsg{node: doc})
			}
			m.Update(eofMsg{})
			drain(m, m.doQuery("save"))
			requireFile(t, path, input)
			require.Contains(t, strings.Join(lines(m), "\n"), "input contains text that is not JSON")
		})
	}
}
