package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-runewidth"

	"github.com/antonmedv/fx/internal/engine"
	. "github.com/antonmedv/fx/internal/jsonx"
	"github.com/antonmedv/fx/internal/theme"
)

// savedView is the original view, set aside while a query result is shown.
type savedView struct {
	viewState
	searchText string
	wrap       bool  // whether the list is wrapped
	width      int   // wrap width of the list
	pending    *Node // first document appended while hidden, not yet wrapped/collapsed
}

// queryRun is one evaluation of a query by the engine. Messages carry the
// run pointer; messages of a run other than m.query are stale.
type queryRun struct {
	query   string
	parser  engine.Parser
	nodes   *nodesParser // parser reading the original list, nil for previews
	preview bool
	pending []*Node // preview output, applied on success only
	cancel  chan struct{}
	msgs    chan tea.Msg
	stopped bool
	done    bool
	gotErr  bool
}

type queryResultMsg struct {
	run  *queryRun
	node *Node
}

type queryErrorMsg struct {
	run *queryRun
	err error
}

type queryDoneMsg struct {
	run      *queryRun
	exitCode int
}

func (m *model) doQuery(query string) tea.Cmd {
	query = strings.TrimSpace(query)
	if isIdentityQuery(query) {
		m.queryInput.SetValue("")
		return m.clearQuery()
	}
	m.stopQuery()
	return m.startQuery(query)
}

// isIdentityQuery reports whether query shows the input unchanged. The engine's
// fast path would return the original nodes themselves, which must never be
// linked into the result view.
func isIdentityQuery(query string) bool {
	for _, arg := range engine.SplitArgs(query) {
		if arg != "." && arg != "x" && arg != "this" {
			return false
		}
	}
	return true
}

func (m *model) startQuery(query string) tea.Cmd {
	// Swap to an empty result view right away: from now on the engine reads
	// the original list, so the UI must not mutate it.
	m.showResultView()

	nodes := newNodesParser(m.original.top, m.original.bottom, m.eof)
	run := &queryRun{
		query:  query,
		parser: nodes,
		nodes:  nodes,
		cancel: make(chan struct{}),
		msgs:   make(chan tea.Msg),
	}
	m.query = run
	m.runningQueries++
	go run.start()
	return waitQuery(run)
}

func (r *queryRun) start() {
	out := make(chan *Node)
	errCh := make(chan error)
	done := make(chan int, 1)
	args := engine.SplitArgs(r.query)
	go func() {
		if r.preview {
			done <- engine.StartPreview(r.parser, args, out, errCh, r.cancel)
		} else {
			done <- engine.Start(r.parser, args, out, errCh, r.cancel)
		}
	}()
	for {
		select {
		case node := <-out:
			r.msgs <- queryResultMsg{run: r, node: node}
		case err := <-errCh:
			r.msgs <- queryErrorMsg{run: r, err: err}
		case exitCode := <-done:
			r.msgs <- queryDoneMsg{run: r, exitCode: exitCode}
			return
		}
	}
}

// waitQuery reads the next message of the run. It is re-issued after every
// message, also for stale runs, until queryDoneMsg, so run goroutines never
// block forever.
func waitQuery(run *queryRun) tea.Cmd {
	return func() tea.Msg {
		return <-run.msgs
	}
}

// stopQuery cancels the running query, if any.
func (m *model) stopQuery() {
	m.query.stop()
}

func (r *queryRun) stop() {
	if r == nil || r.stopped {
		return
	}
	r.stopped = true
	if r.nodes != nil {
		r.nodes.stop()
	}
	close(r.cancel)
}

// appendOriginal attaches a streamed document to the hidden original list.
// It is neither wrapped nor collapsed, as engine goroutines may read it;
// that happens in restoreOriginal.
func (m *model) appendOriginal(node *Node) {
	o := m.original
	if o.top == nil {
		o.head, o.top = node, node
	} else {
		node.Index = -1 // To fix the statusbar path (to show .key instead of [0].key).
		o.bottom.Adjacent(node)
	}
	o.bottom = node
	o.totalLines = node.Bottom().LineNumber
	if o.pending == nil {
		o.pending = node
	}
	if m.query != nil && m.query.nodes != nil {
		m.query.nodes.publish(node)
	}
}

func (m *model) handleQueryResult(msg queryResultMsg) tea.Cmd {
	if msg.run.preview {
		if msg.run == m.livePreview {
			msg.run.pending = append(msg.run.pending, msg.node)
		}
	} else if msg.run == m.query {
		m.appendOutput(msg.node)
	}
	return waitQuery(msg.run)
}

func (m *model) handleQueryError(msg queryErrorMsg) tea.Cmd {
	msg.run.gotErr = true
	if msg.run == m.query {
		m.appendText(msg.err.Error(), true)
	}
	return waitQuery(msg.run)
}

func (m *model) handleQueryDone(msg queryDoneMsg) tea.Cmd {
	msg.run.done = true
	if msg.run.preview {
		if msg.run == m.livePreview {
			m.livePreview = nil
			m.applyPreview(msg.run, msg.exitCode)
		}
		return nil
	}
	m.runningQueries--
	if m.restoring && m.runningQueries == 0 {
		m.restoreOriginal()
		return nil
	}
	if msg.run == m.query && msg.exitCode != 0 && !msg.run.gotErr {
		m.appendText(fmt.Sprintf("exit(%d) is not allowed in interactive mode", msg.exitCode), true)
	}
	return nil
}

// clearQuery drops the result view. The original is restored once no engine
// goroutine reads it anymore, as restoring may re-wrap (mutate) it.
func (m *model) clearQuery() tea.Cmd {
	if m.original == nil {
		return nil
	}
	m.stopQuery()
	m.restoring = true
	if m.runningQueries == 0 {
		m.restoreOriginal()
		return nil
	}
	return m.spinner.Tick
}

func (m *model) restoreOriginal() {
	o := m.original
	m.cancelSearch()
	m.searchID++ // Drop search results in flight for the result list.
	m.viewState = o.viewState
	m.searchInput.SetValue(o.searchText)
	m.original = nil
	m.query = nil
	m.restoring = false
	if m.collapsed {
		for doc := o.pending; doc != nil; doc = nextDoc(doc) {
			doc.CollapseRecursively()
		}
	}
	// Wrap may have been toggled or the width changed while hidden.
	switch {
	case m.wrap && (!o.wrap || o.width != m.viewWidth()):
		Wrap(m.top, m.viewWidth())
	case m.wrap && o.pending != nil:
		Wrap(o.pending, m.viewWidth())
	case !m.wrap && o.wrap:
		DropWrapAll(m.top)
	}
	if m.head != nil && m.head.IsWrap() && !chunkAttached(m.head) {
		// The saved scroll position was a chunk, dropped by re-wrapping.
		m.head = m.head.Parent
		m.scrollIntoView()
	}
}

// chunkAttached reports whether chunk is still one of its string's chunks.
func chunkAttached(chunk *Node) bool {
	for it := chunk.Parent.Next; it != nil && it.IsWrap() && it.Parent == chunk.Parent; it = it.Next {
		if it == chunk {
			return true
		}
	}
	return false
}

// appendOutput attaches an engine output to the result view.
func (m *model) appendOutput(node *Node) {
	if node.Kind == Err {
		// Output of println/console.log.
		m.appendText(node.Value, false)
	} else {
		m.appendResult(node)
	}
}

// appendResult attaches an engine output document to the result view.
// Line numbers restart in every engine output, so they are shifted.
func (m *model) appendResult(node *Node) {
	offset := m.totalLines
	for it := node; it != nil; it = it.Next {
		it.LineNumber += offset
	}
	m.totalLines = node.Bottom().LineNumber
	m.appendNode(node)
}

// appendText attaches text lines (errors, println output) to the result view.
func (m *model) appendText(text string, isErr bool) {
	for _, line := range strings.Split(strings.Trim(text, "\n"), "\n") {
		m.totalLines++
		node := &Node{Kind: Err, Value: line, LineNumber: m.totalLines}
		if isErr {
			if m.queryErrors == nil {
				m.queryErrors = make(map[*Node]struct{})
			}
			m.queryErrors[node] = struct{}{}
		}
		m.appendNode(node)
	}
}

// isQueryError reports whether node is (a wrapped line of) a query error.
func (m *model) isQueryError(node *Node) bool {
	if m.queryErrors == nil {
		return false
	}
	if node.IsWrap() {
		node = node.Parent
	}
	_, ok := m.queryErrors[node]
	return ok
}

func (m *model) saveView() *savedView {
	return &savedView{
		viewState:  m.viewState,
		searchText: m.searchInput.Value(),
		wrap:       m.wrap,
		width:      m.viewWidth(),
	}
}

// showResultView sets the original aside, if not yet, and shows an empty
// result view.
func (m *model) showResultView() {
	if m.original == nil {
		m.original = m.saveView()
	}
	m.resetView()
	m.restoring = false
}

// resetView empties the displayed list.
func (m *model) resetView() {
	m.cancelSearch()
	m.searchID++ // Drop search results in flight for the previous list.
	m.viewState = viewState{search: newSearch()}
	m.searchInput.SetValue("")
}

// nodesParser feeds already parsed top-level documents to the engine
// without copying them. The UI goroutine appends documents to the list
// and calls publish; the engine goroutine walks the list only up to the
// last published document, so Next pointers are read after the mutex
// establishes happens-before.
type nodesParser struct {
	mu        sync.Mutex
	cond      *sync.Cond
	head      *Node // first document, nil until published
	last      *Node // last published document
	returned  *Node // last document returned by Parse
	recovered bool  // Parse skipped recovered text
	eof       bool
	cancelled bool
}

func newNodesParser(head, last *Node, eof bool) *nodesParser {
	p := &nodesParser{head: head, last: last, eof: eof}
	p.cond = sync.NewCond(&p.mu)
	return p
}

func (p *nodesParser) Parse() (*Node, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		if p.cancelled {
			return nil, io.EOF
		}
		if p.returned != p.last {
			if p.returned == nil {
				p.returned = p.head
			} else {
				p.returned = nextDoc(p.returned)
			}
			if p.returned.Kind == Err {
				// Recovered non-JSON text (e.g. HTTP headers), not an input for the engine.
				p.recovered = true
				continue
			}
			return p.returned, nil
		}
		if p.eof {
			return nil, io.EOF
		}
		p.cond.Wait()
	}
}

// More waits until a document after the last returned one is published,
// or the input ends. Recovered text is an error: the engine never sees it,
// so save() would drop it.
func (p *nodesParser) More() (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	for {
		if p.cancelled {
			return false, errCancelled
		}
		if p.recovered {
			return false, errRecovered
		}
		if p.returned != p.last {
			next := p.head
			if p.returned != nil {
				next = nextDoc(p.returned)
			}
			if next.Kind == Err {
				p.recovered = true
				return false, errRecovered
			}
			return true, nil
		}
		if p.eof {
			return false, nil
		}
		p.cond.Wait()
	}
}

var (
	errCancelled = errors.New("cancelled")
	errRecovered = errors.New("input contains text that is not JSON")
)

func (p *nodesParser) Recover() *Node {
	return nil
}

// publish makes doc (already linked after the previous last) available.
func (p *nodesParser) publish(doc *Node) {
	p.mu.Lock()
	if p.head == nil {
		p.head = doc
	}
	p.last = doc
	p.mu.Unlock()
	p.cond.Broadcast()
}

func (p *nodesParser) setEOF() {
	p.mu.Lock()
	p.eof = true
	p.mu.Unlock()
	p.cond.Broadcast()
}

// stop unblocks Parse, which then returns io.EOF.
func (p *nodesParser) stop() {
	p.mu.Lock()
	p.cancelled = true
	p.mu.Unlock()
	p.cond.Broadcast()
}

// nextDoc returns the top-level document following doc. Node.Bottom can't
// be used as it walks to the end of the whole list.
func nextDoc(doc *Node) *Node {
	if doc.End != nil {
		// End.Next is kept by Adjacent even if doc is collapsed.
		return doc.End.Next
	}
	if doc.ChunkEnd != nil {
		return doc.ChunkEnd.Next
	}
	return doc.Next
}

// previewMaxLines is the largest input evaluated on every keystroke.
const previewMaxLines = 10_000

const previewDelay = 100 * time.Millisecond

type previewTickMsg struct {
	seq uint64
}

type previewAction int

const (
	previewNone  previewAction = iota
	previewRun                 // evaluate the query
	previewClear               // show the original, like Enter on an identity query
)

// previewAction decides what the live preview of query does. Only input
// fully arrived and at most previewMaxLines is previewed on every keystroke.
// Queries calling save() or exit() are previewed too: they fail, and a
// failed preview is not applied.
func (m *model) previewAction(query string) previewAction {
	totalLines := m.totalLines
	if m.original != nil {
		totalLines = m.original.totalLines
	}
	if !m.eof || totalLines > previewMaxLines {
		return previewNone
	}
	if !isIdentityQuery(strings.TrimSpace(query)) {
		return previewRun
	}
	if m.original != nil {
		return previewClear
	}
	return previewNone
}

// schedulePreview debounces a live preview of the query being typed.
func (m *model) schedulePreview() tea.Cmd {
	m.cancelPreview()
	if m.previewAction(m.queryInput.Value()) == previewNone {
		return nil
	}
	seq := m.previewSeq
	return tea.Tick(previewDelay, func(time.Time) tea.Msg {
		return previewTickMsg{seq: seq}
	})
}

func (m *model) handlePreviewTick(msg previewTickMsg) tea.Cmd {
	if msg.seq != m.previewSeq || !m.queryInput.Focused() {
		return nil
	}
	query := strings.TrimSpace(m.queryInput.Value())
	switch m.previewAction(query) {
	case previewNone:
		return nil
	case previewClear:
		return m.clearQuery()
	}

	// The preview reads a snapshot, never the displayed nodes, so the UI
	// stays free to mutate them. Only small inputs are previewed.
	top := m.top
	if m.original != nil {
		top = m.original.top
	}
	run := &queryRun{
		query:   query,
		parser:  NewJsonParser(bytes.NewReader(snapshot(top)), false),
		preview: true,
		cancel:  make(chan struct{}),
		msgs:    make(chan tea.Msg),
	}
	m.livePreview = run
	go run.start()
	return waitQuery(run)
}

// cancelPreview stops the running live preview and drops a pending tick.
func (m *model) cancelPreview() {
	m.previewSeq++
	m.livePreview.stop()
	m.livePreview = nil
}

// applyPreview shows the preview output if it is a usable result:
// no error, and the first output is not null.
func (m *model) applyPreview(run *queryRun, exitCode int) {
	if run.gotErr || exitCode != 0 || len(run.pending) == 0 || run.pending[0].Kind == Null {
		return
	}
	m.stopQuery()
	m.showResultView()
	m.query = run
	for _, node := range run.pending {
		m.appendOutput(node)
	}
	run.pending = nil
}

// snapshot serializes the top-level documents starting at top as JSON.
func snapshot(top *Node) []byte {
	var b bytes.Buffer
	for doc := top; doc != nil; doc = nextDoc(doc) {
		if doc.Kind == Err {
			continue
		}
		for it := doc; it != nil; {
			if !it.IsWrap() {
				if it.Key != "" {
					b.WriteString(it.Key)
					b.WriteByte(':')
				}
				b.WriteString(it.Value)
				if it.Comma && it != doc.End {
					b.WriteByte(',')
				}
			}
			if it == doc.End || doc.End == nil {
				break
			}
			if it.IsCollapsed() {
				it = it.Collapsed
			} else {
				it = it.Next
			}
		}
		b.WriteByte('\n')
	}
	return b.Bytes()
}

const queryHistorySize = 100

func (m *model) addQueryHistory(query string) {
	query = strings.TrimSpace(query)
	if isIdentityQuery(query) {
		return
	}
	if n := len(m.queryHistory); n > 0 && m.queryHistory[n-1] == query {
		return
	}
	m.queryHistory = append(m.queryHistory, query)
	if len(m.queryHistory) > queryHistorySize {
		m.queryHistory = m.queryHistory[len(m.queryHistory)-queryHistorySize:]
	}
}

// queryHistoryPrev shows the previous history entry. Browsing does not
// trigger a live preview.
func (m *model) queryHistoryPrev() {
	if m.queryHistoryIndex == 0 {
		return
	}
	if m.queryHistoryIndex == len(m.queryHistory) {
		m.queryHistoryDraft = m.queryInput.Value()
	}
	m.queryHistoryIndex--
	m.setQueryInput(m.queryHistory[m.queryHistoryIndex])
}

func (m *model) queryHistoryNext() {
	if m.queryHistoryIndex >= len(m.queryHistory) {
		return
	}
	m.queryHistoryIndex++
	if m.queryHistoryIndex == len(m.queryHistory) {
		m.setQueryInput(m.queryHistoryDraft)
	} else {
		m.setQueryInput(m.queryHistory[m.queryHistoryIndex])
	}
}

func (m *model) setQueryInput(value string) {
	m.cancelPreview()
	m.queryInput.SetValue(value)
	m.queryInput.CursorEnd()
}

// queryInputView renders the query input with its JS syntax highlighted.
// It replaces the textinput view, which can style the text only as a whole.
func (m *model) queryInputView() string {
	query := m.queryInput.Value()
	value := []rune(query)
	pos := m.queryInput.Position()
	start, end := m.queryInputWindow(value, pos)

	// Token kind of every rune; whitespace is TokenPlain.
	kinds := make([]engine.TokenKind, len(value))
	r, b := 0, 0 // rune and byte index
	for _, t := range engine.Tokenize(query) {
		for ; r < len(value) && b < t.End; r++ {
			if b >= t.Start {
				kinds[r] = t.Kind
			}
			b += utf8.RuneLen(value[r])
		}
	}

	var v strings.Builder
	cursor := m.queryInput.Cursor
	showCursor := m.queryInput.Focused() && !cursor.Blink
	for i := start; i < end; {
		if i == pos && showCursor {
			cursor.SetChar(string(value[i]))
			v.WriteString(cursor.View())
			i++
			continue
		}
		j := i + 1
		for j < end && kinds[j] == kinds[i] && !(j == pos && showCursor) {
			j++
		}
		v.WriteString(tokenColor(kinds[i])(string(value[i:j])))
		i = j
	}
	if pos == len(value) && showCursor {
		cursor.SetChar(" ")
		v.WriteString(cursor.View())
	}
	return m.queryInput.Prompt + v.String()
}

func tokenColor(kind engine.TokenKind) theme.Color {
	t := theme.CurrentTheme
	switch kind {
	case engine.TokenString:
		return t.String
	case engine.TokenNumber:
		return t.Number
	case engine.TokenKeyword, engine.TokenBoolean:
		return t.Boolean
	case engine.TokenNull:
		return t.Null
	case engine.TokenProperty:
		return t.Key
	case engine.TokenPunct:
		return t.Syntax
	case engine.TokenComment:
		return t.Preview
	}
	return func(s string) string { return s }
}

// queryInputWindow returns the runes [start, end) of the query shown in the
// input, scrolling it like the textinput does: the window moves only as far
// as needed to keep the cursor visible.
func (m *model) queryInputWindow(value []rune, pos int) (start, end int) {
	width := m.queryInput.Width
	if width <= 0 || runewidth.StringWidth(string(value)) <= width {
		m.queryInputOffset = 0
		return 0, len(value)
	}
	start = min(m.queryInputOffset, pos)
	end, w := start, 0
	for end < len(value) && w+runewidth.RuneWidth(value[end]) <= width {
		w += runewidth.RuneWidth(value[end])
		end++
	}
	if pos >= end {
		// Align the window's right edge with the cursor.
		end = min(pos+1, len(value))
		start, w = end, 0
		for start > 0 && w+runewidth.RuneWidth(value[start-1]) <= width {
			w += runewidth.RuneWidth(value[start-1])
			start--
		}
	}
	m.queryInputOffset = start
	return start, end
}
