package main

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/antonmedv/fx/internal/complete"
	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

// completion is the autocompletion of the query input. Keys of plain paths
// are walked in the loaded documents on the UI goroutine; other queries run
// the preview engine on a copy of the first document, in the background.
type completion struct {
	cache    complete.Cache
	query    string // query up to the cursor, the replies are for
	start    int    // byte offset of the completed word in query
	word     string
	replies  []complete.Reply
	menu     bool // Tab cycles through the replies
	selected int  // reply inserted by the menu
	seq      uint64
	cancel   chan struct{} // stops the running engine, nil if none
	engine   bool          // the replies come from the engine, not the documents
	dirty    bool          // documents arrived since the replies were computed
	ticking  bool          // a refresh for streamed documents is scheduled
	first    *firstDoc
	grid     gridLayout
}

type gridLayout struct {
	valid          bool
	width, height  int
	cols, colWidth int
	rows, shown    int // rows in all, rows shown
}

// firstDoc is a private copy of the first document for the engine, which
// must not read the displayed nodes the UI mutates.
type firstDoc struct {
	source *Node
	data   []byte
	parse  func() *Node
}

func newFirstDoc(source *Node) *firstDoc {
	var b bytes.Buffer
	writeDoc(&b, source)
	d := &firstDoc{source: source, data: b.Bytes()}
	d.parse = sync.OnceValue(func() *Node {
		node, err := Parse(d.data)
		if err != nil {
			return nil
		}
		return node
	})
	return d
}

const (
	completeEngineDelay = 100 * time.Millisecond
	completeStreamDelay = 50 * time.Millisecond
	completeMaxRows     = 10
)

type completeTickMsg struct {
	seq uint64
}

type completeEngineMsg struct {
	seq     uint64
	request *complete.Request
	keys    complete.Names
	cancel  chan struct{}
}

type completeStreamMsg struct{}

// completionDocs returns the documents queries run on.
func (m *model) completionDocs() complete.Docs {
	top := m.top
	if m.original != nil {
		top = m.original.top
	}
	return complete.Docs{First: top, Next: nextDoc}
}

// resetCompletion starts completing a newly focused query input. The
// documents may have changed (deleted) since the last time.
func (m *model) resetCompletion() tea.Cmd {
	m.stopCompletion()
	// seq keeps counting, so ticks of the previous input stay stale.
	m.completion = completion{seq: m.completion.seq}
	return m.updateCompletion()
}

// stopCompletion cancels the running engine and hides the replies.
func (m *model) stopCompletion() {
	c := &m.completion
	c.seq++
	if c.cancel != nil {
		close(c.cancel)
		c.cancel = nil
	}
	c.menu = false
	c.setReplies(nil)
}

func (c *completion) setReplies(replies []complete.Reply) {
	c.replies = replies
	c.grid.valid = false
}

// updateCompletion computes the replies for the query up to the cursor.
// Only a cursor at the end of the query is completed.
func (m *model) updateCompletion() tea.Cmd {
	m.stopCompletion()
	c := &m.completion
	c.dirty = false
	value := []rune(m.queryInput.Value())
	if !m.queryInput.Focused() || m.queryInput.Position() != len(value) {
		c.query, c.word = "", ""
		return nil
	}
	c.query = string(value)
	r, start := complete.ParseQuery(c.query)
	c.start, c.word = start, r.Word
	replies, needEngine := c.cache.Complete(r, m.completionDocs())
	c.setReplies(replies)
	c.engine = needEngine
	if !needEngine {
		return nil
	}
	seq := c.seq
	return tea.Tick(completeEngineDelay, func(time.Time) tea.Msg {
		return completeTickMsg{seq: seq}
	})
}

func (m *model) handleCompleteTick(msg completeTickMsg) tea.Cmd {
	c := &m.completion
	if msg.seq != c.seq || !m.queryInput.Focused() {
		return nil
	}
	first := m.completionDocs().First
	for first != nil && first.Kind == Err {
		first = nextDoc(first)
	}
	if first == nil {
		return nil
	}
	if c.first == nil || c.first.source != first {
		c.first = newFirstDoc(first)
	}
	r, _ := complete.ParseQuery(c.query)
	doc, cancel, seq := c.first, make(chan struct{}), c.seq
	c.cancel = cancel
	return func() tea.Msg {
		var keys complete.Names
		if node := doc.parse(); node != nil {
			keys = r.EngineKeys(node, cancel)
		}
		return completeEngineMsg{seq: seq, request: r, keys: keys, cancel: cancel}
	}
}

func (m *model) handleCompleteEngine(msg completeEngineMsg) {
	c := &m.completion
	select {
	case <-msg.cancel:
		return // interrupted, the keys may be incomplete
	default:
	}
	c.cancel = nil
	c.cache.PutEngine(msg.request, msg.keys)
	if msg.seq == c.seq {
		replies, _ := c.cache.Complete(msg.request, m.completionDocs())
		c.setReplies(replies)
	}
}

// completionDocsArrived schedules a refresh of the replies with streamed
// documents, at most every completeStreamDelay.
func (m *model) completionDocsArrived() tea.Cmd {
	c := &m.completion
	if !m.queryInput.Focused() {
		return nil
	}
	c.dirty = true
	if c.ticking {
		return nil
	}
	c.ticking = true
	return tea.Tick(completeStreamDelay, func(time.Time) tea.Msg {
		return completeStreamMsg{}
	})
}

func (m *model) handleCompleteStream() tea.Cmd {
	c := &m.completion
	c.ticking = false
	// The engine reads only the first document, so streamed ones change
	// nothing; refreshing would restart its debounce, forever while streaming.
	if !c.dirty || !m.queryInput.Focused() || c.menu || c.engine {
		return nil
	}
	return m.updateCompletion()
}

// completeKey handles Tab and Shift+Tab: insert the only reply or the
// common prefix of the replies, then cycle through them.
func (m *model) completeKey(dir int) tea.Cmd {
	c := &m.completion
	n := len(c.replies)
	if n == 0 {
		return nil
	}
	if c.menu {
		c.selected = (c.selected + dir + n) % n
		return m.insertCompletion(c.replies[c.selected].Value, false)
	}
	if n == 1 {
		return m.insertCompletion(c.replies[0].Value, true)
	}
	if prefix := commonPrefix(c.replies); len(prefix) > len(c.word) {
		return m.insertCompletion(prefix, true)
	}
	c.menu = true
	c.grid.valid = false
	c.selected = 0
	if dir < 0 {
		c.selected = n - 1
	}
	return m.insertCompletion(c.replies[c.selected].Value, false)
}

// acceptGhost inserts the reply shown after the cursor, if any.
func (m *model) acceptGhost() (tea.Cmd, bool) {
	ghost := m.completionGhost()
	if ghost == "" {
		return nil, false
	}
	return m.insertCompletion(m.completion.word+ghost, true), true
}

// insertCompletion replaces the completed word with value. The menu keeps
// its replies; otherwise they are computed for the new query.
func (m *model) insertCompletion(value string, update bool) tea.Cmd {
	c := &m.completion
	m.queryInput.SetValue(c.query[:c.start] + value)
	m.queryInput.CursorEnd()
	cmd := m.schedulePreview()
	if update {
		return tea.Batch(cmd, m.updateCompletion())
	}
	return cmd
}

func commonPrefix(replies []complete.Reply) string {
	prefix := replies[0].Value
	for _, r := range replies[1:] {
		i := 0
		for i < len(prefix) && i < len(r.Value) && prefix[i] == r.Value[i] {
			i++
		}
		prefix = prefix[:i]
	}
	for !utf8.ValidString(prefix) {
		prefix = prefix[:len(prefix)-1]
	}
	return prefix
}

// completionGhost is the rest of the first reply extending the word, shown
// dimmed after the cursor.
func (m *model) completionGhost() string {
	c := &m.completion
	if c.menu || !m.queryInput.Focused() || m.queryInput.Value() != c.query {
		return ""
	}
	for _, r := range c.replies {
		if len(r.Value) > len(c.word) && strings.HasPrefix(r.Value, c.word) {
			return r.Value[len(c.word):]
		}
	}
	return ""
}

// showGrid reports whether the replies are listed: not for a single reply
// already typed or shown as the ghost.
func (m *model) showGrid() bool {
	c := &m.completion
	if !m.queryInput.Focused() || len(c.replies) == 0 || c.word == "" && !c.menu {
		return false
	}
	if len(c.replies) == 1 && !c.menu {
		r := c.replies[0].Value
		return !strings.HasPrefix(r, c.word)
	}
	return true
}

func (m *model) layoutGrid() *gridLayout {
	c := &m.completion
	g := &c.grid
	if g.valid && g.width == m.termWidth && g.height == m.termHeight {
		return g
	}
	*g = gridLayout{valid: true, width: m.termWidth, height: m.termHeight}
	if !m.showGrid() {
		return g
	}
	maxWidth := 1
	for _, r := range c.replies {
		maxWidth = max(maxWidth, runewidth.StringWidth(r.Display))
	}
	g.colWidth = min(maxWidth, max(1, m.termWidth-1)) + 2
	g.cols = max(1, (m.termWidth+2)/g.colWidth)
	g.rows = (len(c.replies) + g.cols - 1) / g.cols
	maxRows := max(1, min(completeMaxRows, (m.termHeight-2)/3))
	g.shown = min(g.rows, maxRows)
	return g
}

// completionRows is the height of the grid above the query input.
func (m *model) completionRows() int {
	if !m.queryInput.Focused() || len(m.completion.replies) == 0 {
		return 0
	}
	return m.layoutGrid().shown
}

// completionGridView renders the replies in columns, row by row. A list
// taller than the grid scrolls to the selected reply; the last row then
// tells which rows are shown.
func (m *model) completionGridView() []string {
	g := m.layoutGrid()
	if g.shown == 0 {
		return nil
	}
	c := &m.completion
	first, shown := 0, g.shown
	status := g.rows > g.shown && g.shown >= 2 // a single row has no room for it
	if status {
		shown--
	}
	if g.rows > shown && c.menu {
		first = max(0, min(c.selected/g.cols-shown/2, g.rows-shown))
	}
	var lines []string
	for row := first; row < first+shown; row++ {
		var line strings.Builder
		for col := 0; col < g.cols; col++ {
			i := row*g.cols + col
			if i >= len(c.replies) {
				break
			}
			text := runewidth.Truncate(c.replies[i].Display, g.colWidth-2, "…")
			if c.menu && i == c.selected {
				line.WriteString(reverseStyle(text))
			} else {
				line.WriteString(text)
			}
			if col < g.cols-1 {
				line.WriteString(strings.Repeat(" ", g.colWidth-runewidth.StringWidth(text)))
			}
		}
		lines = append(lines, line.String())
	}
	if status {
		text := fmt.Sprintf("rows %d to %d of %d", first+1, first+shown, g.rows)
		lines = append(lines, theme.CurrentTheme.Preview(text))
	}
	return lines
}

// ghostView renders the cursor at the end of the query, followed by the
// dimmed ghost, cut to the input width. shown is the visible query.
func (m *model) ghostView(shown []rune, showCursor bool) string {
	ghost := []rune(m.completionGhost())
	room := len(ghost)
	if width := m.queryInput.Width; width > 0 {
		room = width - runewidth.StringWidth(string(shown))
	}
	var v strings.Builder
	char := " "
	if len(ghost) > 0 && room > 0 {
		char = string(ghost[0])
		ghost = ghost[1:]
		room -= runewidth.StringWidth(char)
	}
	if showCursor {
		cursor := m.queryInput.Cursor
		cursor.SetChar(char)
		v.WriteString(cursor.View())
	} else if char != " " {
		v.WriteString(theme.CurrentTheme.Preview(char))
	}
	if len(ghost) > 0 && room > 0 {
		v.WriteString(theme.CurrentTheme.Preview(runewidth.Truncate(string(ghost), room, "")))
	}
	return v.String()
}
