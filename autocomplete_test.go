package main

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/require"

	"github.com/antonmedv/fx/internal/complete"
)

func replyValues(m *model) []string {
	var out []string
	for _, r := range m.completion.replies {
		out = append(out, r.Value)
	}
	return out
}

func pressKey(m *model, k tea.KeyPressMsg) tea.Cmd {
	_, cmd := m.Update(k)
	return cmd
}

const completeData = `{"items": [{"name": 1, "nick": 2}, {"name": 3, "id": 4}], "id": 1, "i-d": 2}`

func TestComplete_GhostAndGrid(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".")
	require.Equal(t, []string{".items", ".id", `.["i-d"]`}, replyValues(m))
	require.Equal(t, "items", m.completionGhost())
	require.Equal(t, []string{`.items    .id       .["i-d"]`}, plain(m.completionGridView()))
	require.Equal(t, m.termHeight-3, m.viewHeight())

	typeKeys(m, "id")
	require.Equal(t, []string{".id"}, replyValues(m))
	require.Equal(t, "", m.completionGhost())
	require.Nil(t, m.completionGridView(), "a typed key is not listed")
	require.Equal(t, m.termHeight-2, m.viewHeight())

	screen := ansi.Strip(view(m))
	require.True(t, strings.HasSuffix(screen, "\n> .id"), screen) // the terminal cursor follows, nothing is drawn for it
}

func TestComplete_GhostIsRendered(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".it")
	lines := strings.Split(ansi.Strip(view(m)), "\n")
	require.Equal(t, "> .items", lines[len(lines)-1])
}

func TestComplete_ArraysAndTab(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".it")
	drain(m, pressKey(m, press("tab")))
	require.Equal(t, ".items", m.queryInput.Value())

	typeKeys(m, ".")
	require.Equal(t, []string{".items[].name", ".items[].nick", ".items[].id"}, replyValues(m))
	require.Equal(t, "", m.completionGhost(), "the replies do not extend `.items.`")
	require.Len(t, m.completionGridView(), 1)

	drain(m, pressKey(m, press("tab"))) // common prefix
	require.Equal(t, ".items[].", m.queryInput.Value())
	drain(m, pressKey(m, press("tab"))) // menu
	require.Equal(t, ".items[].name", m.queryInput.Value())
	drain(m, pressKey(m, press("tab")))
	require.Equal(t, ".items[].nick", m.queryInput.Value())
	drain(m, pressKey(m, press("shift+tab")))
	drain(m, pressKey(m, press("shift+tab")))
	require.Equal(t, ".items[].id", m.queryInput.Value())
	require.Contains(t, m.completionGridView()[0], reverseStyle(".id"))

	pressKey(m, press("backspace")) // leaves the menu
	require.False(t, m.completion.menu)
	require.Equal(t, []string{".items[].id"}, replyValues(m))
}

func TestComplete_RightAcceptsGhost(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".it")
	drain(m, pressKey(m, press("right")))
	require.Equal(t, ".items", m.queryInput.Value())

	// Without a ghost, right moves the cursor as usual.
	pressKey(m, press("left"))
	pressKey(m, press("right"))
	require.Equal(t, ".items", m.queryInput.Value())
}

func TestComplete_OnlyAtEnd(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".i")
	pressKey(m, press("left"))
	require.Nil(t, m.completion.replies)
	require.Equal(t, "", m.completionGhost())
}

func TestComplete_Bracket(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, `.["i`)
	require.Equal(t, []string{`.["items"]`, `.["id"]`, `.["i-d"]`}, replyValues(m))
	typeKeys(m, `-`)
	drain(m, pressKey(m, press("tab")))
	require.Equal(t, `.["i-d"]`, m.queryInput.Value())
}

func TestComplete_Engine(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".items.map(x => x")
	_, cmd := m.Update(press("."))
	require.Nil(t, m.completion.replies)
	require.NotNil(t, cmd, "the engine is scheduled")

	drain(m, m.handleCompleteTick(completeTickMsg{seq: m.completion.seq}))
	require.Equal(t, []string{".items.map(x => x.name", ".items.map(x => x.nick", ".items.map(x => x.id"}, replyValues(m))

	// Typing a key filters the cached keys, without the engine.
	typeKeys(m, "n")
	require.Nil(t, m.completion.cancel)
	require.Equal(t, []string{".items.map(x => x.name", ".items.map(x => x.nick"}, replyValues(m))
}

func TestComplete_EngineStaleIsDropped(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".items.map(x => x.")
	tick := m.handleCompleteTick(completeTickMsg{seq: m.completion.seq})
	typeKeys(m, "n") // cancels the running engine
	drain(m, tick)
	require.Nil(t, m.completion.replies)
}

func TestComplete_Streaming(t *testing.T) {
	m := newQueryModel(t, `{"a": {"x": 1}}`)
	m.eof = false
	typeKeys(m, ".a.")
	require.Equal(t, []string{".a.x"}, replyValues(m))

	_, cmd := m.Update(nodeMsg{node: parseDoc(t, `{"a": {"y": 1}}`)})
	m.Update(nodeMsg{node: parseDoc(t, `{"a": {"z": 1}}`)})
	drain(m, cmd)
	require.Equal(t, []string{".a.x", ".a.y", ".a.z"}, replyValues(m))
}

func TestComplete_AfterPreviewUsesOriginal(t *testing.T) {
	m := newQueryModel(t, `{"a": {"x": 1}}`)
	preview(m, ".a")
	require.NotNil(t, m.original)
	m.queryInput.CursorEnd()
	typeKeys(m, ".")
	require.Equal(t, []string{".a.x"}, replyValues(m))
}

func TestComplete_EscAndEnterHide(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".")
	pressKey(m, press("esc"))
	require.Nil(t, m.completion.replies)
	require.Equal(t, m.termHeight-1, m.viewHeight())

	typeKeys(m, ".i")
	enter(m)
	require.Nil(t, m.completion.replies)
	require.Nil(t, m.completionGridView())
}

func TestComplete_GridScrolls(t *testing.T) {
	var b strings.Builder
	b.WriteString("{")
	for i := range 200 {
		if i > 0 {
			b.WriteString(",")
		}
		fmt.Fprintf(&b, `"key%03d": 1`, i)
	}
	b.WriteString("}")
	m := newQueryModel(t, b.String())
	typeKeys(m, ".k")
	grid := plain(m.completionGridView())
	require.Len(t, grid, completeMaxRows)
	require.Equal(t, "rows 1 to 9 of 23", grid[len(grid)-1])
	require.Equal(t, m.termHeight-2-completeMaxRows, m.viewHeight())

	drain(m, pressKey(m, press("shift+tab"))) // common prefix
	require.Equal(t, ".key", m.queryInput.Value())
	drain(m, pressKey(m, press("shift+tab"))) // menu at the last reply
	require.Equal(t, ".key199", m.queryInput.Value())
	grid = plain(m.completionGridView())
	require.Equal(t, "rows 15 to 23 of 23", grid[len(grid)-1])
}

// typeQuery opens the query input and types query in place of its ".".
func typeQuery(m *model, query string) {
	typeKeys(m, ".")
	pressKey(m, press("backspace"))
	typeKeys(m, query)
}

func plain(lines []string) []string {
	if lines == nil {
		return nil
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		out[i] = strings.TrimRight(ansi.Strip(line), " ")
	}
	return out
}

// BenchmarkComplete_Keystroke measures completing on a 100k-line input
// while it streams in: every document is a nodeMsg.
func BenchmarkComplete_Keystroke(b *testing.B) {
	t := &testing.T{}
	m := newQueryModel(t)
	m.eof = false
	for i := range 100_000 {
		m.Update(nodeMsg{node: parseDoc(t, fmt.Sprintf(`{"id": %d, "user": {"login": "u", "k%d": 1}}`, i, i%50))})
	}
	typeKeys(m, ".user.k")
	var cache complete.Cache
	b.Run("fresh", func(b *testing.B) {
		for b.Loop() {
			m.completion.cache = cache // cold: walks all documents
			m.updateCompletion()
		}
	})
	b.Run("typing", func(b *testing.B) {
		for b.Loop() {
			typeKeys(m, "1")
			pressKey(m, press("backspace"))
		}
	})
	b.Run("streaming", func(b *testing.B) {
		doc := `{"id": 0, "user": {"login": "u", "new": 1}}`
		for b.Loop() {
			m.Update(nodeMsg{node: parseDoc(t, doc)})
			m.handleCompleteStream()
		}
	})
}

func TestComplete_StreamingKeepsEngineScheduled(t *testing.T) {
	m := newQueryModel(t, `{"a-b": 1, "c": 2}`)
	m.eof = false
	typeQuery(m, "x => x .")
	require.True(t, m.completion.engine)
	seq := m.completion.seq

	_, cmd := m.Update(nodeMsg{node: parseDoc(t, `{"d": 3}`)})
	drain(m, cmd)
	require.Equal(t, seq, m.completion.seq, "streaming restarted the engine debounce")

	drain(m, m.handleCompleteTick(completeTickMsg{seq: seq}))
	require.Equal(t, []string{`.["a-b"]`, ".c"}, replyValues(m))
}

func TestComplete_EngineEmptyBaseBracket(t *testing.T) {
	for query, want := range map[string][]string{
		"x => x .":  {`.["a-b"]`, ".c"},
		"x => x .[": {`.["a-b"]`, `.["c"]`},
		"x => x ":   {`.["a-b"]`, ".c"},
	} {
		t.Run(query, func(t *testing.T) {
			docs := []string{`{"a-b": 1, "c": 2}`}
			m := newQueryModel(t, docs...)
			typeQuery(m, query)
			drain(m, m.handleCompleteTick(completeTickMsg{seq: m.completion.seq}))
			require.Equal(t, want, replyValues(m))
		})
	}
}

func TestComplete_ReopenDropsOldTick(t *testing.T) {
	m := newQueryModel(t, completeData)
	typeKeys(m, ".items.map(x => x.")
	old := completeTickMsg{seq: m.completion.seq}
	pressKey(m, press("esc"))
	typeKeys(m, ".items.map(x => x.")
	require.Nil(t, m.handleCompleteTick(old), "a tick of the closed input started an engine")
}

func TestComplete_GridFitsShortTerminal(t *testing.T) {
	m := newQueryModel(t, completeData)
	m.termWidth, m.termHeight = 20, 7
	typeKeys(m, ".")
	require.Greater(t, m.layoutGrid().rows, 1)
	require.Len(t, m.completionGridView(), m.completionRows())
	require.Len(t, strings.Split(view(m), "\n"), m.termHeight)
}

func TestComplete_SingleQuotedControlCharacters(t *testing.T) {
	m := newQueryModel(t, `{"hello\nworld": 42}`)
	typeQuery(m, `.['h`)
	pressKey(m, press("tab"))
	require.Equal(t, `.['hello\nworld']`, m.queryInput.Value())
	enter(m)
	require.Equal(t, []string{"42"}, lines(m), "the key is found")
}
