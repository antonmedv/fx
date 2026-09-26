package main

import (
	"fmt"
	"io"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/antonmedv/fx/internal/engine"
	. "github.com/antonmedv/fx/internal/jsonx"
)

// viewState is the part of the model describing the displayed node list.
type viewState struct {
	head, top, bottom *Node
	cursor            int
	totalLines        int
	locationHistory   []location
	locationIndex     int
	search            *search
}

// queryRun is one evaluation of a query by the engine. Messages carry the
// run pointer; messages of a run other than m.query are stale.
type queryRun struct {
	query   string
	parser  *nodesParser
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
	if query == "" || query == "." {
		return nil
	}
	m.stopQuery()
	return m.startQuery(query)
}

func (m *model) startQuery(query string) tea.Cmd {
	// Swap to an empty result view right away: from now on the engine reads
	// the original list, so the UI must not mutate it.
	if m.original == nil {
		m.original = m.saveView()
	}
	m.resetView()

	run := &queryRun{
		query:  query,
		parser: newNodesParser(m.original.top, m.original.bottom, m.eof),
		cancel: make(chan struct{}),
		msgs:   make(chan tea.Msg),
	}
	m.query = run
	go run.start()
	return waitQuery(run)
}

func (r *queryRun) start() {
	out := make(chan *Node)
	errCh := make(chan error)
	done := make(chan int, 1)
	go func() {
		done <- engine.Start(r.parser, []string{r.query}, out, errCh, r.cancel)
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
	run := m.query
	if run == nil || run.stopped {
		return
	}
	run.stopped = true
	run.parser.stop()
	close(run.cancel)
}

func (m *model) handleQueryResult(msg queryResultMsg) tea.Cmd {
	if msg.run == m.query {
		if msg.node.Kind == Err {
			// Output of println/console.log.
			m.appendText(msg.node.Value)
		} else {
			m.appendResult(msg.node)
		}
	}
	return waitQuery(msg.run)
}

func (m *model) handleQueryError(msg queryErrorMsg) tea.Cmd {
	msg.run.gotErr = true
	if msg.run == m.query {
		m.appendText(msg.err.Error())
	}
	return waitQuery(msg.run)
}

func (m *model) handleQueryDone(msg queryDoneMsg) tea.Cmd {
	msg.run.done = true
	if msg.run == m.query && msg.exitCode != 0 && !msg.run.gotErr {
		m.appendText(fmt.Sprintf("exit(%d) is not allowed in interactive mode", msg.exitCode))
	}
	return nil
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
func (m *model) appendText(text string) {
	for _, line := range strings.Split(text, "\n") {
		m.totalLines++
		m.appendNode(&Node{Kind: Err, Value: line, LineNumber: m.totalLines})
	}
}

func (m *model) saveView() *viewState {
	return &viewState{
		head:            m.head,
		top:             m.top,
		bottom:          m.bottom,
		cursor:          m.cursor,
		totalLines:      m.totalLines,
		locationHistory: m.locationHistory,
		locationIndex:   m.locationIndex,
		search:          m.search,
	}
}

// resetView empties the displayed list.
func (m *model) resetView() {
	m.cancelSearch()
	m.head, m.top, m.bottom = nil, nil, nil
	m.cursor = 0
	m.totalLines = 0
	m.locationHistory = nil
	m.locationIndex = 0
	m.search = newSearch()
	m.keysIndex = nil
	m.keysIndexNodes = nil
	m.fuzzyMatch = nil
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
			return p.returned, nil
		}
		if p.eof {
			return nil, io.EOF
		}
		p.cond.Wait()
	}
}

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
